package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"qms/queue-service/internal/models"
	"qms/queue-service/internal/store"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ticketNumberPad = 3

type Store struct {
	pool                *pgxpool.Pool
	noShowReturnToQueue bool
	priorityStreakLimit int
}

type Options struct {
	NoShowReturnToQueue bool
	PriorityStreakLimit int
}

func NewStore(pool *pgxpool.Pool, options Options) *Store {
	limit := options.PriorityStreakLimit
	if limit <= 0 {
		limit = 3
	}
	return &Store{
		pool:                pool,
		noShowReturnToQueue: options.NoShowReturnToQueue,
		priorityStreakLimit: limit,
	}
}

func (s *Store) CreateTicket(ctx context.Context, input store.CreateTicketInput) (models.Ticket, bool, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return models.Ticket{}, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	existing, found, err := findTicketByRequestID(ctx, tx, input.RequestID)
	if err != nil {
		return models.Ticket{}, false, err
	}
	if found {
		if err = tx.Commit(ctx); err != nil {
			return models.Ticket{}, false, err
		}
		return existing, false, nil
	}

	serviceCode, err := lookupServiceCode(ctx, tx, input)
	if err != nil {
		return models.Ticket{}, false, err
	}

	seq, err := nextTicketNumber(ctx, tx, input.BranchID, input.ServiceID)
	if err != nil {
		return models.Ticket{}, false, err
	}
	formattedNumber := fmt.Sprintf("%s-%0*d", serviceCode, ticketNumberPad, seq)

	ticketID := uuid.NewString()
	createdAt := input.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	var ticket models.Ticket
	row := tx.QueryRow(ctx, `
		INSERT INTO tickets (
			ticket_id, request_id, ticket_number, tenant_id, branch_id, service_id, area_id,
			status, channel, priority_class, created_at, phone_hash
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (request_id) DO NOTHING
		RETURNING ticket_id, ticket_number, status, created_at, request_id
	`, ticketID, input.RequestID, formattedNumber, input.TenantID, input.BranchID, input.ServiceID, nullIfEmpty(input.AreaID), models.StatusWaiting, input.Channel, input.PriorityClass, createdAt, hashPhone(input.Phone))

	if err = row.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &ticket.RequestID); err != nil {
		return models.Ticket{}, false, err
	}
	ticket.TenantID = input.TenantID
	ticket.BranchID = input.BranchID
	ticket.ServiceID = input.ServiceID
	ticket.AreaID = input.AreaID
	ticket.Phone = input.Phone

	if err = insertOutboxEvent(ctx, tx, input.TenantID, ticket); err != nil {
		return models.Ticket{}, false, err
	}

	if err = tx.Commit(ctx); err != nil {
		return models.Ticket{}, false, err
	}

	return ticket, true, nil
}

func (s *Store) GetTicket(ctx context.Context, tenantID, branchID, ticketID string) (models.Ticket, bool, error) {
	var ticket models.Ticket
	var calledAtNull sql.NullTime
	var counterIDNull sql.NullString
	var servedAtNull sql.NullTime
	var completedAtNull sql.NullTime
	var areaIDNull sql.NullString
	row := s.pool.QueryRow(ctx, `
		SELECT ticket_id, ticket_number, status, created_at, called_at, counter_id, served_at, completed_at, branch_id, service_id, area_id, tenant_id
		FROM tickets
		WHERE ticket_id = $1 AND tenant_id = $2 AND branch_id = $3
	`, ticketID, tenantID, branchID)
	if err := row.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &calledAtNull, &counterIDNull, &servedAtNull, &completedAtNull, &ticket.BranchID, &ticket.ServiceID, &areaIDNull, &ticket.TenantID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Ticket{}, false, store.ErrTicketNotFound
		}
		return models.Ticket{}, false, err
	}
	ticket.CalledAt = nullTimePtr(calledAtNull)
	ticket.CounterID = nullStringPtr(counterIDNull)
	ticket.ServedAt = nullTimePtr(servedAtNull)
	ticket.CompletedAt = nullTimePtr(completedAtNull)
	if areaIDNull.Valid {
		ticket.AreaID = areaIDNull.String
	}
	return ticket, true, nil
}

func (s *Store) ListQueue(ctx context.Context, tenantID, branchID, serviceID string) ([]models.Ticket, error) {
	query := `
		SELECT ticket_id, ticket_number, status, created_at, called_at, counter_id, served_at, completed_at, branch_id, service_id, area_id, tenant_id
		FROM tickets
		WHERE tenant_id = $1 AND branch_id = $2 AND status IN ('waiting','held')
	`
	args := []interface{}{tenantID, branchID}
	if serviceID != "" {
		query += " AND service_id = $3"
		args = append(args, serviceID)
	}
	query += " ORDER BY created_at ASC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []models.Ticket
	for rows.Next() {
		var ticket models.Ticket
		var calledAtNull sql.NullTime
		var counterIDNull sql.NullString
		var servedAtNull sql.NullTime
		var completedAtNull sql.NullTime
		var areaIDNull sql.NullString
		if err := rows.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &calledAtNull, &counterIDNull, &servedAtNull, &completedAtNull, &ticket.BranchID, &ticket.ServiceID, &areaIDNull, &ticket.TenantID); err != nil {
			return nil, err
		}
		ticket.CalledAt = nullTimePtr(calledAtNull)
		ticket.CounterID = nullStringPtr(counterIDNull)
		ticket.ServedAt = nullTimePtr(servedAtNull)
		ticket.CompletedAt = nullTimePtr(completedAtNull)
		if areaIDNull.Valid {
			ticket.AreaID = areaIDNull.String
		}
		tickets = append(tickets, ticket)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tickets, nil
}

func (s *Store) CallNext(ctx context.Context, input store.CallNextInput) (models.Ticket, bool, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return models.Ticket{}, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	existing, found, empty, err := findActionRequest(ctx, tx, "call_next", input.RequestID)
	if err != nil {
		return models.Ticket{}, false, err
	}
	if found {
		if err = tx.Commit(ctx); err != nil {
			return models.Ticket{}, false, err
		}
		if empty {
			return models.Ticket{}, false, store.ErrNoTicket
		}
		return existing, false, nil
	}

	if err = ensureServiceExists(ctx, tx, input); err != nil {
		return models.Ticket{}, false, err
	}

	allowed, err := counterAllowsService(ctx, tx, input.CounterID, input.ServiceID)
	if err != nil {
		return models.Ticket{}, false, err
	}
	if !allowed {
		return models.Ticket{}, false, store.ErrAccessDenied
	}

	status, err := getCounterStatus(ctx, tx, input.CounterID, input.BranchID)
	if err != nil {
		return models.Ticket{}, false, err
	}
	if !isCounterAvailable(status) {
		return models.Ticket{}, false, store.ErrCounterUnavailable
	}

	calledAt := input.CalledAt
	if calledAt.IsZero() {
		calledAt = time.Now().UTC()
	}

	state, err := lockRoutingState(ctx, tx, input.TenantID, input.BranchID, input.ServiceID)
	if err != nil {
		return models.Ticket{}, false, err
	}

	preferRegular := state.PriorityStreak >= s.priorityStreakLimit
	policy, _, err := getServicePolicy(ctx, tx, input.TenantID, input.BranchID, input.ServiceID)
	if err != nil {
		return models.Ticket{}, false, err
	}
	appointmentWindow := normalizeAppointmentWindow(policy.AppointmentWindowSize)
	appointmentTarget := appointmentTargetCount(policy.AppointmentRatioPercent, appointmentWindow)
	if state.TotalServed >= appointmentWindow {
		state.TotalServed = 0
		state.AppointmentServed = 0
	}
	preferAppointment := policy.AppointmentRatioPercent > 0 && state.AppointmentServed < appointmentTarget
	boostCutoff := time.Time{}
	if policy.AppointmentBoostMinutes > 0 {
		boostCutoff = calledAt.Add(time.Duration(policy.AppointmentBoostMinutes) * time.Minute)
	}

	ticket, priorityClass, isAppointment, err := updateNextTicket(ctx, tx, input, calledAt, preferRegular, preferAppointment, boostCutoff)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if err = insertActionRequest(ctx, tx, "call_next", input.RequestID, input.TenantID, input.BranchID, input.ServiceID, input.CounterID, ""); err != nil {
				return models.Ticket{}, false, err
			}
			if err = tx.Commit(ctx); err != nil {
				return models.Ticket{}, false, err
			}
			return models.Ticket{}, false, store.ErrNoTicket
		}
		return models.Ticket{}, false, err
	}

	ticket.RequestID = input.RequestID

	if err = insertActionRequest(ctx, tx, "call_next", input.RequestID, input.TenantID, input.BranchID, input.ServiceID, input.CounterID, ticket.TicketID); err != nil {
		return models.Ticket{}, false, err
	}

	if err = updateRoutingState(ctx, tx, input.TenantID, input.BranchID, input.ServiceID, state, priorityClass, isAppointment, appointmentWindow); err != nil {
		return models.Ticket{}, false, err
	}

	if err = insertOutboxEventCalled(ctx, tx, input.TenantID, ticket); err != nil {
		return models.Ticket{}, false, err
	}

	if err = tx.Commit(ctx); err != nil {
		return models.Ticket{}, false, err
	}

	return ticket, true, nil
}

func (s *Store) AutoNoShow(ctx context.Context, grace time.Duration, batchSize int) (int, error) {
	if grace <= 0 {
		return 0, nil
	}
	if batchSize <= 0 {
		batchSize = 100
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	cutoff := time.Now().UTC().Add(-grace)
	rows, err := tx.Query(ctx, `
		SELECT ticket_id, ticket_number, status, created_at, called_at, counter_id, tenant_id, branch_id, service_id, area_id
		FROM tickets
		WHERE status = 'called' AND called_at <= $1
		ORDER BY called_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $2
	`, cutoff, batchSize)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type noShowItem struct {
		ticket    models.Ticket
		tenantID  string
		branchID  string
		serviceID string
	}
	var items []noShowItem
	var ids []string
	for rows.Next() {
		var ticket models.Ticket
		var calledAtNull sql.NullTime
		var counterIDNull sql.NullString
		var areaIDNull sql.NullString
		var tenantID string
		var branchID string
		var serviceID string
		if err := rows.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &calledAtNull, &counterIDNull, &tenantID, &branchID, &serviceID, &areaIDNull); err != nil {
			return 0, err
		}
		ticket.CalledAt = nullTimePtr(calledAtNull)
		ticket.CounterID = nullStringPtr(counterIDNull)
		if areaIDNull.Valid {
			ticket.AreaID = areaIDNull.String
		}
		items = append(items, noShowItem{ticket: ticket, tenantID: tenantID, branchID: branchID, serviceID: serviceID})
		ids = append(ids, ticket.TicketID)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		if err = tx.Commit(ctx); err != nil {
			return 0, err
		}
		return 0, nil
	}

	processed := 0
	for i := range items {
		policy, found, err := getServicePolicy(ctx, tx, items[i].tenantID, items[i].branchID, items[i].serviceID)
		if err != nil {
			return 0, err
		}
		effectiveGrace := grace
		returnToQueue := s.noShowReturnToQueue
		if found {
			effectiveGrace = time.Duration(policy.NoShowGraceSeconds) * time.Second
			returnToQueue = policy.ReturnToQueue
		}
		calledAt := items[i].ticket.CalledAt
		if calledAt == nil || time.Since(*calledAt) < effectiveGrace {
			continue
		}

		if returnToQueue {
			_, err = tx.Exec(ctx, `
				UPDATE tickets
				SET status = 'waiting',
					counter_id = NULL,
					called_at = NULL,
					returned = TRUE
				WHERE ticket_id = $1
			`, items[i].ticket.TicketID)
		} else {
			_, err = tx.Exec(ctx, `
				UPDATE tickets
				SET status = 'no_show'
				WHERE ticket_id = $1
			`, items[i].ticket.TicketID)
		}
		if err != nil {
			return 0, err
		}

		items[i].ticket.RequestID = ""
		items[i].ticket.TenantID = items[i].tenantID
		items[i].ticket.BranchID = items[i].branchID
		items[i].ticket.ServiceID = items[i].serviceID
		if returnToQueue {
			items[i].ticket.Status = models.StatusWaiting
		} else {
			items[i].ticket.Status = models.StatusNoShow
		}
		if err = insertOutboxEventNoShow(ctx, tx, items[i].tenantID, items[i].ticket, returnToQueue); err != nil {
			return 0, err
		}
		processed++
	}

	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}

	return processed, nil
}

func (s *Store) applyNoShow(ctx context.Context, input store.TicketActionInput, returnToQueue bool) (models.Ticket, bool, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return models.Ticket{}, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	existing, found, empty, err := findActionRequest(ctx, tx, "no_show", input.RequestID)
	if err != nil {
		return models.Ticket{}, false, err
	}
	if found {
		if err = tx.Commit(ctx); err != nil {
			return models.Ticket{}, false, err
		}
		if empty {
			return models.Ticket{}, false, store.ErrInvalidState
		}
		return existing, false, nil
	}

	var ticket models.Ticket
	var calledAtNull sql.NullTime
	var counterIDNull sql.NullString
	var areaIDNull sql.NullString
	query := `
		UPDATE tickets
		SET status = 'no_show'
		WHERE ticket_id = $1 AND tenant_id = $2 AND branch_id = $3 AND status = 'called'
		RETURNING ticket_id, ticket_number, status, created_at, called_at, counter_id, service_id, branch_id, area_id, tenant_id
	`
	if returnToQueue {
		query = `
			UPDATE tickets
			SET status = 'waiting',
				counter_id = NULL,
				called_at = NULL,
				returned = TRUE
			WHERE ticket_id = $1 AND tenant_id = $2 AND branch_id = $3 AND status = 'called'
			RETURNING ticket_id, ticket_number, status, created_at, called_at, counter_id, service_id, branch_id, area_id, tenant_id
		`
	}

	row := tx.QueryRow(ctx, query, input.TicketID, input.TenantID, input.BranchID)
	if err = row.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &calledAtNull, &counterIDNull, &ticket.ServiceID, &ticket.BranchID, &areaIDNull, &ticket.TenantID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			state, _, exists, err := loadTicketState(ctx, tx, input.TicketID, input.TenantID, input.BranchID)
			if err != nil {
				return models.Ticket{}, false, err
			}
			if !exists {
				return models.Ticket{}, false, store.ErrTicketNotFound
			}
			if state != models.StatusCalled {
				return models.Ticket{}, false, store.ErrInvalidState
			}
			return models.Ticket{}, false, store.ErrInvalidState
		}
		return models.Ticket{}, false, err
	}

	ticket.RequestID = input.RequestID
	ticket.CalledAt = nullTimePtr(calledAtNull)
	ticket.CounterID = nullStringPtr(counterIDNull)
	if areaIDNull.Valid {
		ticket.AreaID = areaIDNull.String
	}

	if err = insertActionRequest(ctx, tx, "no_show", input.RequestID, input.TenantID, input.BranchID, input.ServiceID, input.CounterID, ticket.TicketID); err != nil {
		return models.Ticket{}, false, err
	}

	if err = insertOutboxEventNoShow(ctx, tx, input.TenantID, ticket, returnToQueue); err != nil {
		return models.Ticket{}, false, err
	}

	if err = tx.Commit(ctx); err != nil {
		return models.Ticket{}, false, err
	}

	return ticket, true, nil
}

func (s *Store) SnapshotTickets(ctx context.Context, tenantID, branchID, serviceID string) ([]models.Ticket, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ticket_id, ticket_number, status, created_at, called_at, counter_id, served_at, completed_at, branch_id, service_id, area_id, tenant_id
		FROM tickets
		WHERE tenant_id = $1 AND branch_id = $2 AND service_id = $3
			AND status IN ('waiting', 'called', 'serving')
		ORDER BY created_at ASC
	`, tenantID, branchID, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []models.Ticket
	for rows.Next() {
		var ticket models.Ticket
		var calledAtNull sql.NullTime
		var counterIDNull sql.NullString
		var servedAtNull sql.NullTime
		var completedAtNull sql.NullTime
		var areaIDNull sql.NullString
		if err := rows.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &calledAtNull, &counterIDNull, &servedAtNull, &completedAtNull, &ticket.BranchID, &ticket.ServiceID, &areaIDNull, &ticket.TenantID); err != nil {
			return nil, err
		}
		ticket.CalledAt = nullTimePtr(calledAtNull)
		ticket.CounterID = nullStringPtr(counterIDNull)
		ticket.ServedAt = nullTimePtr(servedAtNull)
		ticket.CompletedAt = nullTimePtr(completedAtNull)
		if areaIDNull.Valid {
			ticket.AreaID = areaIDNull.String
		}
		tickets = append(tickets, ticket)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tickets, nil
}

func (s *Store) GetActiveTicket(ctx context.Context, tenantID, branchID, counterID string) (models.Ticket, bool, error) {
	var ticket models.Ticket
	var calledAtNull sql.NullTime
	var counterIDNull sql.NullString
	var servedAtNull sql.NullTime
	var completedAtNull sql.NullTime
	var areaIDNull sql.NullString
	row := s.pool.QueryRow(ctx, `
		SELECT ticket_id, ticket_number, status, created_at, called_at, counter_id, served_at, completed_at, branch_id, service_id, area_id, tenant_id
		FROM tickets
		WHERE tenant_id = $1 AND branch_id = $2 AND counter_id = $3
			AND status IN ('called', 'serving')
		ORDER BY called_at DESC
		LIMIT 1
	`, tenantID, branchID, counterID)
	if err := row.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &calledAtNull, &counterIDNull, &servedAtNull, &completedAtNull, &ticket.BranchID, &ticket.ServiceID, &areaIDNull, &ticket.TenantID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Ticket{}, false, nil
		}
		return models.Ticket{}, false, err
	}
	ticket.CalledAt = nullTimePtr(calledAtNull)
	ticket.CounterID = nullStringPtr(counterIDNull)
	ticket.ServedAt = nullTimePtr(servedAtNull)
	ticket.CompletedAt = nullTimePtr(completedAtNull)
	if areaIDNull.Valid {
		ticket.AreaID = areaIDNull.String
	}
	return ticket, true, nil
}

func (s *Store) ListOutboxEvents(ctx context.Context, tenantID string, after time.Time, limit int) ([]store.OutboxEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT event_id, tenant_id, type, payload_json, created_at
		FROM outbox_events
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	if !after.IsZero() {
		query += " AND created_at > $2"
		args = append(args, after)
		query += " ORDER BY created_at ASC LIMIT $3"
		args = append(args, limit)
	} else {
		query += " ORDER BY created_at ASC LIMIT $2"
		args = append(args, limit)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []store.OutboxEvent
	for rows.Next() {
		var event store.OutboxEvent
		if err := rows.Scan(&event.EventID, &event.TenantID, &event.Type, &event.Payload, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *Store) ListTicketEvents(ctx context.Context, tenantID, ticketID string) ([]store.TicketEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT e.ticket_id, e.ticket_seq, e.type, e.payload, e.created_at, e.prev_hash, e.hash
		FROM ticket_events e
		JOIN tickets t ON t.ticket_id = e.ticket_id
		WHERE t.tenant_id = $1 AND e.ticket_id = $2
		ORDER BY e.ticket_seq ASC
	`, tenantID, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []store.TicketEvent
	for rows.Next() {
		var event store.TicketEvent
		if err := rows.Scan(&event.TicketID, &event.TicketSeq, &event.Type, &event.Payload, &event.CreatedAt, &event.PrevHash, &event.Hash); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *Store) ListCounters(ctx context.Context, tenantID, branchID string) ([]models.Counter, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.counter_id, c.branch_id, c.name, c.status
		FROM counters c
		JOIN branches b ON b.branch_id = c.branch_id
		WHERE b.tenant_id = $1 AND c.branch_id = $2
		ORDER BY c.name ASC
	`, tenantID, branchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counters []models.Counter
	for rows.Next() {
		var counter models.Counter
		if err := rows.Scan(&counter.CounterID, &counter.BranchID, &counter.Name, &counter.Status); err != nil {
			return nil, err
		}
		counters = append(counters, counter)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counters, nil
}

func (s *Store) UpdateCounterStatus(ctx context.Context, tenantID, branchID, counterID, status string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE counters c
		SET status = $1
		FROM branches b
		WHERE c.counter_id = $2 AND c.branch_id = $3 AND b.branch_id = c.branch_id AND b.tenant_id = $4
	`, status, counterID, branchID, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return store.ErrCounterNotFound
	}
	return nil
}

func (s *Store) ListServices(ctx context.Context, tenantID, branchID string) ([]models.Service, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.service_id, s.branch_id, s.name, s.code, s.sla_minutes, s.priority_policy, COALESCE(s.hours_json::text, '')
		FROM services s
		JOIN branches b ON b.branch_id = s.branch_id
		WHERE b.tenant_id = $1 AND s.branch_id = $2 AND s.active = TRUE
		ORDER BY s.name ASC
	`, tenantID, branchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var svc models.Service
		if err := rows.Scan(&svc.ServiceID, &svc.BranchID, &svc.Name, &svc.Code, &svc.SLAMinutes, &svc.PriorityPolicy, &svc.HoursJSON); err != nil {
			return nil, err
		}
		services = append(services, svc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return services, nil
}

func (s *Store) CheckInAppointment(ctx context.Context, requestID, tenantID, branchID, appointmentID string) (models.Ticket, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return models.Ticket{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	existing, found, err := findTicketByRequestID(ctx, tx, requestID)
	if err != nil {
		return models.Ticket{}, err
	}
	if found {
		if err = tx.Commit(ctx); err != nil {
			return models.Ticket{}, err
		}
		return existing, nil
	}

	var serviceID string
	var scheduledDate time.Time
	row := tx.QueryRow(ctx, `
		SELECT service_id, scheduled_at::date
		FROM appointments
		WHERE appointment_id = $1 AND tenant_id = $2 AND branch_id = $3 AND status = 'scheduled'
	`, appointmentID, tenantID, branchID)
	if err = row.Scan(&serviceID, &scheduledDate); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Ticket{}, store.ErrTicketNotFound
		}
		return models.Ticket{}, err
	}

	var holidayExists bool
	row = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM holidays
			WHERE tenant_id = $1 AND branch_id = $2 AND date = $3
		)
	`, tenantID, branchID, scheduledDate)
	if err = row.Scan(&holidayExists); err != nil {
		return models.Ticket{}, err
	}
	if holidayExists {
		return models.Ticket{}, store.ErrHolidayClosed
	}

	_, err = tx.Exec(ctx, `
		UPDATE appointments
		SET status = 'checked_in'
		WHERE appointment_id = $1
	`, appointmentID)
	if err != nil {
		return models.Ticket{}, err
	}

	serviceCode, err := lookupServiceCode(ctx, tx, store.CreateTicketInput{
		TenantID:  tenantID,
		BranchID:  branchID,
		ServiceID: serviceID,
	})
	if err != nil {
		return models.Ticket{}, err
	}
	seq, err := nextTicketNumber(ctx, tx, branchID, serviceID)
	if err != nil {
		return models.Ticket{}, err
	}
	formattedNumber := fmt.Sprintf("%s-%0*d", serviceCode, ticketNumberPad, seq)
	ticketID := uuid.NewString()
	createdAt := time.Now().UTC()

	var ticket models.Ticket
	row = tx.QueryRow(ctx, `
		INSERT INTO tickets (
			ticket_id, request_id, ticket_number, tenant_id, branch_id, service_id, status, channel, priority_class, created_at, appointment_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING ticket_id, ticket_number, status, created_at, request_id
	`, ticketID, requestID, formattedNumber, tenantID, branchID, serviceID, models.StatusWaiting, "kiosk", "regular", createdAt, appointmentID)
	if err = row.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &ticket.RequestID); err != nil {
		return models.Ticket{}, err
	}
	ticket.TenantID = tenantID
	ticket.BranchID = branchID
	ticket.ServiceID = serviceID

	if err = insertOutboxEvent(ctx, tx, tenantID, ticket); err != nil {
		return models.Ticket{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return models.Ticket{}, err
	}
	return ticket, nil
}

func (s *Store) StartServing(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	return s.updateTicketStatus(ctx, input, "start_serving", models.StatusCalled, models.StatusServing, "ticket.serving", "served_at", true)
}

func (s *Store) CompleteTicket(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	return s.updateTicketStatus(ctx, input, "complete", models.StatusServing, models.StatusDone, "ticket.done", "completed_at", false)
}

func (s *Store) CancelTicket(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	return s.updateTicketStatus(ctx, input, "cancel", models.StatusWaiting, models.StatusCancelled, "ticket.cancelled", "", false)
}

func (s *Store) HoldTicket(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	return s.updateTicketStatus(ctx, input, "hold", models.StatusWaiting, models.StatusHeld, "ticket.held", "", false)
}

func (s *Store) UnholdTicket(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	return s.updateTicketStatus(ctx, input, "unhold", models.StatusHeld, models.StatusWaiting, "ticket.unheld", "", false)
}

func (s *Store) NoShowTicket(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	return s.applyNoShow(ctx, input, input.ReturnToQueue)
}

func (s *Store) RecallTicket(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	return s.emitTicketEvent(ctx, input, "recall", models.StatusCalled, "ticket.recalled")
}

func (s *Store) TransferTicket(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return models.Ticket{}, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	existing, found, empty, err := findActionRequest(ctx, tx, "transfer", input.RequestID)
	if err != nil {
		return models.Ticket{}, false, err
	}
	if found {
		if err = tx.Commit(ctx); err != nil {
			return models.Ticket{}, false, err
		}
		if empty {
			return models.Ticket{}, false, store.ErrInvalidState
		}
		return existing, false, nil
	}

	if err = ensureTargetServiceExists(ctx, tx, input); err != nil {
		return models.Ticket{}, false, err
	}

	var ticket models.Ticket
	var fromServiceID string
	var areaIDNull sql.NullString
	row := tx.QueryRow(ctx, `
		WITH current AS (
			SELECT service_id
			FROM tickets
			WHERE ticket_id = $1 AND tenant_id = $2 AND branch_id = $3
			FOR UPDATE
		), updated AS (
			UPDATE tickets
			SET status = 'waiting',
				service_id = $4,
				counter_id = NULL
			WHERE ticket_id = $1 AND tenant_id = $2 AND branch_id = $3 AND status IN ('waiting','called','serving')
			RETURNING ticket_id, ticket_number, status, created_at, area_id
		)
		SELECT updated.ticket_id, updated.ticket_number, updated.status, updated.created_at, updated.area_id, current.service_id
		FROM updated
		JOIN current ON TRUE
	`, input.TicketID, input.TenantID, input.BranchID, input.ServiceID)

	if err = row.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &areaIDNull, &fromServiceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Ticket{}, false, store.ErrInvalidState
		}
		return models.Ticket{}, false, err
	}
	if areaIDNull.Valid {
		ticket.AreaID = areaIDNull.String
	}

	ticket.RequestID = input.RequestID
	ticket.TenantID = input.TenantID
	ticket.BranchID = input.BranchID
	ticket.ServiceID = input.ServiceID

	if err = insertActionRequest(ctx, tx, "transfer", input.RequestID, input.TenantID, input.BranchID, input.ServiceID, input.CounterID, ticket.TicketID); err != nil {
		return models.Ticket{}, false, err
	}

	if err = insertOutboxEventTransfer(ctx, tx, input.TenantID, ticket, fromServiceID, input.ServiceID, input.Reason); err != nil {
		return models.Ticket{}, false, err
	}

	if err = tx.Commit(ctx); err != nil {
		return models.Ticket{}, false, err
	}

	return ticket, true, nil
}

func lookupServiceCode(ctx context.Context, tx pgx.Tx, input store.CreateTicketInput) (string, error) {
	var code string
	row := tx.QueryRow(ctx, `
		SELECT s.code
		FROM services s
		JOIN branches b ON b.branch_id = s.branch_id
		WHERE s.service_id = $1 AND s.branch_id = $2 AND b.tenant_id = $3 AND s.active = TRUE
	`, input.ServiceID, input.BranchID, input.TenantID)
	if err := row.Scan(&code); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", store.ErrServiceNotFound
		}
		return "", err
	}
	return code, nil
}

func ensureServiceExists(ctx context.Context, tx pgx.Tx, input store.CallNextInput) error {
	var serviceID string
	row := tx.QueryRow(ctx, `
		SELECT s.service_id
		FROM services s
		JOIN branches b ON b.branch_id = s.branch_id
		WHERE s.service_id = $1 AND s.branch_id = $2 AND b.tenant_id = $3 AND s.active = TRUE
	`, input.ServiceID, input.BranchID, input.TenantID)
	if err := row.Scan(&serviceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.ErrServiceNotFound
		}
		return err
	}
	return nil
}

type routingState struct {
	PriorityStreak    int
	AppointmentServed int
	TotalServed       int
}

func lockRoutingState(ctx context.Context, tx pgx.Tx, tenantID, branchID, serviceID string) (routingState, error) {
	_, err := tx.Exec(ctx, `
		INSERT INTO service_routing_state (tenant_id, branch_id, service_id, priority_streak, appointment_served, total_served)
		VALUES ($1, $2, $3, 0, 0, 0)
		ON CONFLICT (tenant_id, branch_id, service_id) DO NOTHING
	`, tenantID, branchID, serviceID)
	if err != nil {
		return routingState{}, err
	}

	var state routingState
	row := tx.QueryRow(ctx, `
		SELECT priority_streak, appointment_served, total_served
		FROM service_routing_state
		WHERE tenant_id = $1 AND branch_id = $2 AND service_id = $3
		FOR UPDATE
	`, tenantID, branchID, serviceID)
	if err := row.Scan(&state.PriorityStreak, &state.AppointmentServed, &state.TotalServed); err != nil {
		return routingState{}, err
	}
	return state, nil
}

func updateRoutingState(ctx context.Context, tx pgx.Tx, tenantID, branchID, serviceID string, state routingState, priorityClass string, isAppointment bool, appointmentWindow int) error {
	newStreak := state.PriorityStreak
	if priorityClass == "regular" {
		newStreak = 0
	} else if priorityClass != "" {
		newStreak = state.PriorityStreak + 1
	}

	totalServed := state.TotalServed + 1
	appointmentServed := state.AppointmentServed
	if isAppointment {
		appointmentServed++
	}
	if appointmentWindow > 0 && totalServed >= appointmentWindow {
		totalServed = 0
		appointmentServed = 0
	}
	_, err := tx.Exec(ctx, `
		UPDATE service_routing_state
		SET priority_streak = $1,
			appointment_served = $2,
			total_served = $3
		WHERE tenant_id = $4 AND branch_id = $5 AND service_id = $6
	`, newStreak, appointmentServed, totalServed, tenantID, branchID, serviceID)
	return err
}

func updateNextTicket(ctx context.Context, tx pgx.Tx, input store.CallNextInput, calledAt time.Time, preferRegular, preferAppointment bool, boostCutoff time.Time) (models.Ticket, string, bool, error) {
	if !boostCutoff.IsZero() {
		ticket, class, err := updateNextAppointmentTicket(ctx, tx, input, calledAt, "", boostCutoff, preferRegular)
		if err == nil {
			return ticket, class, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return models.Ticket{}, "", false, err
		}
	}

	if preferAppointment {
		ticket, class, err := updateNextAppointmentTicket(ctx, tx, input, calledAt, "", time.Time{}, preferRegular)
		if err == nil {
			return ticket, class, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return models.Ticket{}, "", false, err
		}
	}

	ticket, class, err := updateNextWalkinTicket(ctx, tx, input, calledAt, preferRegular)
	if err == nil {
		return ticket, class, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return models.Ticket{}, "", false, err
	}

	ticket, class, err = updateNextAppointmentTicket(ctx, tx, input, calledAt, "", time.Time{}, preferRegular)
	if err == nil {
		return ticket, class, true, nil
	}
	return models.Ticket{}, "", false, err
}

func updateNextAppointmentTicket(ctx context.Context, tx pgx.Tx, input store.CallNextInput, calledAt time.Time, filter string, cutoff time.Time, preferRegular bool) (models.Ticket, string, error) {
	if preferRegular {
		ticket, class, err := updateNextAppointmentTicketWithFilter(ctx, tx, input, calledAt, "AND t.priority_class = 'regular' "+filter, cutoff)
		if err == nil || !errors.Is(err, pgx.ErrNoRows) {
			return ticket, class, err
		}
		return updateNextAppointmentTicketWithFilter(ctx, tx, input, calledAt, filter, cutoff)
	}

	ticket, class, err := updateNextAppointmentTicketWithFilter(ctx, tx, input, calledAt, "AND t.priority_class <> 'regular' "+filter, cutoff)
	if err == nil || !errors.Is(err, pgx.ErrNoRows) {
		return ticket, class, err
	}
	return updateNextAppointmentTicketWithFilter(ctx, tx, input, calledAt, filter, cutoff)
}

func updateNextAppointmentTicketWithFilter(ctx context.Context, tx pgx.Tx, input store.CallNextInput, calledAt time.Time, filter string, cutoff time.Time) (models.Ticket, string, error) {
	var ticket models.Ticket
	var calledAtNull sql.NullTime
	var counterIDNull sql.NullString
	var priorityClass sql.NullString
	var areaIDNull sql.NullString

	args := []interface{}{input.TenantID, input.BranchID, input.ServiceID, input.CounterID, calledAt}
	cutoffFilter := ""
	if !cutoff.IsZero() {
		cutoffFilter = " AND a.scheduled_at <= $6"
		args = append(args, cutoff)
	}

	query := `
		WITH next_ticket AS (
			SELECT t.ticket_id
			FROM tickets t
			JOIN appointments a ON a.appointment_id = t.appointment_id
			WHERE t.tenant_id = $1 AND t.branch_id = $2 AND t.service_id = $3 AND t.status = 'waiting'
				AND t.appointment_id IS NOT NULL ` + filter + cutoffFilter + `
			ORDER BY a.scheduled_at ASC, t.created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE tickets
		SET status = 'called',
			counter_id = $4,
			called_at = $5
		FROM next_ticket
		WHERE tickets.ticket_id = next_ticket.ticket_id
		RETURNING tickets.ticket_id, tickets.ticket_number, tickets.status, tickets.created_at, tickets.called_at, tickets.counter_id, tickets.priority_class, tickets.branch_id, tickets.service_id, tickets.area_id, tickets.tenant_id
	`
	row := tx.QueryRow(ctx, query, args...)
	if err := row.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &calledAtNull, &counterIDNull, &priorityClass, &ticket.BranchID, &ticket.ServiceID, &areaIDNull, &ticket.TenantID); err != nil {
		return models.Ticket{}, "", err
	}
	ticket.CalledAt = nullTimePtr(calledAtNull)
	ticket.CounterID = nullStringPtr(counterIDNull)
	if areaIDNull.Valid {
		ticket.AreaID = areaIDNull.String
	}
	if priorityClass.Valid {
		return ticket, priorityClass.String, nil
	}
	return ticket, "", nil
}

func updateNextWalkinTicket(ctx context.Context, tx pgx.Tx, input store.CallNextInput, calledAt time.Time, preferRegular bool) (models.Ticket, string, error) {
	if preferRegular {
		ticket, class, err := updateNextWalkinTicketWithFilter(ctx, tx, input, calledAt, "AND priority_class = 'regular'")
		if err == nil || !errors.Is(err, pgx.ErrNoRows) {
			return ticket, class, err
		}
		return updateNextWalkinTicketWithFilter(ctx, tx, input, calledAt, "")
	}

	ticket, class, err := updateNextWalkinTicketWithFilter(ctx, tx, input, calledAt, "AND priority_class <> 'regular'")
	if err == nil || !errors.Is(err, pgx.ErrNoRows) {
		return ticket, class, err
	}
	return updateNextWalkinTicketWithFilter(ctx, tx, input, calledAt, "")
}

func updateNextWalkinTicketWithFilter(ctx context.Context, tx pgx.Tx, input store.CallNextInput, calledAt time.Time, filter string) (models.Ticket, string, error) {
	var ticket models.Ticket
	var calledAtNull sql.NullTime
	var counterIDNull sql.NullString
	var priorityClass sql.NullString
	var areaIDNull sql.NullString
	query := `
		WITH next_ticket AS (
			SELECT ticket_id
			FROM tickets
			WHERE tenant_id = $1 AND branch_id = $2 AND service_id = $3 AND status = 'waiting'
				AND appointment_id IS NULL ` + filter + `
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE tickets
		SET status = 'called',
			counter_id = $4,
			called_at = $5
		FROM next_ticket
		WHERE tickets.ticket_id = next_ticket.ticket_id
		RETURNING tickets.ticket_id, tickets.ticket_number, tickets.status, tickets.created_at, tickets.called_at, tickets.counter_id, tickets.priority_class, tickets.branch_id, tickets.service_id, tickets.area_id, tickets.tenant_id
	`
	row := tx.QueryRow(ctx, query, input.TenantID, input.BranchID, input.ServiceID, input.CounterID, calledAt)
	if err := row.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &calledAtNull, &counterIDNull, &priorityClass, &ticket.BranchID, &ticket.ServiceID, &areaIDNull, &ticket.TenantID); err != nil {
		return models.Ticket{}, "", err
	}
	ticket.CalledAt = nullTimePtr(calledAtNull)
	ticket.CounterID = nullStringPtr(counterIDNull)
	if areaIDNull.Valid {
		ticket.AreaID = areaIDNull.String
	}
	if priorityClass.Valid {
		return ticket, priorityClass.String, nil
	}
	return ticket, "", nil
}

func ensureTargetServiceExists(ctx context.Context, tx pgx.Tx, input store.TicketActionInput) error {
	var serviceID string
	row := tx.QueryRow(ctx, `
		SELECT s.service_id
		FROM services s
		JOIN branches b ON b.branch_id = s.branch_id
		WHERE s.service_id = $1 AND s.branch_id = $2 AND b.tenant_id = $3 AND s.active = TRUE
	`, input.ServiceID, input.BranchID, input.TenantID)
	if err := row.Scan(&serviceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.ErrServiceNotFound
		}
		return err
	}
	return nil
}

func counterAllowsService(ctx context.Context, tx pgx.Tx, counterID, serviceID string) (bool, error) {
	var count int
	row := tx.QueryRow(ctx, `
		SELECT COUNT(1)
		FROM counter_services
		WHERE counter_id = $1
	`, counterID)
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	if count == 0 {
		return true, nil
	}
	row = tx.QueryRow(ctx, `
		SELECT COUNT(1)
		FROM counter_services
		WHERE counter_id = $1 AND service_id = $2
	`, counterID, serviceID)
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func getCounterStatus(ctx context.Context, tx pgx.Tx, counterID, branchID string) (string, error) {
	var status string
	row := tx.QueryRow(ctx, `
		SELECT status
		FROM counters
		WHERE counter_id = $1 AND branch_id = $2
	`, counterID, branchID)
	if err := row.Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", store.ErrCounterNotFound
		}
		return "", err
	}
	return status, nil
}

func isCounterAvailable(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "active", "available", "busy":
		return true
	default:
		return false
	}
}

type servicePolicy struct {
	NoShowGraceSeconds      int
	ReturnToQueue           bool
	AppointmentRatioPercent int
	AppointmentWindowSize   int
	AppointmentBoostMinutes int
}

func getServicePolicy(ctx context.Context, tx pgx.Tx, tenantID, branchID, serviceID string) (servicePolicy, bool, error) {
	var policy servicePolicy
	row := tx.QueryRow(ctx, `
		SELECT no_show_grace_seconds, return_to_queue, appointment_ratio_percent, appointment_window_size, appointment_boost_minutes
		FROM service_policies
		WHERE tenant_id = $1 AND branch_id = $2 AND service_id = $3
	`, tenantID, branchID, serviceID)
	if err := row.Scan(&policy.NoShowGraceSeconds, &policy.ReturnToQueue, &policy.AppointmentRatioPercent, &policy.AppointmentWindowSize, &policy.AppointmentBoostMinutes); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return servicePolicy{}, false, nil
		}
		return servicePolicy{}, false, err
	}
	return policy, true, nil
}

func normalizeAppointmentWindow(value int) int {
	if value <= 0 {
		return 10
	}
	return value
}

func appointmentTargetCount(ratioPercent int, window int) int {
	if ratioPercent <= 0 || window <= 0 {
		return 0
	}
	return int(math.Round(float64(ratioPercent) * float64(window) / 100.0))
}

func nextTicketNumber(ctx context.Context, tx pgx.Tx, branchID, serviceID string) (int64, error) {
	var next int64
	row := tx.QueryRow(ctx, `
		INSERT INTO ticket_sequences (branch_id, service_id, next_number)
		VALUES ($1, $2, 1)
		ON CONFLICT (branch_id, service_id)
		DO UPDATE SET next_number = ticket_sequences.next_number + 1
		RETURNING next_number
	`, branchID, serviceID)
	if err := row.Scan(&next); err != nil {
		return 0, err
	}
	return next, nil
}

func insertOutboxEvent(ctx context.Context, tx pgx.Tx, tenantID string, ticket models.Ticket) error {
	payload := map[string]interface{}{
		"ticket_id":     ticket.TicketID,
		"ticket_number": ticket.TicketNumber,
		"status":        ticket.Status,
		"created_at":    ticket.CreatedAt,
		"request_id":    ticket.RequestID,
		"tenant_id":     ticket.TenantID,
		"branch_id":     ticket.BranchID,
		"service_id":    ticket.ServiceID,
		"area_id":       ticket.AreaID,
		"phone":         ticket.Phone,
	}

	payloadJSON, err := jsonBytes(payload)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events (event_id, tenant_id, type, payload_json, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.NewString(), tenantID, "ticket.created", payloadJSON, time.Now().UTC())
	if err != nil {
		return err
	}
	return insertTicketEvent(ctx, tx, ticket.TicketID, "ticket.created", payloadJSON)
}

func insertOutboxEventCalled(ctx context.Context, tx pgx.Tx, tenantID string, ticket models.Ticket) error {
	payload := map[string]interface{}{
		"ticket_id":     ticket.TicketID,
		"ticket_number": ticket.TicketNumber,
		"status":        ticket.Status,
		"called_at":     ticket.CalledAt,
		"counter_id":    ticket.CounterID,
		"request_id":    ticket.RequestID,
		"tenant_id":     ticket.TenantID,
		"branch_id":     ticket.BranchID,
		"service_id":    ticket.ServiceID,
		"area_id":       ticket.AreaID,
	}

	payloadJSON, err := jsonBytes(payload)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events (event_id, tenant_id, type, payload_json, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.NewString(), tenantID, "ticket.called", payloadJSON, time.Now().UTC())
	if err != nil {
		return err
	}
	return insertTicketEvent(ctx, tx, ticket.TicketID, "ticket.called", payloadJSON)
}

func insertOutboxEventGeneric(ctx context.Context, tx pgx.Tx, tenantID, eventType string, ticket models.Ticket) error {
	payload := map[string]interface{}{
		"ticket_id":     ticket.TicketID,
		"ticket_number": ticket.TicketNumber,
		"status":        ticket.Status,
		"request_id":    ticket.RequestID,
		"called_at":     ticket.CalledAt,
		"served_at":     ticket.ServedAt,
		"completed_at":  ticket.CompletedAt,
		"counter_id":    ticket.CounterID,
		"tenant_id":     ticket.TenantID,
		"branch_id":     ticket.BranchID,
		"service_id":    ticket.ServiceID,
		"area_id":       ticket.AreaID,
	}

	payloadJSON, err := jsonBytes(payload)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events (event_id, tenant_id, type, payload_json, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.NewString(), tenantID, eventType, payloadJSON, time.Now().UTC())
	if err != nil {
		return err
	}
	return insertTicketEvent(ctx, tx, ticket.TicketID, eventType, payloadJSON)
}

func insertOutboxEventTransfer(ctx context.Context, tx pgx.Tx, tenantID string, ticket models.Ticket, fromServiceID, toServiceID, reason string) error {
	payload := map[string]interface{}{
		"ticket_id":       ticket.TicketID,
		"ticket_number":   ticket.TicketNumber,
		"status":          ticket.Status,
		"request_id":      ticket.RequestID,
		"from_service_id": fromServiceID,
		"to_service_id":   toServiceID,
		"tenant_id":       ticket.TenantID,
		"branch_id":       ticket.BranchID,
		"service_id":      ticket.ServiceID,
		"area_id":         ticket.AreaID,
	}
	if reason != "" {
		payload["reason"] = reason
	}

	payloadJSON, err := jsonBytes(payload)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events (event_id, tenant_id, type, payload_json, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.NewString(), tenantID, "ticket.transferred", payloadJSON, time.Now().UTC())
	if err != nil {
		return err
	}
	return insertTicketEvent(ctx, tx, ticket.TicketID, "ticket.transferred", payloadJSON)
}

func insertOutboxEventNoShow(ctx context.Context, tx pgx.Tx, tenantID string, ticket models.Ticket, returned bool) error {
	payload := map[string]interface{}{
		"ticket_id":     ticket.TicketID,
		"ticket_number": ticket.TicketNumber,
		"status":        ticket.Status,
		"request_id":    ticket.RequestID,
		"called_at":     ticket.CalledAt,
		"counter_id":    ticket.CounterID,
		"returned":      returned,
		"tenant_id":     ticket.TenantID,
		"branch_id":     ticket.BranchID,
		"service_id":    ticket.ServiceID,
		"area_id":       ticket.AreaID,
	}

	payloadJSON, err := jsonBytes(payload)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events (event_id, tenant_id, type, payload_json, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.NewString(), tenantID, "ticket.no_show", payloadJSON, time.Now().UTC())
	if err != nil {
		return err
	}
	return insertTicketEvent(ctx, tx, ticket.TicketID, "ticket.no_show", payloadJSON)
}

func jsonBytes(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func insertTicketEvent(ctx context.Context, tx pgx.Tx, ticketID, eventType string, payload []byte) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, ticketID); err != nil {
		return err
	}

	var lastSeq int
	var prevHash sql.NullString
	row := tx.QueryRow(ctx, `
		SELECT ticket_seq, hash
		FROM ticket_events
		WHERE ticket_id = $1
		ORDER BY ticket_seq DESC
		LIMIT 1
		FOR UPDATE
	`, ticketID)
	if err := row.Scan(&lastSeq, &prevHash); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	nextSeq := lastSeq + 1
	prev := ""
	if prevHash.Valid {
		prev = prevHash.String
	}
	createdAt := time.Now().UTC()
	hash := store.ComputeTicketEventHash(prev, ticketID, eventType, payload, createdAt, nextSeq)

	_, err := tx.Exec(ctx, `
		INSERT INTO ticket_events (ticket_id, ticket_seq, type, payload, created_at, prev_hash, hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, ticketID, nextSeq, eventType, payload, createdAt, prev, hash)
	return err
}

func findTicketByRequestID(ctx context.Context, tx pgx.Tx, requestID string) (models.Ticket, bool, error) {
	var ticket models.Ticket
	var areaIDNull sql.NullString
	row := tx.QueryRow(ctx, `
		SELECT ticket_id, ticket_number, status, created_at, request_id, area_id, branch_id, service_id, tenant_id
		FROM tickets
		WHERE request_id = $1
	`, requestID)
	if err := row.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &ticket.RequestID, &areaIDNull, &ticket.BranchID, &ticket.ServiceID, &ticket.TenantID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Ticket{}, false, nil
		}
		return models.Ticket{}, false, err
	}
	if areaIDNull.Valid {
		ticket.AreaID = areaIDNull.String
	}
	return ticket, true, nil
}

func findActionRequest(ctx context.Context, tx pgx.Tx, action, requestID string) (models.Ticket, bool, bool, error) {
	var ticketID sql.NullString
	row := tx.QueryRow(ctx, `
		SELECT ticket_id
		FROM ticket_action_requests
		WHERE request_id = $1 AND action = $2
	`, requestID, action)
	if err := row.Scan(&ticketID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Ticket{}, false, false, nil
		}
		return models.Ticket{}, false, false, err
	}

	if !ticketID.Valid {
		return models.Ticket{}, true, true, nil
	}

	var ticket models.Ticket
	var calledAtNull sql.NullTime
	var counterIDNull sql.NullString
	var areaIDNull sql.NullString
	row = tx.QueryRow(ctx, `
		SELECT ticket_id, ticket_number, status, created_at, called_at, counter_id, branch_id, service_id, area_id, tenant_id
		FROM tickets
		WHERE ticket_id = $1
	`, ticketID.String)
	if err := row.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &calledAtNull, &counterIDNull, &ticket.BranchID, &ticket.ServiceID, &areaIDNull, &ticket.TenantID); err != nil {
		return models.Ticket{}, false, false, err
	}
	ticket.RequestID = requestID
	ticket.CalledAt = nullTimePtr(calledAtNull)
	ticket.CounterID = nullStringPtr(counterIDNull)
	if areaIDNull.Valid {
		ticket.AreaID = areaIDNull.String
	}

	return ticket, true, false, nil
}

func (s *Store) updateTicketStatus(ctx context.Context, input store.TicketActionInput, action, fromStatus, toStatus, eventType, timestampColumn string, requireCounter bool) (models.Ticket, bool, error) {
	if !store.ValidTransition(action, fromStatus) {
		return models.Ticket{}, false, store.ErrInvalidState
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return models.Ticket{}, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	existing, found, empty, err := findActionRequest(ctx, tx, action, input.RequestID)
	if err != nil {
		return models.Ticket{}, false, err
	}
	if found {
		if err = tx.Commit(ctx); err != nil {
			return models.Ticket{}, false, err
		}
		if empty {
			return models.Ticket{}, false, store.ErrInvalidState
		}
		return existing, false, nil
	}

	occurredAt := input.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	updateQuery := `
		UPDATE tickets
		SET status = $1
	`
	args := []interface{}{toStatus}
	argPos := 2

	if timestampColumn != "" {
		updateQuery += fmt.Sprintf(", %s = $%d", timestampColumn, argPos)
		args = append(args, occurredAt)
		argPos++
	}

	updateQuery += fmt.Sprintf(`
		WHERE ticket_id = $%d AND tenant_id = $%d AND branch_id = $%d AND status = $%d`, argPos, argPos+1, argPos+2, argPos+3)
	args = append(args, input.TicketID, input.TenantID, input.BranchID, fromStatus)
	argPos += 4

	if requireCounter {
		updateQuery += fmt.Sprintf(" AND counter_id = $%d", argPos)
		args = append(args, input.CounterID)
		argPos++
	}

	updateQuery += " RETURNING ticket_id, ticket_number, status, created_at, called_at, counter_id, served_at, completed_at, branch_id, service_id, area_id, tenant_id"

	var ticket models.Ticket
	var calledAtNull sql.NullTime
	var counterIDNull sql.NullString
	var servedAtNull sql.NullTime
	var completedAtNull sql.NullTime
	var areaIDNull sql.NullString
	row := tx.QueryRow(ctx, updateQuery, args...)
	if err = row.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &calledAtNull, &counterIDNull, &servedAtNull, &completedAtNull, &ticket.BranchID, &ticket.ServiceID, &areaIDNull, &ticket.TenantID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			state, counter, exists, err := loadTicketState(ctx, tx, input.TicketID, input.TenantID, input.BranchID)
			if err != nil {
				return models.Ticket{}, false, err
			}
			if !exists {
				return models.Ticket{}, false, store.ErrTicketNotFound
			}
			if requireCounter && counter != "" && counter != input.CounterID {
				return models.Ticket{}, false, store.ErrCounterMismatch
			}
			if state != fromStatus {
				return models.Ticket{}, false, store.ErrInvalidState
			}
			return models.Ticket{}, false, store.ErrInvalidState
		}
		return models.Ticket{}, false, err
	}

	ticket.RequestID = input.RequestID
	ticket.CalledAt = nullTimePtr(calledAtNull)
	ticket.CounterID = nullStringPtr(counterIDNull)
	ticket.ServedAt = nullTimePtr(servedAtNull)
	ticket.CompletedAt = nullTimePtr(completedAtNull)
	if areaIDNull.Valid {
		ticket.AreaID = areaIDNull.String
	}

	if err = insertActionRequest(ctx, tx, action, input.RequestID, input.TenantID, input.BranchID, input.ServiceID, input.CounterID, ticket.TicketID); err != nil {
		return models.Ticket{}, false, err
	}

	if err = insertOutboxEventGeneric(ctx, tx, input.TenantID, eventType, ticket); err != nil {
		return models.Ticket{}, false, err
	}

	if err = tx.Commit(ctx); err != nil {
		return models.Ticket{}, false, err
	}

	return ticket, true, nil
}

func (s *Store) emitTicketEvent(ctx context.Context, input store.TicketActionInput, action, requiredStatus, eventType string) (models.Ticket, bool, error) {
	if !store.ValidTransition(action, requiredStatus) {
		return models.Ticket{}, false, store.ErrInvalidState
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return models.Ticket{}, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	existing, found, empty, err := findActionRequest(ctx, tx, action, input.RequestID)
	if err != nil {
		return models.Ticket{}, false, err
	}
	if found {
		if err = tx.Commit(ctx); err != nil {
			return models.Ticket{}, false, err
		}
		if empty {
			return models.Ticket{}, false, store.ErrInvalidState
		}
		return existing, false, nil
	}

	ticket, err := getTicketByID(ctx, tx, input.TicketID, input.TenantID, input.BranchID)
	if err != nil {
		return models.Ticket{}, false, err
	}
	if ticket.Status != requiredStatus {
		return models.Ticket{}, false, store.ErrInvalidState
	}
	ticket.RequestID = input.RequestID

	if err = insertActionRequest(ctx, tx, action, input.RequestID, input.TenantID, input.BranchID, input.ServiceID, input.CounterID, ticket.TicketID); err != nil {
		return models.Ticket{}, false, err
	}

	if err = insertOutboxEventGeneric(ctx, tx, input.TenantID, eventType, ticket); err != nil {
		return models.Ticket{}, false, err
	}

	if err = tx.Commit(ctx); err != nil {
		return models.Ticket{}, false, err
	}

	return ticket, true, nil
}

func insertActionRequest(ctx context.Context, tx pgx.Tx, action, requestID, tenantID, branchID, serviceID, counterID, ticketID string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO ticket_action_requests (request_id, action, tenant_id, branch_id, service_id, counter_id, ticket_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (request_id) DO NOTHING
	`, requestID, action, tenantID, branchID, nullIfEmpty(serviceID), nullIfEmpty(counterID), nullIfEmpty(ticketID))
	return err
}

func loadTicketState(ctx context.Context, tx pgx.Tx, ticketID, tenantID, branchID string) (string, string, bool, error) {
	var status string
	var counterID sql.NullString
	row := tx.QueryRow(ctx, `
		SELECT status, counter_id
		FROM tickets
		WHERE ticket_id = $1 AND tenant_id = $2 AND branch_id = $3
	`, ticketID, tenantID, branchID)
	if err := row.Scan(&status, &counterID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", false, nil
		}
		return "", "", false, err
	}
	if counterID.Valid {
		return status, counterID.String, true, nil
	}
	return status, "", true, nil
}

func getTicketByID(ctx context.Context, tx pgx.Tx, ticketID, tenantID, branchID string) (models.Ticket, error) {
	var ticket models.Ticket
	var calledAtNull sql.NullTime
	var counterIDNull sql.NullString
	var servedAtNull sql.NullTime
	var completedAtNull sql.NullTime
	var areaIDNull sql.NullString
	row := tx.QueryRow(ctx, `
		SELECT ticket_id, ticket_number, status, created_at, called_at, counter_id, served_at, completed_at, branch_id, service_id, area_id, tenant_id
		FROM tickets
		WHERE ticket_id = $1 AND tenant_id = $2 AND branch_id = $3
	`, ticketID, tenantID, branchID)
	if err := row.Scan(&ticket.TicketID, &ticket.TicketNumber, &ticket.Status, &ticket.CreatedAt, &calledAtNull, &counterIDNull, &servedAtNull, &completedAtNull, &ticket.BranchID, &ticket.ServiceID, &areaIDNull, &ticket.TenantID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Ticket{}, store.ErrTicketNotFound
		}
		return models.Ticket{}, err
	}
	ticket.CalledAt = nullTimePtr(calledAtNull)
	ticket.CounterID = nullStringPtr(counterIDNull)
	ticket.ServedAt = nullTimePtr(servedAtNull)
	ticket.CompletedAt = nullTimePtr(completedAtNull)
	if areaIDNull.Valid {
		ticket.AreaID = areaIDNull.String
	}
	return ticket, nil
}

func nullIfEmpty(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func hashPhone(phone string) interface{} {
	trimmed := strings.TrimSpace(phone)
	if trimmed == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(trimmed))
	return fmt.Sprintf("%x", sum)
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

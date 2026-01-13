package postgres

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"qms/queue-service/internal/models"
	"qms/queue-service/internal/store"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCallNextConcurrency(t *testing.T) {
	ctx := context.Background()
	st, pool, cleanup := setupTestStore(t, ctx)
	t.Cleanup(cleanup)

	tenantID := uuid.NewString()
	branchID := uuid.NewString()
	serviceID := uuid.NewString()
	counterA := uuid.NewString()
	counterB := uuid.NewString()

	seedBaseData(t, ctx, pool, tenantID, branchID, serviceID, counterA, counterB)

	createTicket(t, ctx, st, tenantID, branchID, serviceID, uuid.NewString())
	createTicket(t, ctx, st, tenantID, branchID, serviceID, uuid.NewString())

	var wg sync.WaitGroup
	results := make(chan callResult, 2)
	inputs := []store.CallNextInput{
		{
			RequestID: uuid.NewString(),
			TenantID:  tenantID,
			BranchID:  branchID,
			ServiceID: serviceID,
			CounterID: counterA,
		},
		{
			RequestID: uuid.NewString(),
			TenantID:  tenantID,
			BranchID:  branchID,
			ServiceID: serviceID,
			CounterID: counterB,
		},
	}

	for _, input := range inputs {
		wg.Add(1)
		go func(in store.CallNextInput) {
			defer wg.Done()
			ticket, ok, err := st.CallNext(ctx, in)
			results <- callResult{ticketID: ticket.TicketID, ok: ok, err: err}
		}(input)
	}
	wg.Wait()
	close(results)

	var ids []string
	for result := range results {
		if result.err != nil {
			t.Fatalf("call next error: %v", result.err)
		}
		if !result.ok {
			t.Fatalf("expected ticket assignment")
		}
		ids = append(ids, result.ticketID)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 tickets, got %d", len(ids))
	}
	if ids[0] == ids[1] {
		t.Fatalf("expected distinct tickets, got %s", ids[0])
	}
}

func TestCreateTicketIdempotency(t *testing.T) {
	ctx := context.Background()
	st, pool, cleanup := setupTestStore(t, ctx)
	t.Cleanup(cleanup)

	tenantID := uuid.NewString()
	branchID := uuid.NewString()
	serviceID := uuid.NewString()
	counterA := uuid.NewString()

	seedBaseData(t, ctx, pool, tenantID, branchID, serviceID, counterA, uuid.NewString())

	requestID := uuid.NewString()
	first := createTicket(t, ctx, st, tenantID, branchID, serviceID, requestID)
	second := createTicket(t, ctx, st, tenantID, branchID, serviceID, requestID)

	if first.TicketID != second.TicketID {
		t.Fatalf("expected same ticket ID for duplicate request")
	}

	var count int
	row := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events WHERE type = 'ticket.created'
	`)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count outbox events: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 ticket.created event, got %d", count)
	}
}

func TestCallNextSkillMismatch(t *testing.T) {
	ctx := context.Background()
	st, pool, cleanup := setupTestStore(t, ctx)
	t.Cleanup(cleanup)

	tenantID := uuid.NewString()
	branchID := uuid.NewString()
	serviceA := uuid.NewString()
	serviceB := uuid.NewString()
	counterA := uuid.NewString()
	counterB := uuid.NewString()

	seedBaseData(t, ctx, pool, tenantID, branchID, serviceA, counterA, counterB)
	if _, err := pool.Exec(ctx, `
		INSERT INTO services (service_id, branch_id, name, code, active)
		VALUES ($1, $2, 'Service B', 'SB', true)
	`, serviceB, branchID); err != nil {
		t.Fatalf("insert service B: %v", err)
	}

	createTicket(t, ctx, st, tenantID, branchID, serviceB, uuid.NewString())

	_, _, err := st.CallNext(ctx, store.CallNextInput{
		RequestID: uuid.NewString(),
		TenantID:  tenantID,
		BranchID:  branchID,
		ServiceID: serviceB,
		CounterID: counterA,
	})
	if err == nil || !errors.Is(err, store.ErrAccessDenied) {
		t.Fatalf("expected access denied, got %v", err)
	}
}

func TestCallNextAppointmentRatio(t *testing.T) {
	ctx := context.Background()
	st, pool, cleanup := setupTestStore(t, ctx)
	t.Cleanup(cleanup)

	tenantID := uuid.NewString()
	branchID := uuid.NewString()
	serviceID := uuid.NewString()
	counterID := uuid.NewString()

	seedBaseData(t, ctx, pool, tenantID, branchID, serviceID, counterID, uuid.NewString())

	if _, err := pool.Exec(ctx, `
		INSERT INTO service_policies (tenant_id, branch_id, service_id, no_show_grace_seconds, return_to_queue, appointment_ratio_percent, appointment_window_size, appointment_boost_minutes)
		VALUES ($1, $2, $3, 300, false, 100, 5, 0)
	`, tenantID, branchID, serviceID); err != nil {
		t.Fatalf("insert policy: %v", err)
	}

	apptID := createAppointment(t, ctx, pool, tenantID, branchID, serviceID, time.Now().Add(30*time.Minute))
	apptTicket, err := st.CheckInAppointment(ctx, uuid.NewString(), tenantID, branchID, apptID)
	if err != nil {
		t.Fatalf("checkin appointment: %v", err)
	}
	createTicket(t, ctx, st, tenantID, branchID, serviceID, uuid.NewString())

	ticket, _, err := st.CallNext(ctx, store.CallNextInput{
		RequestID: uuid.NewString(),
		TenantID:  tenantID,
		BranchID:  branchID,
		ServiceID: serviceID,
		CounterID: counterID,
	})
	if err != nil {
		t.Fatalf("call next: %v", err)
	}
	if ticket.TicketID != apptTicket.TicketID {
		t.Fatalf("expected appointment ticket, got %s", ticket.TicketID)
	}
}

func TestCallNextAppointmentBoost(t *testing.T) {
	ctx := context.Background()
	st, pool, cleanup := setupTestStore(t, ctx)
	t.Cleanup(cleanup)

	tenantID := uuid.NewString()
	branchID := uuid.NewString()
	serviceID := uuid.NewString()
	counterID := uuid.NewString()

	seedBaseData(t, ctx, pool, tenantID, branchID, serviceID, counterID, uuid.NewString())

	if _, err := pool.Exec(ctx, `
		INSERT INTO service_policies (tenant_id, branch_id, service_id, no_show_grace_seconds, return_to_queue, appointment_ratio_percent, appointment_window_size, appointment_boost_minutes)
		VALUES ($1, $2, $3, 300, false, 0, 5, 60)
	`, tenantID, branchID, serviceID); err != nil {
		t.Fatalf("insert policy: %v", err)
	}

	apptID := createAppointment(t, ctx, pool, tenantID, branchID, serviceID, time.Now().Add(20*time.Minute))
	apptTicket, err := st.CheckInAppointment(ctx, uuid.NewString(), tenantID, branchID, apptID)
	if err != nil {
		t.Fatalf("checkin appointment: %v", err)
	}
	createTicket(t, ctx, st, tenantID, branchID, serviceID, uuid.NewString())

	ticket, _, err := st.CallNext(ctx, store.CallNextInput{
		RequestID: uuid.NewString(),
		TenantID:  tenantID,
		BranchID:  branchID,
		ServiceID: serviceID,
		CounterID: counterID,
	})
	if err != nil {
		t.Fatalf("call next: %v", err)
	}
	if ticket.TicketID != apptTicket.TicketID {
		t.Fatalf("expected boosted appointment ticket, got %s", ticket.TicketID)
	}
}

func TestTicketEventHashAndRehydrate(t *testing.T) {
	ctx := context.Background()
	st, pool, cleanup := setupTestStore(t, ctx)
	t.Cleanup(cleanup)

	tenantID := uuid.NewString()
	branchID := uuid.NewString()
	serviceID := uuid.NewString()
	counterID := uuid.NewString()

	seedBaseData(t, ctx, pool, tenantID, branchID, serviceID, counterID, uuid.NewString())

	ticket := createTicket(t, ctx, st, tenantID, branchID, serviceID, uuid.NewString())

	called, _, err := st.CallNext(ctx, store.CallNextInput{
		RequestID: uuid.NewString(),
		TenantID:  tenantID,
		BranchID:  branchID,
		ServiceID: serviceID,
		CounterID: counterID,
	})
	if err != nil {
		t.Fatalf("call next: %v", err)
	}

	serving, _, err := st.StartServing(ctx, store.TicketActionInput{
		RequestID: uuid.NewString(),
		TenantID:  tenantID,
		BranchID:  branchID,
		TicketID:  ticket.TicketID,
		CounterID: counterID,
	})
	if err != nil {
		t.Fatalf("start serving: %v", err)
	}

	complete, _, err := st.CompleteTicket(ctx, store.TicketActionInput{
		RequestID: uuid.NewString(),
		TenantID:  tenantID,
		BranchID:  branchID,
		TicketID:  ticket.TicketID,
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	events, err := st.ListTicketEvents(ctx, tenantID, ticket.TicketID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) < 3 {
		t.Fatalf("expected ticket events, got %d", len(events))
	}

	prevHash := ""
	for idx, event := range events {
		if event.TicketSeq != idx+1 {
			t.Fatalf("expected seq %d, got %d", idx+1, event.TicketSeq)
		}
		expected := store.ComputeTicketEventHash(prevHash, event.TicketID, event.Type, event.Payload, event.CreatedAt, event.TicketSeq)
		if event.Hash != expected {
			t.Fatalf("hash mismatch for seq %d", event.TicketSeq)
		}
		prevHash = event.Hash
	}

	rehydrated, err := store.RehydrateTicket(events)
	if err != nil {
		t.Fatalf("rehydrate: %v", err)
	}
	if rehydrated.Status != complete.Status {
		t.Fatalf("status mismatch: %s vs %s", rehydrated.Status, complete.Status)
	}
	if rehydrated.ServiceID != called.ServiceID {
		t.Fatalf("service mismatch: %s vs %s", rehydrated.ServiceID, called.ServiceID)
	}
	if rehydrated.CounterID == nil || *rehydrated.CounterID != *called.CounterID {
		t.Fatalf("counter mismatch")
	}
	if rehydrated.ServedAt == nil || serving.ServedAt == nil {
		t.Fatalf("served_at missing")
	}
	if rehydrated.CompletedAt == nil || complete.CompletedAt == nil {
		t.Fatalf("completed_at missing")
	}
}

type callResult struct {
	ticketID string
	ok       bool
	err      error
}

func setupTestStore(t *testing.T, ctx context.Context) (*Store, *pgxpool.Pool, func()) {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = os.Getenv("DB_DSN")
	}
	if dsn == "" {
		t.Skip("TEST_DB_DSN or DB_DSN is required for integration tests")
	}

	schema := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := createSchema(ctx, dsn, schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	pool, err := newPoolWithSchema(ctx, dsn, schema)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}

	if err := applyMigrations(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("apply migrations: %v", err)
	}

	store := NewStore(pool, Options{NoShowReturnToQueue: false, PriorityStreakLimit: 3})
	cleanup := func() {
		pool.Close()
		_ = dropSchema(context.Background(), dsn, schema)
	}
	return store, pool, cleanup
}

func createSchema(ctx context.Context, dsn, schema string) error {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, "CREATE SCHEMA "+schema)
	return err
}

func dropSchema(ctx context.Context, dsn, schema string) error {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
	return err
}

func newPoolWithSchema(ctx context.Context, dsn, schema string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	return pgxpool.NewWithConfig(ctx, cfg)
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	dir := filepath.Join("..", "..", "..", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	for _, name := range files {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(content)) == "" {
			continue
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return err
		}
	}
	return nil
}

func seedBaseData(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, branchID, serviceID, counterA, counterB string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants (tenant_id, name) VALUES ($1, 'Tenant')
	`, tenantID); err != nil {
		t.Fatalf("insert tenant: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO branches (branch_id, tenant_id, name) VALUES ($1, $2, 'Branch')
	`, branchID, tenantID); err != nil {
		t.Fatalf("insert branch: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO services (service_id, branch_id, name, code, active) VALUES ($1, $2, 'Service', 'SV', true)
	`, serviceID, branchID); err != nil {
		t.Fatalf("insert service: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO counters (counter_id, branch_id, name, status) VALUES ($1, $2, 'Counter A', 'active')
	`, counterA, branchID); err != nil {
		t.Fatalf("insert counter A: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO counters (counter_id, branch_id, name, status) VALUES ($1, $2, 'Counter B', 'active')
	`, counterB, branchID); err != nil {
		t.Fatalf("insert counter B: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO counter_services (counter_id, service_id) VALUES ($1, $2)
	`, counterA, serviceID); err != nil {
		t.Fatalf("map counter A: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO counter_services (counter_id, service_id) VALUES ($1, $2)
	`, counterB, serviceID); err != nil {
		t.Fatalf("map counter B: %v", err)
	}
}

func createTicket(t *testing.T, ctx context.Context, st *Store, tenantID, branchID, serviceID, requestID string) models.Ticket {
	t.Helper()
	ticket, _, err := st.CreateTicket(ctx, store.CreateTicketInput{
		RequestID:     requestID,
		TenantID:      tenantID,
		BranchID:      branchID,
		ServiceID:     serviceID,
		Channel:       "kiosk",
		PriorityClass: "regular",
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	return ticket
}

func createAppointment(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, branchID, serviceID string, scheduledAt time.Time) string {
	t.Helper()
	appointmentID := uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO appointments (appointment_id, tenant_id, branch_id, service_id, scheduled_at, status)
		VALUES ($1, $2, $3, $4, $5, 'scheduled')
	`, appointmentID, tenantID, branchID, serviceID, scheduledAt); err != nil {
		t.Fatalf("insert appointment: %v", err)
	}
	return appointmentID
}

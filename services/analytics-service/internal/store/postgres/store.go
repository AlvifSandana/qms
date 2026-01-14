package postgres

import (
	"context"
	"errors"
	"time"

	"qms/analytics-service/internal/store"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) GetKPIs(ctx context.Context, tenantID, branchID, serviceID string, from, to time.Time) (store.KPIResult, error) {
	var result store.KPIResult
	row := s.pool.QueryRow(ctx, `
		SELECT
			AVG(EXTRACT(EPOCH FROM (called_at - created_at))) AS avg_wait,
			AVG(EXTRACT(EPOCH FROM (completed_at - served_at))) AS avg_service,
			COUNT(*)
		FROM tickets
		WHERE tenant_id = $1 AND branch_id = $2 AND service_id = $3
			AND created_at >= $4 AND created_at <= $5
	`, tenantID, branchID, serviceID, from, to)
	if err := row.Scan(&result.AvgWaitSeconds, &result.AvgServiceSeconds, &result.Count); err != nil {
		return store.KPIResult{}, err
	}
	return result, nil
}

func (s *Store) GetRealtime(ctx context.Context, tenantID, branchID, serviceID string) (store.RealtimeResult, error) {
	var result store.RealtimeResult
	row := s.pool.QueryRow(ctx, `
		SELECT
			SUM(CASE WHEN status = 'waiting' THEN 1 ELSE 0 END) AS waiting,
			SUM(CASE WHEN status = 'serving' THEN 1 ELSE 0 END) AS serving
		FROM tickets
		WHERE tenant_id = $1 AND branch_id = $2 AND service_id = $3
	`, tenantID, branchID, serviceID)
	if err := row.Scan(&result.QueueLength, &result.Serving); err != nil {
		return store.RealtimeResult{}, err
	}

	row = s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS active_counters,
			SUM(CASE WHEN status = 'busy' THEN 1 ELSE 0 END) AS busy_counters
		FROM counters c
		JOIN branches b ON b.branch_id = c.branch_id
		WHERE b.tenant_id = $1 AND c.branch_id = $2
	`, tenantID, branchID)
	if err := row.Scan(&result.ActiveCounters, &result.BusyCounters); err != nil {
		return store.RealtimeResult{}, err
	}
	return result, nil
}

func (s *Store) ListTickets(ctx context.Context, tenantID, branchID, serviceID string, from, to time.Time) ([]store.TicketRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ticket_id, ticket_number, status, created_at, called_at, served_at, completed_at
		FROM tickets
		WHERE tenant_id = $1 AND branch_id = $2 AND service_id = $3
			AND created_at >= $4 AND created_at <= $5
		ORDER BY created_at ASC
	`, tenantID, branchID, serviceID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []store.TicketRow
	for rows.Next() {
		var row store.TicketRow
		if err := rows.Scan(&row.TicketID, &row.Number, &row.Status, &row.CreatedAt, &row.CalledAt, &row.ServedAt, &row.CompletedAt); err != nil {
			return nil, err
		}
		tickets = append(tickets, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tickets, nil
}

func (s *Store) CreateScheduledReport(ctx context.Context, tenantID, branchID, serviceID, cron, channel, recipient string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO scheduled_reports (report_id, tenant_id, branch_id, service_id, cron, channel, recipient, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE)
	`, uuid.NewString(), tenantID, branchID, serviceID, cron, channel, recipient)
	return err
}

func (s *Store) ListScheduledReports(ctx context.Context, tenantID string) ([]store.ScheduledReport, error) {
	query := `
		SELECT report_id, tenant_id, branch_id, service_id, cron, channel, recipient, active, last_sent_at
		FROM scheduled_reports
	`
	args := []interface{}{}
	if tenantID != "" {
		query += " WHERE tenant_id = $1"
		args = append(args, tenantID)
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []store.ScheduledReport
	for rows.Next() {
		var r store.ScheduledReport
		if err := rows.Scan(&r.ReportID, &r.TenantID, &r.BranchID, &r.ServiceID, &r.Cron, &r.Channel, &r.Recipient, &r.Active, &r.LastSentAt); err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reports, nil
}

func (s *Store) UpdateScheduledReportSent(ctx context.Context, reportID string, sentAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE scheduled_reports
		SET last_sent_at = $1
		WHERE report_id = $2
	`, sentAt, reportID)
	return err
}

func (s *Store) ListAnomalies(ctx context.Context, tenantID string) ([]store.Anomaly, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT anomaly_id, tenant_id, branch_id, service_id, type, value, threshold, created_at
		FROM anomalies
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var anomalies []store.Anomaly
	for rows.Next() {
		var a store.Anomaly
		if err := rows.Scan(&a.AnomalyID, &a.TenantID, &a.BranchID, &a.ServiceID, &a.Type, &a.Value, &a.Threshold, &a.CreatedAt); err != nil {
			return nil, err
		}
		anomalies = append(anomalies, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return anomalies, nil
}

func (s *Store) InsertAnomaly(ctx context.Context, anomaly store.Anomaly) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO anomalies (anomaly_id, tenant_id, branch_id, service_id, type, value, threshold)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.NewString(), anomaly.TenantID, anomaly.BranchID, anomaly.ServiceID, anomaly.Type, anomaly.Value, anomaly.Threshold)
	return err
}

func (s *Store) ListServices(ctx context.Context) ([]store.ServiceRef, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT b.tenant_id, s.branch_id, s.service_id
		FROM services s
		JOIN branches b ON b.branch_id = s.branch_id
		WHERE s.active = TRUE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []store.ServiceRef
	for rows.Next() {
		var ref store.ServiceRef
		if err := rows.Scan(&ref.TenantID, &ref.BranchID, &ref.ServiceID); err != nil {
			return nil, err
		}
		services = append(services, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return services, nil
}

func (s *Store) GetSession(ctx context.Context, sessionID string) (store.Session, error) {
	var session store.Session
	row := s.pool.QueryRow(ctx, `
		SELECT s.session_id, s.user_id, s.expires_at, u.tenant_id, r.name
		FROM sessions s
		JOIN users u ON u.user_id = s.user_id
		JOIN roles r ON r.role_id = u.role_id
		WHERE s.session_id = $1 AND s.expires_at > NOW()
	`, sessionID)
	if err := row.Scan(&session.SessionID, &session.UserID, &session.ExpiresAt, &session.TenantID, &session.Role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.Session{}, store.ErrSessionNotFound
		}
		return store.Session{}, err
	}
	return session, nil
}

package postgres

import (
	"context"
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

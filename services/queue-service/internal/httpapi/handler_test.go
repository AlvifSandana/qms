package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"qms/queue-service/internal/models"
	"qms/queue-service/internal/store"
)

type fakeStore struct {
	createFn        func(ctx context.Context, input store.CreateTicketInput) (models.Ticket, bool, error)
	getTicketFn     func(ctx context.Context, tenantID, branchID, ticketID string) (models.Ticket, bool, error)
	listQueueFn     func(ctx context.Context, tenantID, branchID, serviceID string) ([]models.Ticket, error)
	callFn          func(ctx context.Context, input store.CallNextInput) (models.Ticket, bool, error)
	startFn         func(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error)
	completeFn      func(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error)
	cancelFn        func(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error)
	recallFn        func(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error)
	holdFn          func(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error)
	unholdFn        func(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error)
	transferFn      func(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error)
	noShowFn        func(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error)
	snapshotFn      func(ctx context.Context, tenantID, branchID, serviceID string) ([]models.Ticket, error)
	outboxFn        func(ctx context.Context, tenantID string, after time.Time, limit int) ([]store.OutboxEvent, error)
	eventsFn        func(ctx context.Context, tenantID, ticketID string) ([]store.TicketEvent, error)
	countersFn      func(ctx context.Context, tenantID, branchID string) ([]models.Counter, error)
	updateCounterFn func(ctx context.Context, tenantID, branchID, counterID, status string) error
	servicesFn      func(ctx context.Context, tenantID, branchID string) ([]models.Service, error)
	activeFn        func(ctx context.Context, tenantID, branchID, counterID string) (models.Ticket, bool, error)
	apptFn          func(ctx context.Context, requestID, tenantID, branchID, appointmentID string) (models.Ticket, error)
}

func (f fakeStore) CreateTicket(ctx context.Context, input store.CreateTicketInput) (models.Ticket, bool, error) {
	if f.createFn == nil {
		return models.Ticket{}, false, nil
	}
	return f.createFn(ctx, input)
}

func (f fakeStore) GetTicket(ctx context.Context, tenantID, branchID, ticketID string) (models.Ticket, bool, error) {
	if f.getTicketFn == nil {
		return models.Ticket{}, false, nil
	}
	return f.getTicketFn(ctx, tenantID, branchID, ticketID)
}

func (f fakeStore) ListQueue(ctx context.Context, tenantID, branchID, serviceID string) ([]models.Ticket, error) {
	if f.listQueueFn == nil {
		return nil, nil
	}
	return f.listQueueFn(ctx, tenantID, branchID, serviceID)
}

func (f fakeStore) CallNext(ctx context.Context, input store.CallNextInput) (models.Ticket, bool, error) {
	if f.callFn == nil {
		return models.Ticket{}, false, nil
	}
	return f.callFn(ctx, input)
}

func (f fakeStore) StartServing(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	if f.startFn == nil {
		return models.Ticket{}, false, nil
	}
	return f.startFn(ctx, input)
}

func (f fakeStore) CompleteTicket(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	if f.completeFn == nil {
		return models.Ticket{}, false, nil
	}
	return f.completeFn(ctx, input)
}

func (f fakeStore) CancelTicket(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	if f.cancelFn == nil {
		return models.Ticket{}, false, nil
	}
	return f.cancelFn(ctx, input)
}

func (f fakeStore) RecallTicket(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	if f.recallFn == nil {
		return models.Ticket{}, false, nil
	}
	return f.recallFn(ctx, input)
}

func (f fakeStore) HoldTicket(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	if f.holdFn == nil {
		return models.Ticket{}, false, nil
	}
	return f.holdFn(ctx, input)
}

func (f fakeStore) UnholdTicket(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	if f.unholdFn == nil {
		return models.Ticket{}, false, nil
	}
	return f.unholdFn(ctx, input)
}

func (f fakeStore) TransferTicket(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	if f.transferFn == nil {
		return models.Ticket{}, false, nil
	}
	return f.transferFn(ctx, input)
}

func (f fakeStore) NoShowTicket(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
	if f.noShowFn == nil {
		return models.Ticket{}, false, nil
	}
	return f.noShowFn(ctx, input)
}

func (f fakeStore) SnapshotTickets(ctx context.Context, tenantID, branchID, serviceID string) ([]models.Ticket, error) {
	if f.snapshotFn == nil {
		return nil, nil
	}
	return f.snapshotFn(ctx, tenantID, branchID, serviceID)
}

func (f fakeStore) ListOutboxEvents(ctx context.Context, tenantID string, after time.Time, limit int) ([]store.OutboxEvent, error) {
	if f.outboxFn == nil {
		return nil, nil
	}
	return f.outboxFn(ctx, tenantID, after, limit)
}

func (f fakeStore) ListTicketEvents(ctx context.Context, tenantID, ticketID string) ([]store.TicketEvent, error) {
	if f.eventsFn == nil {
		return nil, nil
	}
	return f.eventsFn(ctx, tenantID, ticketID)
}

func (f fakeStore) ListCounters(ctx context.Context, tenantID, branchID string) ([]models.Counter, error) {
	if f.countersFn == nil {
		return nil, nil
	}
	return f.countersFn(ctx, tenantID, branchID)
}

func (f fakeStore) UpdateCounterStatus(ctx context.Context, tenantID, branchID, counterID, status string) error {
	if f.updateCounterFn == nil {
		return nil
	}
	return f.updateCounterFn(ctx, tenantID, branchID, counterID, status)
}

func (f fakeStore) ListServices(ctx context.Context, tenantID, branchID string) ([]models.Service, error) {
	if f.servicesFn == nil {
		return nil, nil
	}
	return f.servicesFn(ctx, tenantID, branchID)
}

func (f fakeStore) GetActiveTicket(ctx context.Context, tenantID, branchID, counterID string) (models.Ticket, bool, error) {
	if f.activeFn == nil {
		return models.Ticket{}, false, nil
	}
	return f.activeFn(ctx, tenantID, branchID, counterID)
}

func (f fakeStore) CheckInAppointment(ctx context.Context, requestID, tenantID, branchID, appointmentID string) (models.Ticket, error) {
	if f.apptFn == nil {
		return models.Ticket{}, nil
	}
	return f.apptFn(ctx, requestID, tenantID, branchID, appointmentID)
}

func TestCreateTicketSuccess(t *testing.T) {
	createdAt := time.Date(2026, 1, 12, 8, 0, 0, 0, time.UTC)
	st := fakeStore{
		createFn: func(ctx context.Context, input store.CreateTicketInput) (models.Ticket, bool, error) {
			return models.Ticket{
				TicketID:     "ticket-1",
				TicketNumber: "CS-001",
				Status:       models.StatusWaiting,
				CreatedAt:    createdAt,
				RequestID:    input.RequestID,
			}, true, nil
		},
		callFn: func(ctx context.Context, input store.CallNextInput) (models.Ticket, bool, error) {
			return models.Ticket{}, false, nil
		},
	}

	h := NewHandler(st, Options{})

	payload := map[string]string{
		"request_id": "11111111-1111-1111-1111-111111111111",
		"tenant_id":  "22222222-2222-2222-2222-222222222222",
		"branch_id":  "33333333-3333-3333-3333-333333333333",
		"service_id": "44444444-4444-4444-4444-444444444444",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/tickets", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	h.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var ticket models.Ticket
	if err := json.NewDecoder(resp.Body).Decode(&ticket); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if ticket.TicketID == "" || ticket.TicketNumber == "" || ticket.Status != models.StatusWaiting {
		t.Fatalf("unexpected ticket response: %+v", ticket)
	}
}

func TestGetTicketSuccess(t *testing.T) {
	st := fakeStore{
		getTicketFn: func(ctx context.Context, tenantID, branchID, ticketID string) (models.Ticket, bool, error) {
			return models.Ticket{
				TicketID:     ticketID,
				TicketNumber: "CS-010",
				Status:       models.StatusWaiting,
			}, true, nil
		},
	}
	h := NewHandler(st, Options{})

	req := httptest.NewRequest(http.MethodGet, "/api/tickets/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa?tenant_id=bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb&branch_id=cccccccc-cccc-cccc-cccc-cccccccccccc", nil)
	resp := httptest.NewRecorder()

	h.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}

func TestGetTicketMissingParams(t *testing.T) {
	h := NewHandler(fakeStore{}, Options{})

	req := httptest.NewRequest(http.MethodGet, "/api/tickets/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", nil)
	resp := httptest.NewRecorder()

	h.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestListQueueSuccess(t *testing.T) {
	st := fakeStore{
		listQueueFn: func(ctx context.Context, tenantID, branchID, serviceID string) ([]models.Ticket, error) {
			return []models.Ticket{{TicketID: "ticket-1"}}, nil
		},
	}
	h := NewHandler(st, Options{})

	req := httptest.NewRequest(http.MethodGet, "/api/queues?tenant_id=bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb&branch_id=cccccccc-cccc-cccc-cccc-cccccccccccc&service_id=dddddddd-dddd-dddd-dddd-dddddddddddd", nil)
	resp := httptest.NewRecorder()

	h.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}

func TestCreateTicketMissingFields(t *testing.T) {
	st := fakeStore{
		createFn: func(ctx context.Context, input store.CreateTicketInput) (models.Ticket, bool, error) {
			return models.Ticket{}, false, nil
		},
		callFn: func(ctx context.Context, input store.CallNextInput) (models.Ticket, bool, error) {
			return models.Ticket{}, false, nil
		},
	}

	h := NewHandler(st, Options{})

	payload := map[string]string{
		"request_id": "11111111-1111-1111-1111-111111111111",
		"tenant_id":  "22222222-2222-2222-2222-222222222222",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/tickets", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	h.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestCreateTicketServiceNotFound(t *testing.T) {
	st := fakeStore{
		createFn: func(ctx context.Context, input store.CreateTicketInput) (models.Ticket, bool, error) {
			return models.Ticket{}, false, store.ErrServiceNotFound
		},
		callFn: func(ctx context.Context, input store.CallNextInput) (models.Ticket, bool, error) {
			return models.Ticket{}, false, nil
		},
	}

	h := NewHandler(st, Options{})

	payload := map[string]string{
		"request_id": "11111111-1111-1111-1111-111111111111",
		"tenant_id":  "22222222-2222-2222-2222-222222222222",
		"branch_id":  "33333333-3333-3333-3333-333333333333",
		"service_id": "44444444-4444-4444-4444-444444444444",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/tickets", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	h.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.Code)
	}
}

func TestAppointmentCheckinHolidayClosed(t *testing.T) {
	st := fakeStore{
		apptFn: func(ctx context.Context, requestID, tenantID, branchID, appointmentID string) (models.Ticket, error) {
			return models.Ticket{}, store.ErrHolidayClosed
		},
	}

	h := NewHandler(st, Options{})

	payload := map[string]string{
		"request_id":     "11111111-1111-1111-1111-111111111111",
		"tenant_id":      "22222222-2222-2222-2222-222222222222",
		"branch_id":      "33333333-3333-3333-3333-333333333333",
		"appointment_id": "44444444-4444-4444-4444-444444444444",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/appointments/checkin", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	h.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", resp.Code)
	}

	var errResp errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if errResp.Error.Code != "holiday_closed" {
		t.Fatalf("expected error code holiday_closed, got %s", errResp.Error.Code)
	}
}

func TestCallNextSuccess(t *testing.T) {
	createdAt := time.Date(2026, 1, 12, 8, 0, 0, 0, time.UTC)
	calledAt := time.Date(2026, 1, 12, 8, 1, 0, 0, time.UTC)
	counterID := "55555555-5555-5555-5555-555555555555"
	st := fakeStore{
		createFn: func(ctx context.Context, input store.CreateTicketInput) (models.Ticket, bool, error) {
			return models.Ticket{}, false, nil
		},
		callFn: func(ctx context.Context, input store.CallNextInput) (models.Ticket, bool, error) {
			return models.Ticket{
				TicketID:     "ticket-2",
				TicketNumber: "CS-002",
				Status:       models.StatusCalled,
				CreatedAt:    createdAt,
				RequestID:    input.RequestID,
				CalledAt:     &calledAt,
				CounterID:    &counterID,
			}, true, nil
		},
	}

	h := NewHandler(st, Options{})
	payload := map[string]string{
		"request_id": "11111111-1111-1111-1111-111111111111",
		"tenant_id":  "22222222-2222-2222-2222-222222222222",
		"branch_id":  "33333333-3333-3333-3333-333333333333",
		"service_id": "44444444-4444-4444-4444-444444444444",
		"counter_id": counterID,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/tickets/actions/call-next", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	h.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}

func TestCallNextEmptyQueue(t *testing.T) {
	st := fakeStore{
		createFn: func(ctx context.Context, input store.CreateTicketInput) (models.Ticket, bool, error) {
			return models.Ticket{}, false, nil
		},
		callFn: func(ctx context.Context, input store.CallNextInput) (models.Ticket, bool, error) {
			return models.Ticket{}, false, store.ErrNoTicket
		},
	}

	h := NewHandler(st, Options{})
	payload := map[string]string{
		"request_id": "11111111-1111-1111-1111-111111111111",
		"tenant_id":  "22222222-2222-2222-2222-222222222222",
		"branch_id":  "33333333-3333-3333-3333-333333333333",
		"service_id": "44444444-4444-4444-4444-444444444444",
		"counter_id": "55555555-5555-5555-5555-555555555555",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/tickets/actions/call-next", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	h.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", resp.Code)
	}
}

func TestStartServingMissingCounter(t *testing.T) {
	st := fakeStore{}
	h := NewHandler(st, Options{})
	payload := map[string]string{
		"request_id": "11111111-1111-1111-1111-111111111111",
		"tenant_id":  "22222222-2222-2222-2222-222222222222",
		"branch_id":  "33333333-3333-3333-3333-333333333333",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/tickets/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/actions/start", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	h.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestTransferMissingService(t *testing.T) {
	st := fakeStore{}
	h := NewHandler(st, Options{})
	payload := map[string]string{
		"request_id": "11111111-1111-1111-1111-111111111111",
		"tenant_id":  "22222222-2222-2222-2222-222222222222",
		"branch_id":  "33333333-3333-3333-3333-333333333333",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/tickets/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/actions/transfer", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	h.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestCompleteTicketSuccess(t *testing.T) {
	st := fakeStore{
		completeFn: func(ctx context.Context, input store.TicketActionInput) (models.Ticket, bool, error) {
			return models.Ticket{
				TicketID:     input.TicketID,
				TicketNumber: "CS-010",
				Status:       models.StatusDone,
				CreatedAt:    time.Now().UTC(),
				RequestID:    input.RequestID,
			}, true, nil
		},
	}
	h := NewHandler(st, Options{})
	payload := map[string]string{
		"request_id": "11111111-1111-1111-1111-111111111111",
		"tenant_id":  "22222222-2222-2222-2222-222222222222",
		"branch_id":  "33333333-3333-3333-3333-333333333333",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/tickets/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/actions/complete", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	h.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}

func TestListServicesMissingParams(t *testing.T) {
	st := fakeStore{}
	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	resp := httptest.NewRecorder()

	NewHandler(st, Options{}).Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestActiveTicketMissingParams(t *testing.T) {
	st := fakeStore{}
	req := httptest.NewRequest(http.MethodGet, "/api/tickets/active", nil)
	resp := httptest.NewRecorder()

	NewHandler(st, Options{}).Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

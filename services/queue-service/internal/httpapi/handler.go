package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"qms/queue-service/internal/store"

	"github.com/google/uuid"
)

type Handler struct {
	store store.TicketStore
	noShowReturnToQueue bool
}

type createTicketRequest struct {
	RequestID     string `json:"request_id"`
	TenantID      string `json:"tenant_id"`
	BranchID      string `json:"branch_id"`
	ServiceID     string `json:"service_id"`
	AreaID        string `json:"area_id"`
	Channel       string `json:"channel"`
	PriorityClass string `json:"priority_class"`
	Phone         string `json:"phone"`
}

type callNextRequest struct {
	RequestID string `json:"request_id"`
	TenantID  string `json:"tenant_id"`
	BranchID  string `json:"branch_id"`
	ServiceID string `json:"service_id"`
	CounterID string `json:"counter_id"`
}

type errorResponse struct {
	RequestID string         `json:"request_id"`
	Error     responseError  `json:"error"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Options struct {
	NoShowReturnToQueue bool
}

func NewHandler(store store.TicketStore, options Options) *Handler {
	return &Handler{
		store: store,
		noShowReturnToQueue: options.NoShowReturnToQueue,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.handleHealth)
	mux.HandleFunc("/api/tickets", h.handleTickets)
	mux.HandleFunc("/api/tickets/actions/call-next", h.handleCallNext)
	mux.HandleFunc("/api/tickets/active", h.handleActiveTicket)
	mux.HandleFunc("/api/tickets/snapshot", h.handleTicketSnapshot)
	mux.HandleFunc("/api/tickets/", h.handleTicketActions)
	mux.HandleFunc("/api/appointments/checkin", h.handleAppointmentCheckin)
	mux.HandleFunc("/api/events", h.handleEvents)
	mux.HandleFunc("/api/services", h.handleServices)
	return mux
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleTickets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req createTicketRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, req.RequestID, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	req.RequestID = strings.TrimSpace(req.RequestID)
	req.TenantID = strings.TrimSpace(req.TenantID)
	req.BranchID = strings.TrimSpace(req.BranchID)
	req.ServiceID = strings.TrimSpace(req.ServiceID)
	req.Channel = strings.TrimSpace(req.Channel)
	req.PriorityClass = strings.TrimSpace(req.PriorityClass)
	req.Phone = strings.TrimSpace(req.Phone)

	if req.RequestID == "" || req.TenantID == "" || req.BranchID == "" || req.ServiceID == "" {
		writeError(w, req.RequestID, http.StatusBadRequest, "invalid_request", "request_id, tenant_id, branch_id, and service_id are required")
		return
	}

	if !isValidUUID(req.RequestID) || !isValidUUID(req.TenantID) || !isValidUUID(req.BranchID) || !isValidUUID(req.ServiceID) {
		writeError(w, req.RequestID, http.StatusBadRequest, "invalid_request", "request_id, tenant_id, branch_id, and service_id must be UUIDs")
		return
	}

	if req.AreaID != "" && !isValidUUID(req.AreaID) {
		writeError(w, req.RequestID, http.StatusBadRequest, "invalid_request", "area_id must be a UUID when provided")
		return
	}

	if req.Channel == "" {
		req.Channel = "staff"
	}
	if req.PriorityClass == "" {
		req.PriorityClass = "regular"
	}
	if req.Phone != "" && !isValidPhone(req.Phone) {
		writeError(w, req.RequestID, http.StatusBadRequest, "invalid_request", "phone must be 8-16 digits")
		return
	}

	input := store.CreateTicketInput{
		RequestID:     req.RequestID,
		TenantID:      req.TenantID,
		BranchID:      req.BranchID,
		ServiceID:     req.ServiceID,
		AreaID:        req.AreaID,
		Channel:       req.Channel,
		PriorityClass: req.PriorityClass,
		Phone:         req.Phone,
		CreatedAt:     time.Now().UTC(),
	}

	ticket, _, err := h.store.CreateTicket(r.Context(), input)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, req.RequestID, status, code, msg)
		return
	}

	writeJSON(w, http.StatusOK, ticket)
}

func isValidUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

func isValidPhone(value string) bool {
	if len(value) < 8 || len(value) > 16 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (h *Handler) handleCallNext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req callNextRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, req.RequestID, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	req.RequestID = strings.TrimSpace(req.RequestID)
	req.TenantID = strings.TrimSpace(req.TenantID)
	req.BranchID = strings.TrimSpace(req.BranchID)
	req.ServiceID = strings.TrimSpace(req.ServiceID)
	req.CounterID = strings.TrimSpace(req.CounterID)

	if req.RequestID == "" || req.TenantID == "" || req.BranchID == "" || req.ServiceID == "" || req.CounterID == "" {
		writeError(w, req.RequestID, http.StatusBadRequest, "invalid_request", "request_id, tenant_id, branch_id, service_id, and counter_id are required")
		return
	}

	if !isValidUUID(req.RequestID) || !isValidUUID(req.TenantID) || !isValidUUID(req.BranchID) || !isValidUUID(req.ServiceID) || !isValidUUID(req.CounterID) {
		writeError(w, req.RequestID, http.StatusBadRequest, "invalid_request", "request_id, tenant_id, branch_id, service_id, and counter_id must be UUIDs")
		return
	}

	input := store.CallNextInput{
		RequestID: req.RequestID,
		TenantID:  req.TenantID,
		BranchID:  req.BranchID,
		ServiceID: req.ServiceID,
		CounterID: req.CounterID,
		CalledAt:  time.Now().UTC(),
	}

	ticket, _, err := h.store.CallNext(r.Context(), input)
	if err != nil {
		if errors.Is(err, store.ErrNoTicket) {
			writeError(w, req.RequestID, http.StatusConflict, "queue_empty", "no tickets available")
			return
		}
		status, code, msg := mapError(err)
		writeError(w, req.RequestID, status, code, msg)
		return
	}

	writeJSON(w, http.StatusOK, ticket)
}

func (h *Handler) handleTicketSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	branchID := strings.TrimSpace(r.URL.Query().Get("branch_id"))
	serviceID := strings.TrimSpace(r.URL.Query().Get("service_id"))
	if tenantID == "" || branchID == "" || serviceID == "" {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "tenant_id, branch_id, and service_id are required")
		return
	}
	if !isValidUUID(tenantID) || !isValidUUID(branchID) || !isValidUUID(serviceID) {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "tenant_id, branch_id, and service_id must be UUIDs")
		return
	}

	tickets, err := h.store.SnapshotTickets(r.Context(), tenantID, branchID, serviceID)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, "", status, code, msg)
		return
	}

	writeJSON(w, http.StatusOK, tickets)
}

func (h *Handler) handleAppointmentCheckin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		RequestID     string `json:"request_id"`
		TenantID      string `json:"tenant_id"`
		BranchID      string `json:"branch_id"`
		AppointmentID string `json:"appointment_id"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, payload.RequestID, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}
	payload.RequestID = strings.TrimSpace(payload.RequestID)
	payload.TenantID = strings.TrimSpace(payload.TenantID)
	payload.BranchID = strings.TrimSpace(payload.BranchID)
	payload.AppointmentID = strings.TrimSpace(payload.AppointmentID)
	if payload.RequestID == "" || payload.TenantID == "" || payload.BranchID == "" || payload.AppointmentID == "" {
		writeError(w, payload.RequestID, http.StatusBadRequest, "invalid_request", "request_id, tenant_id, branch_id, appointment_id are required")
		return
	}
	if !isValidUUID(payload.RequestID) || !isValidUUID(payload.TenantID) || !isValidUUID(payload.BranchID) || !isValidUUID(payload.AppointmentID) {
		writeError(w, payload.RequestID, http.StatusBadRequest, "invalid_request", "ids must be UUIDs")
		return
	}

	ticket, err := h.store.CheckInAppointment(r.Context(), payload.RequestID, payload.TenantID, payload.BranchID, payload.AppointmentID)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, payload.RequestID, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, ticket)
}

func (h *Handler) handleActiveTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	branchID := strings.TrimSpace(r.URL.Query().Get("branch_id"))
	counterID := strings.TrimSpace(r.URL.Query().Get("counter_id"))
	if tenantID == "" || branchID == "" || counterID == "" {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "tenant_id, branch_id, and counter_id are required")
		return
	}
	if !isValidUUID(tenantID) || !isValidUUID(branchID) || !isValidUUID(counterID) {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "tenant_id, branch_id, and counter_id must be UUIDs")
		return
	}

	ticket, found, err := h.store.GetActiveTicket(r.Context(), tenantID, branchID, counterID)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, "", status, code, msg)
		return
	}
	if !found {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeJSON(w, http.StatusOK, ticket)
}

func (h *Handler) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "tenant_id is required")
		return
	}
	if !isValidUUID(tenantID) {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "tenant_id must be a UUID")
		return
	}

	afterRaw := strings.TrimSpace(r.URL.Query().Get("after"))
	var after time.Time
	if afterRaw != "" {
		parsed, err := time.Parse(time.RFC3339, afterRaw)
		if err != nil {
			writeError(w, "", http.StatusBadRequest, "invalid_request", "after must be RFC3339 timestamp")
			return
		}
		after = parsed
	}

	limit := 100
	if limitRaw := strings.TrimSpace(r.URL.Query().Get("limit")); limitRaw != "" {
		parsed, err := strconv.Atoi(limitRaw)
		if err != nil || parsed <= 0 {
			writeError(w, "", http.StatusBadRequest, "invalid_request", "limit must be a positive integer")
			return
		}
		limit = parsed
	}

	events, err := h.store.ListOutboxEvents(r.Context(), tenantID, after, limit)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, "", status, code, msg)
		return
	}

	writeJSON(w, http.StatusOK, events)
}

func (h *Handler) handleServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	branchID := strings.TrimSpace(r.URL.Query().Get("branch_id"))
	if tenantID == "" || branchID == "" {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "tenant_id and branch_id are required")
		return
	}
	if !isValidUUID(tenantID) || !isValidUUID(branchID) {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "tenant_id and branch_id must be UUIDs")
		return
	}

	services, err := h.store.ListServices(r.Context(), tenantID, branchID)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, "", status, code, msg)
		return
	}

	writeJSON(w, http.StatusOK, services)
}

func (h *Handler) handleTicketActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/tickets/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[1] != "actions" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ticketID := parts[0]
	action := parts[2]
	if !isValidUUID(ticketID) {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "ticket_id must be a UUID")
		return
	}

	switch action {
	case "start":
		h.handleStartServing(w, r, ticketID)
	case "complete":
		h.handleCompleteTicket(w, r, ticketID)
	case "cancel":
		h.handleCancelTicket(w, r, ticketID)
	case "recall":
		h.handleRecallTicket(w, r, ticketID)
	case "hold":
		h.handleHoldTicket(w, r, ticketID)
	case "unhold":
		h.handleUnholdTicket(w, r, ticketID)
	case "transfer":
		h.handleTransferTicket(w, r, ticketID)
	case "no-show":
		h.handleNoShowTicket(w, r, ticketID)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

type ticketActionRequest struct {
	RequestID string `json:"request_id"`
	TenantID  string `json:"tenant_id"`
	BranchID  string `json:"branch_id"`
	CounterID string `json:"counter_id"`
}

type transferRequest struct {
	RequestID   string `json:"request_id"`
	TenantID    string `json:"tenant_id"`
	BranchID    string `json:"branch_id"`
	CounterID   string `json:"counter_id"`
	ToServiceID string `json:"to_service_id"`
}

func (h *Handler) handleStartServing(w http.ResponseWriter, r *http.Request, ticketID string) {
	var req ticketActionRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	req.CounterID = strings.TrimSpace(req.CounterID)
	if req.CounterID == "" {
		writeError(w, req.RequestID, http.StatusBadRequest, "invalid_request", "counter_id is required")
		return
	}
	if !isValidUUID(req.CounterID) {
		writeError(w, req.RequestID, http.StatusBadRequest, "invalid_request", "counter_id must be a UUID")
		return
	}

	ticket, _, err := h.store.StartServing(r.Context(), store.TicketActionInput{
		RequestID: req.RequestID,
		TenantID:  req.TenantID,
		BranchID:  req.BranchID,
		TicketID:  ticketID,
		CounterID: req.CounterID,
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, req.RequestID, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, ticket)
}

func (h *Handler) handleCompleteTicket(w http.ResponseWriter, r *http.Request, ticketID string) {
	var req ticketActionRequest
	if !decodeRequest(w, r, &req) {
		return
	}

	ticket, _, err := h.store.CompleteTicket(r.Context(), store.TicketActionInput{
		RequestID: req.RequestID,
		TenantID:  req.TenantID,
		BranchID:  req.BranchID,
		TicketID:  ticketID,
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, req.RequestID, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, ticket)
}

func (h *Handler) handleCancelTicket(w http.ResponseWriter, r *http.Request, ticketID string) {
	var req ticketActionRequest
	if !decodeRequest(w, r, &req) {
		return
	}

	ticket, _, err := h.store.CancelTicket(r.Context(), store.TicketActionInput{
		RequestID: req.RequestID,
		TenantID:  req.TenantID,
		BranchID:  req.BranchID,
		TicketID:  ticketID,
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, req.RequestID, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, ticket)
}

func (h *Handler) handleRecallTicket(w http.ResponseWriter, r *http.Request, ticketID string) {
	var req ticketActionRequest
	if !decodeRequest(w, r, &req) {
		return
	}

	ticket, _, err := h.store.RecallTicket(r.Context(), store.TicketActionInput{
		RequestID: req.RequestID,
		TenantID:  req.TenantID,
		BranchID:  req.BranchID,
		TicketID:  ticketID,
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, req.RequestID, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, ticket)
}

func (h *Handler) handleHoldTicket(w http.ResponseWriter, r *http.Request, ticketID string) {
	var req ticketActionRequest
	if !decodeRequest(w, r, &req) {
		return
	}

	ticket, _, err := h.store.HoldTicket(r.Context(), store.TicketActionInput{
		RequestID: req.RequestID,
		TenantID:  req.TenantID,
		BranchID:  req.BranchID,
		TicketID:  ticketID,
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, req.RequestID, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, ticket)
}

func (h *Handler) handleUnholdTicket(w http.ResponseWriter, r *http.Request, ticketID string) {
	var req ticketActionRequest
	if !decodeRequest(w, r, &req) {
		return
	}

	ticket, _, err := h.store.UnholdTicket(r.Context(), store.TicketActionInput{
		RequestID: req.RequestID,
		TenantID:  req.TenantID,
		BranchID:  req.BranchID,
		TicketID:  ticketID,
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, req.RequestID, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, ticket)
}

func (h *Handler) handleTransferTicket(w http.ResponseWriter, r *http.Request, ticketID string) {
	var req transferRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	req.ToServiceID = strings.TrimSpace(req.ToServiceID)
	if req.ToServiceID == "" {
		writeError(w, req.RequestID, http.StatusBadRequest, "invalid_request", "to_service_id is required")
		return
	}
	if !isValidUUID(req.ToServiceID) {
		writeError(w, req.RequestID, http.StatusBadRequest, "invalid_request", "to_service_id must be a UUID")
		return
	}

	ticket, _, err := h.store.TransferTicket(r.Context(), store.TicketActionInput{
		RequestID: req.RequestID,
		TenantID:  req.TenantID,
		BranchID:  req.BranchID,
		TicketID:  ticketID,
		ServiceID: req.ToServiceID,
		CounterID: req.CounterID,
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, req.RequestID, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, ticket)
}

func (h *Handler) handleNoShowTicket(w http.ResponseWriter, r *http.Request, ticketID string) {
	var req ticketActionRequest
	if !decodeRequest(w, r, &req) {
		return
	}

	ticket, _, err := h.store.NoShowTicket(r.Context(), store.TicketActionInput{
		RequestID: req.RequestID,
		TenantID:  req.TenantID,
		BranchID:  req.BranchID,
		TicketID:  ticketID,
		OccurredAt: time.Now().UTC(),
		ReturnToQueue: h.noShowReturnToQueue,
	})
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, req.RequestID, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, ticket)
}

func decodeRequest(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, "", http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return false
	}
	req, ok := target.(*ticketActionRequest)
	if ok {
		req.RequestID = strings.TrimSpace(req.RequestID)
		req.TenantID = strings.TrimSpace(req.TenantID)
		req.BranchID = strings.TrimSpace(req.BranchID)
		req.CounterID = strings.TrimSpace(req.CounterID)
	}
	tr, ok := target.(*transferRequest)
	if ok {
		tr.RequestID = strings.TrimSpace(tr.RequestID)
		tr.TenantID = strings.TrimSpace(tr.TenantID)
		tr.BranchID = strings.TrimSpace(tr.BranchID)
		tr.CounterID = strings.TrimSpace(tr.CounterID)
		tr.ToServiceID = strings.TrimSpace(tr.ToServiceID)
	}

	switch t := target.(type) {
	case *ticketActionRequest:
		if t.RequestID == "" || t.TenantID == "" || t.BranchID == "" {
			writeError(w, t.RequestID, http.StatusBadRequest, "invalid_request", "request_id, tenant_id, and branch_id are required")
			return false
		}
		if !isValidUUID(t.RequestID) || !isValidUUID(t.TenantID) || !isValidUUID(t.BranchID) {
			writeError(w, t.RequestID, http.StatusBadRequest, "invalid_request", "request_id, tenant_id, and branch_id must be UUIDs")
			return false
		}
	case *transferRequest:
		if t.RequestID == "" || t.TenantID == "" || t.BranchID == "" {
			writeError(w, t.RequestID, http.StatusBadRequest, "invalid_request", "request_id, tenant_id, and branch_id are required")
			return false
		}
		if !isValidUUID(t.RequestID) || !isValidUUID(t.TenantID) || !isValidUUID(t.BranchID) {
			writeError(w, t.RequestID, http.StatusBadRequest, "invalid_request", "request_id, tenant_id, and branch_id must be UUIDs")
			return false
		}
	default:
		writeError(w, "", http.StatusBadRequest, "invalid_request", "invalid request payload")
		return false
	}

	return true
}

func mapError(err error) (int, string, string) {
	switch {
	case errors.Is(err, store.ErrServiceNotFound):
		return http.StatusNotFound, "service_not_found", "service not found"
	case errors.Is(err, store.ErrTicketNotFound):
		return http.StatusNotFound, "ticket_not_found", "ticket not found"
	case errors.Is(err, store.ErrInvalidState):
		return http.StatusConflict, "invalid_state", "ticket state does not allow this action"
	case errors.Is(err, store.ErrCounterMismatch):
		return http.StatusConflict, "counter_mismatch", "ticket assigned to different counter"
	case errors.Is(err, store.ErrAccessDenied):
		return http.StatusForbidden, "access_denied", "access denied"
	case errors.Is(err, store.ErrBranchNotFound):
		return http.StatusNotFound, "branch_not_found", "branch not found"
	case errors.Is(err, store.ErrHolidayClosed):
		return http.StatusConflict, "holiday_closed", "appointments are closed for this holiday"
	default:
		return http.StatusInternalServerError, "internal_error", "internal server error"
	}
}

func writeError(w http.ResponseWriter, requestID string, status int, code, message string) {
	writeJSON(w, status, errorResponse{
		RequestID: requestID,
		Error: responseError{
			Code:    code,
			Message: message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

package httpapi

import (
	"encoding/json"
	"errors"
	"expvar"
	"net/http"
	"strconv"
	"strings"
	"time"

	"qms/queue-service/internal/store"

	"github.com/google/uuid"
)

type Handler struct {
	store               store.TicketStore
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
	RequestID string        `json:"request_id"`
	Error     responseError `json:"error"`
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
		store:               store,
		noShowReturnToQueue: options.NoShowReturnToQueue,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.handleHealth)
	mux.Handle("/metrics", expvar.Handler())
	mux.HandleFunc("/api/tickets", h.handleTickets)
	mux.HandleFunc("/api/tickets/actions/call-next", h.handleCallNext)
	mux.HandleFunc("/api/tickets/active", h.handleActiveTicket)
	mux.HandleFunc("/api/tickets/snapshot", h.handleTicketSnapshot)
	mux.HandleFunc("/api/tickets/", h.handleTicketActions)
	mux.HandleFunc("/api/queues", h.handleQueues)
	mux.HandleFunc("/api/appointments/checkin", h.handleAppointmentCheckin)
	mux.HandleFunc("/api/events", h.handleEvents)
	mux.HandleFunc("/api/counters", h.handleCounters)
	mux.HandleFunc("/api/counters/", h.handleCounterStatus)
	mux.HandleFunc("/api/services", h.handleServices)
	return AuthMiddleware(h.store, mux)
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

func normalizeCounterStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "available":
		return "available"
	case "busy":
		return "busy"
	case "break":
		return "break"
	case "active":
		return "active"
	default:
		return ""
	}
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
	if !requireTenant(w, r, req.TenantID) {
		return
	}
	if !requireBranchAccess(w, r, req.BranchID) {
		return
	}
	if !requireServiceAccess(w, r, req.ServiceID) {
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
	if !requireTenant(w, r, tenantID) {
		return
	}
	if !requireBranchAccess(w, r, branchID) {
		return
	}
	if !requireServiceAccess(w, r, serviceID) {
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
	if !requireTenant(w, r, payload.TenantID) {
		return
	}
	if !requireBranchAccess(w, r, payload.BranchID) {
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
	if !requireTenant(w, r, tenantID) {
		return
	}
	if !requireBranchAccess(w, r, branchID) {
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
	if !requireTenant(w, r, tenantID) {
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

func (h *Handler) handleCounters(w http.ResponseWriter, r *http.Request) {
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
	if !requireTenant(w, r, tenantID) {
		return
	}
	if !requireBranchAccess(w, r, branchID) {
		return
	}

	counters, err := h.store.ListCounters(r.Context(), tenantID, branchID)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, "", status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, counters)
}

func (h *Handler) handleCounterStatus(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/counters/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "status" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	counterID := parts[0]
	if !isValidUUID(counterID) {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "counter_id must be a UUID")
		return
	}

	var payload struct {
		TenantID string `json:"tenant_id"`
		BranchID string `json:"branch_id"`
		Status   string `json:"status"`
	}
	if !decodeRequest(w, r, &payload) {
		return
	}
	payload.Status = normalizeCounterStatus(payload.Status)
	if !isValidUUID(payload.TenantID) || !isValidUUID(payload.BranchID) {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "tenant_id and branch_id are required")
		return
	}
	if payload.Status == "" {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "status is required")
		return
	}
	if !requireTenant(w, r, payload.TenantID) {
		return
	}
	if !requireBranchAccess(w, r, payload.BranchID) {
		return
	}

	if err := h.store.UpdateCounterStatus(r.Context(), payload.TenantID, payload.BranchID, counterID, payload.Status); err != nil {
		status, code, msg := mapError(err)
		writeError(w, "", status, code, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	path := strings.TrimPrefix(r.URL.Path, "/api/tickets/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && r.Method == http.MethodGet {
		h.handleGetTicket(w, r, parts[0])
		return
	}
	if len(parts) < 2 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ticketID := parts[0]
	if !isValidUUID(ticketID) {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "ticket_id must be a UUID")
		return
	}

	if len(parts) == 2 && parts[1] == "events" {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		h.handleTicketEvents(w, r, ticketID)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(parts) != 3 || parts[1] != "actions" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	action := parts[2]

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

func (h *Handler) handleGetTicket(w http.ResponseWriter, r *http.Request, ticketID string) {
	if !isValidUUID(ticketID) {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "ticket_id must be a UUID")
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
	if !requireTenant(w, r, tenantID) {
		return
	}
	ticket, found, err := h.store.GetTicket(r.Context(), tenantID, branchID, ticketID)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, "", status, code, msg)
		return
	}
	if !found {
		writeError(w, "", http.StatusNotFound, "ticket_not_found", "ticket not found")
		return
	}
	writeJSON(w, http.StatusOK, ticket)
}

func (h *Handler) handleTicketEvents(w http.ResponseWriter, r *http.Request, ticketID string) {
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "tenant_id is required")
		return
	}
	if !isValidUUID(tenantID) {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "tenant_id must be a UUID")
		return
	}
	if !requireTenant(w, r, tenantID) {
		return
	}
	events, err := h.store.ListTicketEvents(r.Context(), tenantID, ticketID)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, "", status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *Handler) handleQueues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	branchID := strings.TrimSpace(r.URL.Query().Get("branch_id"))
	serviceID := strings.TrimSpace(r.URL.Query().Get("service_id"))
	if tenantID == "" || branchID == "" {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "tenant_id and branch_id are required")
		return
	}
	if !isValidUUID(tenantID) || !isValidUUID(branchID) {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "tenant_id and branch_id must be UUIDs")
		return
	}
	if serviceID != "" && !isValidUUID(serviceID) {
		writeError(w, "", http.StatusBadRequest, "invalid_request", "service_id must be a UUID when provided")
		return
	}
	if !requireTenant(w, r, tenantID) {
		return
	}
	if !requireBranchAccess(w, r, branchID) {
		return
	}
	if info, ok := accessFromContext(r.Context()); ok && len(info.Services) > 0 && serviceID == "" {
		writeError(w, "", http.StatusForbidden, "access_denied", "service access denied")
		return
	}
	if serviceID != "" && !requireServiceAccess(w, r, serviceID) {
		return
	}
	tickets, err := h.store.ListQueue(r.Context(), tenantID, branchID, serviceID)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, "", status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, tickets)
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
	Reason      string `json:"reason"`
}

func (h *Handler) handleStartServing(w http.ResponseWriter, r *http.Request, ticketID string) {
	var req ticketActionRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	if !requireTenant(w, r, req.TenantID) {
		return
	}
	if !requireBranchAccess(w, r, req.BranchID) {
		return
	}
	if !requireBranchAccess(w, r, req.BranchID) {
		return
	}
	if !requireBranchAccess(w, r, req.BranchID) {
		return
	}
	if !requireBranchAccess(w, r, req.BranchID) {
		return
	}
	if !requireBranchAccess(w, r, req.BranchID) {
		return
	}
	if !requireBranchAccess(w, r, req.BranchID) {
		return
	}
	if !requireBranchAccess(w, r, req.BranchID) {
		return
	}
	if !requireBranchAccess(w, r, req.BranchID) {
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
		RequestID:  req.RequestID,
		TenantID:   req.TenantID,
		BranchID:   req.BranchID,
		TicketID:   ticketID,
		CounterID:  req.CounterID,
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
	if !requireTenant(w, r, req.TenantID) {
		return
	}

	ticket, _, err := h.store.CompleteTicket(r.Context(), store.TicketActionInput{
		RequestID:  req.RequestID,
		TenantID:   req.TenantID,
		BranchID:   req.BranchID,
		TicketID:   ticketID,
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
	if !requireTenant(w, r, req.TenantID) {
		return
	}

	ticket, _, err := h.store.CancelTicket(r.Context(), store.TicketActionInput{
		RequestID:  req.RequestID,
		TenantID:   req.TenantID,
		BranchID:   req.BranchID,
		TicketID:   ticketID,
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
	if !requireTenant(w, r, req.TenantID) {
		return
	}

	ticket, _, err := h.store.RecallTicket(r.Context(), store.TicketActionInput{
		RequestID:  req.RequestID,
		TenantID:   req.TenantID,
		BranchID:   req.BranchID,
		TicketID:   ticketID,
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
	if !requireTenant(w, r, req.TenantID) {
		return
	}

	ticket, _, err := h.store.HoldTicket(r.Context(), store.TicketActionInput{
		RequestID:  req.RequestID,
		TenantID:   req.TenantID,
		BranchID:   req.BranchID,
		TicketID:   ticketID,
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
	if !requireTenant(w, r, req.TenantID) {
		return
	}

	ticket, _, err := h.store.UnholdTicket(r.Context(), store.TicketActionInput{
		RequestID:  req.RequestID,
		TenantID:   req.TenantID,
		BranchID:   req.BranchID,
		TicketID:   ticketID,
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
	if !requireTenant(w, r, req.TenantID) {
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
	if !requireServiceAccess(w, r, req.ToServiceID) {
		return
	}

	ticket, _, err := h.store.TransferTicket(r.Context(), store.TicketActionInput{
		RequestID:  req.RequestID,
		TenantID:   req.TenantID,
		BranchID:   req.BranchID,
		TicketID:   ticketID,
		ServiceID:  req.ToServiceID,
		Reason:     strings.TrimSpace(req.Reason),
		CounterID:  req.CounterID,
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
	if !requireTenant(w, r, req.TenantID) {
		return
	}

	ticket, _, err := h.store.NoShowTicket(r.Context(), store.TicketActionInput{
		RequestID:     req.RequestID,
		TenantID:      req.TenantID,
		BranchID:      req.BranchID,
		TicketID:      ticketID,
		OccurredAt:    time.Now().UTC(),
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
	case errors.Is(err, store.ErrCounterNotFound):
		return http.StatusNotFound, "counter_not_found", "counter not found"
	case errors.Is(err, store.ErrCounterUnavailable):
		return http.StatusConflict, "counter_unavailable", "counter unavailable"
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

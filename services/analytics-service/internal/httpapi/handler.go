package httpapi

import (
	"encoding/csv"
	"encoding/json"
	"expvar"
	"net/http"
	"strconv"
	"strings"
	"time"

	"qms/analytics-service/internal/store"

	"github.com/google/uuid"
)

type Handler struct {
	store store.Store
	opts  Options
}

type Options struct {
	BIAPIToken string
}

type errorResponse struct {
	Error responseError `json:"error"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewHandler(store store.Store, opts Options) *Handler {
	return &Handler{store: store, opts: opts}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", expvar.Handler())
	mux.HandleFunc("/healthz", h.handleHealth)
	mux.HandleFunc("/api/analytics/kpis", h.handleKPIs)
	mux.HandleFunc("/api/analytics/realtime", h.handleRealtime)
	mux.HandleFunc("/api/analytics/export", h.handleExport)
	mux.HandleFunc("/api/analytics/reports", h.handleReports)
	mux.HandleFunc("/api/analytics/anomalies", h.handleAnomalies)
	mux.HandleFunc("/api/analytics/bi/tickets", h.handleBITickets)
	mux.HandleFunc("/api/analytics/telemetry", h.handleTelemetry)
	return mux
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleKPIs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	params, ok := parseParams(w, r)
	if !ok {
		return
	}

	result, err := h.store.GetKPIs(r.Context(), params.tenantID, params.branchID, params.serviceID, params.from, params.to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleRealtime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	params, ok := parseParams(w, r)
	if !ok {
		return
	}

	result, err := h.store.GetRealtime(r.Context(), params.tenantID, params.branchID, params.serviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	params, ok := parseParams(w, r)
	if !ok {
		return
	}

	rows, err := h.store.ListTickets(r.Context(), params.tenantID, params.branchID, params.serviceID, params.from, params.to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=report.csv")
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"ticket_id", "ticket_number", "status", "created_at", "called_at", "served_at", "completed_at"})
	for _, row := range rows {
		_ = writer.Write([]string{
			row.TicketID,
			row.Number,
			row.Status,
			row.CreatedAt.Format(time.RFC3339),
			formatTime(row.CalledAt),
			formatTime(row.ServedAt),
			formatTime(row.CompletedAt),
		})
	}
	writer.Flush()
}

func (h *Handler) handleReports(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if !isValidUUID(tenantID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id is required")
			return
		}
		reports, err := h.store.ListScheduledReports(r.Context(), tenantID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, reports)
	case http.MethodPost:
		var payload struct {
			TenantID  string `json:"tenant_id"`
			BranchID  string `json:"branch_id"`
			ServiceID string `json:"service_id"`
			Cron      string `json:"cron"`
			Channel   string `json:"channel"`
			Recipient string `json:"recipient"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
			return
		}
		if !isValidUUID(payload.TenantID) || !isValidUUID(payload.BranchID) || !isValidUUID(payload.ServiceID) || payload.Cron == "" || payload.Channel == "" || payload.Recipient == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "missing required fields")
			return
		}
		if err := h.store.CreateScheduledReport(r.Context(), payload.TenantID, payload.BranchID, payload.ServiceID, payload.Cron, payload.Channel, payload.Recipient); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleAnomalies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if !isValidUUID(tenantID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id is required")
		return
	}
	anomalies, err := h.store.ListAnomalies(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, anomalies)
}

func (h *Handler) handleBITickets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	token := h.opts.BIAPIToken
	if token == "" {
		writeError(w, http.StatusForbidden, "bi_disabled", "BI connector is disabled")
		return
	}
	if !hasBIToken(r, token) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
		return
	}
	params, ok := parseParams(w, r)
	if !ok {
		return
	}
	tickets, err := h.store.ListTickets(r.Context(), params.tenantID, params.branchID, params.serviceID, params.from, params.to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, tickets)
}

func (h *Handler) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var payload map[string]interface{}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}
	_ = payload
	w.WriteHeader(http.StatusNoContent)
}

func hasBIToken(r *http.Request, token string) bool {
	header := strings.TrimSpace(r.Header.Get("X-BI-Token"))
	if header == token {
		return true
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")) == token
	}
	return false
}

type queryParams struct {
	tenantID  string
	branchID  string
	serviceID string
	from      time.Time
	to        time.Time
}

func parseParams(w http.ResponseWriter, r *http.Request) (queryParams, bool) {
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	branchID := strings.TrimSpace(r.URL.Query().Get("branch_id"))
	serviceID := strings.TrimSpace(r.URL.Query().Get("service_id"))
	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))

	if !isValidUUID(tenantID) || !isValidUUID(branchID) || !isValidUUID(serviceID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id, branch_id, service_id are required")
		return queryParams{}, false
	}

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	if fromRaw != "" {
		parsed, err := time.Parse(time.RFC3339, fromRaw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "from must be RFC3339")
			return queryParams{}, false
		}
		from = parsed
	}
	if toRaw != "" {
		parsed, err := time.Parse(time.RFC3339, toRaw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "to must be RFC3339")
			return queryParams{}, false
		}
		to = parsed
	}

	return queryParams{tenantID: tenantID, branchID: branchID, serviceID: serviceID, from: from, to: to}, true
}

func formatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: responseError{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func isValidUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

var _ = strconv.IntSize

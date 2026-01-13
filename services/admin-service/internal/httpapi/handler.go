package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"qms/admin-service/internal/models"
	"qms/admin-service/internal/store"

	"github.com/google/uuid"
)

type Handler struct {
	store store.Store
}

type errorResponse struct {
	Error responseError `json:"error"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewHandler(store store.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/admin/branches", h.handleBranches)
	mux.HandleFunc("/api/admin/branches/", h.handleBranch)
	mux.HandleFunc("/api/admin/areas", h.handleAreas)
	mux.HandleFunc("/api/admin/services", h.handleServices)
	mux.HandleFunc("/api/admin/services/", h.handleService)
	mux.HandleFunc("/api/admin/counters", h.handleCounters)
	mux.HandleFunc("/api/admin/counters/", h.handleCounterServices)
	mux.HandleFunc("/api/admin/policies/service", h.handleServicePolicy)
	mux.HandleFunc("/api/admin/devices", h.handleDevices)
	mux.HandleFunc("/api/admin/devices/", h.handleDeviceStatus)
	mux.HandleFunc("/api/admin/device-configs", h.handleDeviceConfigs)
	mux.HandleFunc("/api/devices/config", h.handleDeviceConfigFetch)
	mux.HandleFunc("/api/devices/status", h.handleDeviceStatusUpdate)
	mux.HandleFunc("/api/admin/audit", h.handleAudit)
	mux.HandleFunc("/api/admin/roles", h.handleRoles)
	mux.HandleFunc("/api/admin/users/", h.handleUserRole)
	mux.HandleFunc("/api/admin/holidays", h.handleHolidays)
	mux.HandleFunc("/api/admin/approvals", h.handleApprovals)
	mux.HandleFunc("/api/admin/approvals/prefs", h.handleApprovalPrefs)
	mux.HandleFunc("/api/admin/approvals/", h.handleApprovalAction)
	return mux
}

func (h *Handler) handleBranches(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionConfigWrite) && r.Method != http.MethodGet {
		return
	}
	if r.Method == http.MethodGet && !requirePermission(w, r, permissionConfigRead) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if !isValidUUID(tenantID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id is required")
			return
		}
		branches, err := h.store.ListBranches(r.Context(), tenantID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, branches)
	case http.MethodPost:
		var branch models.Branch
		if !decodeRequest(w, r, &branch) {
			return
		}
		if !isValidUUID(branch.TenantID) || branch.Name == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id and name are required")
			return
		}
		if h.maybeCreateApproval(w, r, branch.TenantID, "branch.create", branch) {
			return
		}
		created, err := h.store.CreateBranch(r.Context(), branch)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		h.recordAudit(r, branch.TenantID, "branch.create", "branch", created.BranchID)
		writeJSON(w, http.StatusOK, created)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleBranch(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionConfigWrite) {
		return
	}
	branchID := strings.TrimPrefix(r.URL.Path, "/api/admin/branches/")
	if !isValidUUID(branchID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "branch_id must be a UUID")
		return
	}
	if r.Method == http.MethodPut {
		var branch models.Branch
		if !decodeRequest(w, r, &branch) {
			return
		}
		if !isValidUUID(branch.TenantID) || branch.Name == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id and name are required")
			return
		}
		branch.BranchID = branchID
		if h.maybeCreateApproval(w, r, branch.TenantID, "branch.update", branch) {
			return
		}
		updated, err := h.store.UpdateBranch(r.Context(), branch)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		h.recordAudit(r, branch.TenantID, "branch.update", "branch", updated.BranchID)
		writeJSON(w, http.StatusOK, updated)
		return
	}
	if r.Method == http.MethodDelete {
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if !isValidUUID(tenantID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id is required")
			return
		}
		if h.maybeCreateApproval(w, r, tenantID, "branch.delete", map[string]string{"branch_id": branchID}) {
			return
		}
		err := h.store.DeleteBranch(r.Context(), tenantID, branchID)
		if err != nil {
			if errors.Is(err, store.ErrBranchHasServices) {
				writeError(w, http.StatusConflict, "branch_has_services", "branch has active services")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		h.recordAudit(r, tenantID, "branch.delete", "branch", branchID)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (h *Handler) handleAreas(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionConfigWrite) && r.Method != http.MethodGet {
		return
	}
	if r.Method == http.MethodGet && !requirePermission(w, r, permissionConfigRead) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		branchID := strings.TrimSpace(r.URL.Query().Get("branch_id"))
		if !isValidUUID(branchID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "branch_id is required")
			return
		}
		areas, err := h.store.ListAreas(r.Context(), branchID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, areas)
	case http.MethodPost:
		var area models.Area
		if !decodeRequest(w, r, &area) {
			return
		}
		if !isValidUUID(area.BranchID) || area.Name == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "branch_id and name are required")
			return
		}
		if h.maybeCreateApproval(w, r, "", "area.create", area) {
			return
		}
		created, err := h.store.CreateArea(r.Context(), area)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		h.recordAudit(r, "", "area.create", "area", created.AreaID)
		writeJSON(w, http.StatusOK, created)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleServices(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionConfigWrite) && r.Method != http.MethodGet {
		return
	}
	if r.Method == http.MethodGet && !requirePermission(w, r, permissionConfigRead) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		branchID := strings.TrimSpace(r.URL.Query().Get("branch_id"))
		if !isValidUUID(branchID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "branch_id is required")
			return
		}
		services, err := h.store.ListServices(r.Context(), branchID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, services)
	case http.MethodPost:
		var svc models.Service
		if !decodeRequest(w, r, &svc) {
			return
		}
		if !isValidUUID(svc.BranchID) || svc.Name == "" || svc.Code == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "branch_id, name, code are required")
			return
		}
		if svc.SLAMinutes <= 0 {
			svc.SLAMinutes = 5
		}
		if h.maybeCreateApproval(w, r, "", "service.create", svc) {
			return
		}
		created, err := h.store.CreateService(r.Context(), svc)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		h.recordAudit(r, "", "service.create", "service", created.ServiceID)
		writeJSON(w, http.StatusOK, created)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleService(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionConfigWrite) {
		return
	}
	serviceID := strings.TrimPrefix(r.URL.Path, "/api/admin/services/")
	if !isValidUUID(serviceID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "service_id must be a UUID")
		return
	}
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var svc models.Service
	if !decodeRequest(w, r, &svc) {
		return
	}
	if !isValidUUID(svc.BranchID) || svc.Name == "" || svc.Code == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "branch_id, name, code are required")
		return
	}
	if svc.SLAMinutes <= 0 {
		svc.SLAMinutes = 5
	}
	svc.ServiceID = serviceID
	if h.maybeCreateApproval(w, r, "", "service.update", svc) {
		return
	}
	updated, err := h.store.UpdateService(r.Context(), svc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	h.recordAudit(r, "", "service.update", "service", updated.ServiceID)
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) handleCounters(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionConfigWrite) && r.Method != http.MethodGet {
		return
	}
	if r.Method == http.MethodGet && !requirePermission(w, r, permissionConfigRead) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		branchID := strings.TrimSpace(r.URL.Query().Get("branch_id"))
		if !isValidUUID(branchID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "branch_id is required")
			return
		}
		counters, err := h.store.ListCounters(r.Context(), branchID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, counters)
	case http.MethodPost:
		var counter models.Counter
		if !decodeRequest(w, r, &counter) {
			return
		}
		if !isValidUUID(counter.BranchID) || counter.Name == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "branch_id and name are required")
			return
		}
		if counter.Status == "" {
			counter.Status = "active"
		}
		if h.maybeCreateApproval(w, r, "", "counter.create", counter) {
			return
		}
		created, err := h.store.CreateCounter(r.Context(), counter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		h.recordAudit(r, "", "counter.create", "counter", created.CounterID)
		writeJSON(w, http.StatusOK, created)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleCounterServices(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionConfigWrite) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/counters/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "services" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	counterID := parts[0]
	if !isValidUUID(counterID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "counter_id must be a UUID")
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		ServiceID string `json:"service_id"`
	}
	if !decodeRequest(w, r, &payload) {
		return
	}
	if !isValidUUID(payload.ServiceID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "service_id must be a UUID")
		return
	}
	if h.maybeCreateApproval(w, r, "", "counter.map_service", map[string]string{"counter_id": counterID, "service_id": payload.ServiceID}) {
		return
	}
	if err := h.store.MapCounterService(r.Context(), counterID, payload.ServiceID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	h.recordAudit(r, "", "counter.map_service", "counter", counterID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleServicePolicy(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionConfigWrite) && r.Method != http.MethodGet {
		return
	}
	if r.Method == http.MethodGet && !requirePermission(w, r, permissionConfigRead) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		branchID := strings.TrimSpace(r.URL.Query().Get("branch_id"))
		serviceID := strings.TrimSpace(r.URL.Query().Get("service_id"))
		if !isValidUUID(tenantID) || !isValidUUID(branchID) || !isValidUUID(serviceID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id, branch_id, service_id are required")
			return
		}
		policy, found, err := h.store.GetServicePolicy(r.Context(), tenantID, branchID, serviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		if !found {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, http.StatusOK, policy)
	case http.MethodPost:
		var policy models.ServicePolicy
		if !decodeRequest(w, r, &policy) {
			return
		}
		if !isValidUUID(policy.TenantID) || !isValidUUID(policy.BranchID) || !isValidUUID(policy.ServiceID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id, branch_id, service_id are required")
			return
		}
		if policy.NoShowGraceSeconds <= 0 {
			policy.NoShowGraceSeconds = 300
		}
		if h.maybeCreateApproval(w, r, policy.TenantID, "policy.update", policy) {
			return
		}
		updated, err := h.store.UpsertServicePolicy(r.Context(), policy)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		h.recordAudit(r, policy.TenantID, "policy.update", "service_policy", policy.ServiceID)
		writeJSON(w, http.StatusOK, updated)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleDevices(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionConfigWrite) && r.Method != http.MethodGet {
		return
	}
	if r.Method == http.MethodGet && !requirePermission(w, r, permissionConfigRead) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if !isValidUUID(tenantID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id is required")
			return
		}
		devices, err := h.store.ListDevices(r.Context(), tenantID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, devices)
	case http.MethodPost:
		var device models.Device
		if !decodeRequest(w, r, &device) {
			return
		}
		if !isValidUUID(device.TenantID) || !isValidUUID(device.BranchID) || device.Type == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id, branch_id, type are required")
			return
		}
		if device.Status == "" {
			device.Status = "offline"
		}
		if h.maybeCreateApproval(w, r, device.TenantID, "device.register", device) {
			return
		}
		created, err := h.store.RegisterDevice(r.Context(), device)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		h.recordAudit(r, device.TenantID, "device.register", "device", created.DeviceID)
		writeJSON(w, http.StatusOK, created)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleDeviceStatus(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionConfigWrite) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/devices/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "status" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	deviceID := parts[0]
	if !isValidUUID(deviceID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "device_id must be a UUID")
		return
	}
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Status string `json:"status"`
	}
	if !decodeRequest(w, r, &payload) {
		return
	}
	if payload.Status == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "status is required")
		return
	}
	if err := h.store.UpdateDeviceStatus(r.Context(), deviceID, payload.Status); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	h.recordAudit(r, "", "device.status", "device", deviceID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleDeviceConfigs(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionConfigWrite) {
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		DeviceID string          `json:"device_id"`
		Version  int             `json:"version"`
		Payload  json.RawMessage `json:"payload"`
	}
	if !decodeRequest(w, r, &payload) {
		return
	}
	if !isValidUUID(payload.DeviceID) || payload.Version <= 0 || len(payload.Payload) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "device_id, version, payload are required")
		return
	}
	if h.maybeCreateApproval(w, r, "", "device.config", payload) {
		return
	}
	if err := h.store.CreateDeviceConfig(r.Context(), payload.DeviceID, payload.Version, string(payload.Payload)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	h.recordAudit(r, "", "device.config", "device", payload.DeviceID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleDeviceConfigFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	if !isValidUUID(deviceID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "device_id is required")
		return
	}
	version, payload, err := h.store.GetLatestDeviceConfig(r.Context(), deviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	if version == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"device_id": deviceID,
		"version":   version,
		"payload":   json.RawMessage(payload),
	})
}

func (h *Handler) handleDeviceStatusUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		DeviceID string `json:"device_id"`
		Status   string `json:"status"`
	}
	if !decodeRequest(w, r, &payload) {
		return
	}
	if !isValidUUID(payload.DeviceID) || payload.Status == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "device_id and status are required")
		return
	}
	if err := h.store.UpdateDeviceStatus(r.Context(), payload.DeviceID, payload.Status); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleAudit(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionAuditRead) {
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	actionType := strings.TrimSpace(r.URL.Query().Get("action_type"))
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if !isValidUUID(tenantID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id is required")
		return
	}
	entries, err := h.store.ListAudit(r.Context(), tenantID, actionType, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *Handler) handleRoles(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionRolesManage) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if !isValidUUID(tenantID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id is required")
			return
		}
		roles, err := h.store.ListRoles(r.Context(), tenantID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, roles)
	case http.MethodPost:
		var role models.Role
		if !decodeRequest(w, r, &role) {
			return
		}
		if !isValidUUID(role.TenantID) || role.Name == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id and name are required")
			return
		}
		created, err := h.store.CreateRole(r.Context(), role)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		h.recordAudit(r, role.TenantID, "role.create", "role", created.RoleID)
		writeJSON(w, http.StatusOK, created)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleUserRole(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionRolesManage) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	userID := parts[0]
	if !isValidUUID(userID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "user_id must be a UUID")
		return
	}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if !isValidUUID(tenantID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id is required")
			return
		}
		user, found, err := h.store.GetUser(r.Context(), tenantID, userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		if !found {
			writeError(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		writeJSON(w, http.StatusOK, user)
		return
	}
	if len(parts) != 2 || parts[1] != "role" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		TenantID string `json:"tenant_id"`
		RoleID   string `json:"role_id"`
	}
	if !decodeRequest(w, r, &payload) {
		return
	}
	if !isValidUUID(payload.TenantID) || !isValidUUID(payload.RoleID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id and role_id are required")
		return
	}
	if err := h.store.UpdateUserRole(r.Context(), payload.TenantID, userID, payload.RoleID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	h.recordAudit(r, payload.TenantID, "user.role_update", "user", userID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleHolidays(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionConfigWrite) && r.Method != http.MethodGet {
		return
	}
	if r.Method == http.MethodGet && !requirePermission(w, r, permissionConfigRead) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		branchID := strings.TrimSpace(r.URL.Query().Get("branch_id"))
		if !isValidUUID(tenantID) || !isValidUUID(branchID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id and branch_id are required")
			return
		}
		holidays, err := h.store.ListHolidays(r.Context(), tenantID, branchID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, holidays)
	case http.MethodPost:
		var holiday models.Holiday
		if !decodeRequest(w, r, &holiday) {
			return
		}
		if !isValidUUID(holiday.TenantID) || !isValidUUID(holiday.BranchID) || holiday.Date == "" || holiday.Name == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id, branch_id, date, name are required")
			return
		}
		if h.maybeCreateApproval(w, r, holiday.TenantID, "holiday.create", holiday) {
			return
		}
		created, err := h.store.CreateHoliday(r.Context(), holiday)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		h.recordAudit(r, holiday.TenantID, "holiday.create", "holiday", created.HolidayID)
		writeJSON(w, http.StatusOK, created)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleApprovals(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionApprovalManage) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		status := strings.TrimSpace(r.URL.Query().Get("status"))
		if !isValidUUID(tenantID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id is required")
			return
		}
		approvals, err := h.store.ListApprovals(r.Context(), tenantID, status)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, approvals)
	case http.MethodPost:
		var approval models.ApprovalRequest
		if !decodeRequest(w, r, &approval) {
			return
		}
		if !isValidUUID(approval.TenantID) || approval.RequestType == "" || approval.Payload == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id, request_type, payload are required")
			return
		}
		created, err := h.store.CreateApproval(r.Context(), approval)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		h.recordAudit(r, approval.TenantID, "approval.request", "approval", created.ApprovalID)
		writeJSON(w, http.StatusOK, created)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleApprovalPrefs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !requirePermission(w, r, permissionConfigRead) {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if !isValidUUID(tenantID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id is required")
			return
		}
		enabled, err := h.store.GetApprovalPrefs(r.Context(), tenantID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"tenant_id": tenantID,
			"approvals_enabled": enabled,
		})
	case http.MethodPost:
		if !requirePermission(w, r, permissionConfigWrite) {
			return
		}
		var payload struct {
			TenantID         string `json:"tenant_id"`
			ApprovalsEnabled bool   `json:"approvals_enabled"`
		}
		if !decodeRequest(w, r, &payload) {
			return
		}
		if !isValidUUID(payload.TenantID) {
			writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id is required")
			return
		}
		if err := h.store.SetApprovalPrefs(r.Context(), payload.TenantID, payload.ApprovalsEnabled); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		h.recordAudit(r, payload.TenantID, "approval.prefs_update", "tenant", payload.TenantID)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"tenant_id": payload.TenantID,
			"approvals_enabled": payload.ApprovalsEnabled,
		})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleApprovalAction(w http.ResponseWriter, r *http.Request) {
	if !requirePermission(w, r, permissionApprovalManage) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/approvals/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "approve" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	approvalID := parts[0]
	if !isValidUUID(approvalID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "approval_id must be a UUID")
		return
	}
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	approverID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if err := h.store.ApproveRequest(r.Context(), approvalID, approverID); err != nil {
		if errors.Is(err, store.ErrApprovalNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "approval request not found")
			return
		}
		if errors.Is(err, store.ErrApprovalNotPending) {
			writeError(w, http.StatusConflict, "already_processed", "approval request is not pending")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	if err := h.applyApproval(r.Context(), approvalID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "apply approval failed")
		return
	}
	h.recordAudit(r, "", "approval.approve", "approval", approvalID)
	w.WriteHeader(http.StatusNoContent)
}

func decodeRequest(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return false
	}
	return true
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

type permission string

const (
	permissionConfigRead   permission = "config.read"
	permissionConfigWrite  permission = "config.write"
	permissionAuditRead    permission = "audit.read"
	permissionRolesManage  permission = "roles.manage"
	permissionApprovalManage permission = "approval.manage"
)

func requirePermission(w http.ResponseWriter, r *http.Request, perm permission) bool {
	role := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Role")))
	if role == "" {
		role = "agent"
	}
	if hasPermission(role, perm) {
		return true
	}
	writeError(w, http.StatusForbidden, "access_denied", "insufficient role")
	return false
}

func hasPermission(role string, perm permission) bool {
	switch role {
	case "admin":
		return true
	case "supervisor":
		switch perm {
		case permissionConfigRead, permissionConfigWrite, permissionAuditRead, permissionApprovalManage:
			return true
		default:
			return false
		}
	case "agent":
		return false
	default:
		return false
	}
}

func (h *Handler) maybeCreateApproval(w http.ResponseWriter, r *http.Request, tenantID, reqType string, payload interface{}) bool {
	if tenantID == "" {
		tenantID = strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	}
	if !isValidUUID(tenantID) {
		return false
	}
	enabled, err := h.store.ApprovalsEnabled(r.Context(), tenantID)
	if err != nil || !enabled {
		return false
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "approval marshal failed")
		return true
	}
	approval := models.ApprovalRequest{
		TenantID:    tenantID,
		RequestType: reqType,
		Payload:     string(raw),
		CreatedBy:   strings.TrimSpace(r.Header.Get("X-User-ID")),
	}
	created, err := h.store.CreateApproval(r.Context(), approval)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "approval create failed")
		return true
	}
	writeJSON(w, http.StatusAccepted, created)
	return true
}

func (h *Handler) applyApproval(ctx context.Context, approvalID string) error {
	approval, found, err := h.store.GetApproval(ctx, approvalID)
	if err != nil || !found {
		return err
	}
	switch approval.RequestType {
	case "branch.create":
		var branch models.Branch
		if err := json.Unmarshal([]byte(approval.Payload), &branch); err != nil {
			return err
		}
		_, err = h.store.CreateBranch(ctx, branch)
		return err
	case "branch.update":
		var branch models.Branch
		if err := json.Unmarshal([]byte(approval.Payload), &branch); err != nil {
			return err
		}
		_, err = h.store.UpdateBranch(ctx, branch)
		return err
	case "branch.delete":
		var payload map[string]string
		if err := json.Unmarshal([]byte(approval.Payload), &payload); err != nil {
			return err
		}
		return h.store.DeleteBranch(ctx, approval.TenantID, payload["branch_id"])
	case "area.create":
		var area models.Area
		if err := json.Unmarshal([]byte(approval.Payload), &area); err != nil {
			return err
		}
		_, err = h.store.CreateArea(ctx, area)
		return err
	case "service.create":
		var svc models.Service
		if err := json.Unmarshal([]byte(approval.Payload), &svc); err != nil {
			return err
		}
		_, err = h.store.CreateService(ctx, svc)
		return err
	case "service.update":
		var svc models.Service
		if err := json.Unmarshal([]byte(approval.Payload), &svc); err != nil {
			return err
		}
		_, err = h.store.UpdateService(ctx, svc)
		return err
	case "counter.create":
		var counter models.Counter
		if err := json.Unmarshal([]byte(approval.Payload), &counter); err != nil {
			return err
		}
		_, err = h.store.CreateCounter(ctx, counter)
		return err
	case "counter.map_service":
		var payload map[string]string
		if err := json.Unmarshal([]byte(approval.Payload), &payload); err != nil {
			return err
		}
		return h.store.MapCounterService(ctx, payload["counter_id"], payload["service_id"])
	case "policy.update":
		var policy models.ServicePolicy
		if err := json.Unmarshal([]byte(approval.Payload), &policy); err != nil {
			return err
		}
		_, err = h.store.UpsertServicePolicy(ctx, policy)
		return err
	case "device.register":
		var device models.Device
		if err := json.Unmarshal([]byte(approval.Payload), &device); err != nil {
			return err
		}
		_, err = h.store.RegisterDevice(ctx, device)
		return err
	case "device.config":
		var payload struct {
			DeviceID string          `json:"device_id"`
			Version  int             `json:"version"`
			Payload  json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal([]byte(approval.Payload), &payload); err != nil {
			return err
		}
		return h.store.CreateDeviceConfig(ctx, payload.DeviceID, payload.Version, string(payload.Payload))
	case "holiday.create":
		var holiday models.Holiday
		if err := json.Unmarshal([]byte(approval.Payload), &holiday); err != nil {
			return err
		}
		_, err = h.store.CreateHoliday(ctx, holiday)
		return err
	default:
		return nil
	}
}

func (h *Handler) recordAudit(r *http.Request, tenantID, actionType, targetType, targetID string) {
	if tenantID == "" {
		tenantID = strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	}
	if !isValidUUID(tenantID) {
		return
	}
	_ = h.store.InsertAudit(r.Context(), models.AuditLog{
		TenantID:    tenantID,
		ActorUserID: strings.TrimSpace(r.Header.Get("X-User-ID")),
		ActionType:  actionType,
		TargetType:  targetType,
		TargetID:    targetID,
		IP:          r.RemoteAddr,
		UserAgent:   r.UserAgent(),
	})
}

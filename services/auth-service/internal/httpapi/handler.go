package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"qms/auth-service/internal/store"

	"github.com/google/uuid"
)

type Handler struct {
	store store.Store
}

type loginRequest struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	BranchID string `json:"branch_id"`
}

type loginResponse struct {
	SessionID string   `json:"session_id"`
	ExpiresAt string   `json:"expires_at"`
	User      userInfo `json:"user"`
	Branches  []string `json:"branches"`
	Services  []string `json:"services"`
}

type userInfo struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	Email    string `json:"email"`
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
	mux.HandleFunc("/api/auth/login", h.handleLogin)
	mux.HandleFunc("/api/auth/sso", h.handleSSO)
	mux.HandleFunc("/api/auth/me", h.handleMe)
	return mux
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	req.TenantID = strings.TrimSpace(req.TenantID)
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	req.BranchID = strings.TrimSpace(req.BranchID)

	if req.TenantID == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id, email, and password are required")
		return
	}
	if !isValidUUID(req.TenantID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id must be a UUID")
		return
	}
	if req.BranchID != "" && !isValidUUID(req.BranchID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "branch_id must be a UUID")
		return
	}

	result, err := h.store.Login(r.Context(), store.LoginInput{
		TenantID: req.TenantID,
		Email:    req.Email,
		Password: req.Password,
		BranchID: req.BranchID,
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
		case errors.Is(err, store.ErrAccessDenied):
			writeError(w, http.StatusForbidden, "access_denied", "access to branch denied")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	resp := loginResponse{
		SessionID: result.Session.SessionID,
		ExpiresAt: result.Session.ExpiresAt.Format(time.RFC3339),
		User: userInfo{
			UserID:   result.User.UserID,
			TenantID: result.User.TenantID,
			Role:     result.User.RoleName,
			Email:    result.User.Email,
		},
		Branches: result.Branches,
		Services: result.Services,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleSSO(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		TenantID string `json:"tenant_id"`
		Provider string `json:"provider"`
		Subject  string `json:"subject"`
		Email    string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	payload.TenantID = strings.TrimSpace(payload.TenantID)
	payload.Provider = strings.TrimSpace(payload.Provider)
	payload.Subject = strings.TrimSpace(payload.Subject)
	payload.Email = strings.TrimSpace(payload.Email)
	if payload.TenantID == "" || payload.Provider == "" || payload.Subject == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id, provider, subject are required")
		return
	}
	if !isValidUUID(payload.TenantID) {
		writeError(w, http.StatusBadRequest, "invalid_request", "tenant_id must be a UUID")
		return
	}

	result, err := h.store.SSOLogin(r.Context(), payload.TenantID, payload.Provider, payload.Subject, payload.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	resp := loginResponse{
		SessionID: result.Session.SessionID,
		ExpiresAt: result.Session.ExpiresAt.Format(time.RFC3339),
		User: userInfo{
			UserID:   result.User.UserID,
			TenantID: result.User.TenantID,
			Role:     result.User.RoleName,
			Email:    result.User.Email,
		},
		Branches: result.Branches,
		Services: result.Services,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := strings.TrimSpace(bearerToken(r.Header.Get("Authorization")))
	if sessionID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing session token")
		return
	}

	session, user, err := h.store.GetSession(r.Context(), sessionID)
	if err != nil {
		if errors.Is(err, store.ErrSessionNotFound) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid session")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	branches, services, err := h.store.GetAccess(r.Context(), user.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	resp := loginResponse{
		SessionID: session.SessionID,
		ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
		User: userInfo{
			UserID:   user.UserID,
			TenantID: user.TenantID,
			Role:     user.RoleName,
			Email:    user.Email,
		},
		Branches: branches,
		Services: services,
	}

	writeJSON(w, http.StatusOK, resp)
}

func bearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Fields(header)
	if len(parts) != 2 {
		return ""
	}
	if strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
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

package httpapi

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"expvar"
	"net/http"
	"os"
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

type samlRequest struct {
	TenantID  string `json:"tenant_id"`
	Assertion string `json:"assertion"`
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
	RequestID string        `json:"request_id"`
	Error     responseError `json:"error"`
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
	mux.Handle("/metrics", expvar.Handler())
	mux.HandleFunc("/healthz", h.handleHealth)
	mux.HandleFunc("/api/auth/login", h.handleLogin)
	mux.HandleFunc("/api/auth/sso", h.handleSSO)
	mux.HandleFunc("/api/auth/sso/jwt", h.handleJWTSSO)
	mux.HandleFunc("/api/auth/sso/saml", h.handleSAMLSSO)
	mux.HandleFunc("/api/auth/me", h.handleMe)
	return mux
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
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
		writeError(w, r, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	req.TenantID = strings.TrimSpace(req.TenantID)
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	req.BranchID = strings.TrimSpace(req.BranchID)

	if req.TenantID == "" || req.Email == "" || req.Password == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "tenant_id, email, and password are required")
		return
	}
	if !isValidUUID(req.TenantID) {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "tenant_id must be a UUID")
		return
	}
	if req.BranchID != "" && !isValidUUID(req.BranchID) {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "branch_id must be a UUID")
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
			writeError(w, r, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
		case errors.Is(err, store.ErrAccessDenied):
			writeError(w, r, http.StatusForbidden, "access_denied", "access to branch denied")
		default:
			writeError(w, r, http.StatusInternalServerError, "internal_error", "internal server error")
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
		writeError(w, r, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	payload.TenantID = strings.TrimSpace(payload.TenantID)
	payload.Provider = strings.TrimSpace(payload.Provider)
	payload.Subject = strings.TrimSpace(payload.Subject)
	payload.Email = strings.TrimSpace(payload.Email)
	if payload.TenantID == "" || payload.Provider == "" || payload.Subject == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "tenant_id, provider, subject are required")
		return
	}
	if !isValidUUID(payload.TenantID) {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "tenant_id must be a UUID")
		return
	}

	result, err := h.store.SSOLogin(r.Context(), payload.TenantID, payload.Provider, payload.Subject, payload.Email)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "internal server error")
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

func (h *Handler) handleJWTSSO(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		TenantID string `json:"tenant_id"`
		Token    string `json:"token"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	payload.TenantID = strings.TrimSpace(payload.TenantID)
	payload.Token = strings.TrimSpace(payload.Token)
	if payload.TenantID == "" || payload.Token == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "tenant_id and token are required")
		return
	}
	if !isValidUUID(payload.TenantID) {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "tenant_id must be a UUID")
		return
	}

	secret := os.Getenv("AUTH_SSO_JWT_SECRET")
	if secret == "" {
		writeError(w, r, http.StatusInternalServerError, "config_missing", "AUTH_SSO_JWT_SECRET is not set")
		return
	}
	issuer := strings.TrimSpace(os.Getenv("AUTH_SSO_JWT_ISSUER"))
	subject, email, err := validateJWT(payload.Token, []byte(secret), issuer)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "invalid_token", "invalid SSO token")
		return
	}

	result, err := h.store.SSOLogin(r.Context(), payload.TenantID, "jwt", subject, email)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "internal server error")
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

func (h *Handler) handleSAMLSSO(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req samlRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	req.TenantID = strings.TrimSpace(req.TenantID)
	req.Assertion = strings.TrimSpace(req.Assertion)
	if !isValidUUID(req.TenantID) || req.Assertion == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "tenant_id and assertion are required")
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(req.Assertion)
	if err != nil {
		decoded = []byte(req.Assertion)
	}

	nameID, err := extractSAMLNameID(decoded)
	if err != nil || nameID == "" {
		writeError(w, r, http.StatusUnauthorized, "invalid_assertion", "invalid SAML assertion")
		return
	}

	result, err := h.store.SSOLogin(r.Context(), req.TenantID, "saml", nameID, nameID)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "invalid_credentials", "invalid SSO credentials")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func extractSAMLNameID(data []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", err
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if strings.EqualFold(start.Name.Local, "NameID") {
			var value string
			if err := decoder.DecodeElement(&value, &start); err != nil {
				return "", err
			}
			return strings.TrimSpace(value), nil
		}
	}
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := strings.TrimSpace(bearerToken(r.Header.Get("Authorization")))
	if sessionID == "" {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "missing session token")
		return
	}

	session, user, err := h.store.GetSession(r.Context(), sessionID)
	if err != nil {
		if errors.Is(err, store.ErrSessionNotFound) {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "invalid session")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	branches, services, err := h.store.GetAccess(r.Context(), user.UserID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "internal server error")
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

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	requestID := ""
	if r != nil {
		requestID = strings.TrimSpace(r.Header.Get("X-Request-ID"))
	}
	writeJSON(w, status, errorResponse{RequestID: requestID, Error: responseError{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func validateJWT(token string, secret []byte, issuer string) (string, string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", "", errors.New("invalid token")
	}
	signed := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", "", err
	}
	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write([]byte(signed)); err != nil {
		return "", "", err
	}
	expected := mac.Sum(nil)
	if !hmac.Equal(signature, expected) {
		return "", "", errors.New("signature mismatch")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", err
	}
	var claims struct {
		Subject string `json:"sub"`
		Email   string `json:"email"`
		Issuer  string `json:"iss"`
		Expires int64  `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return "", "", err
	}
	if claims.Subject == "" {
		return "", "", errors.New("missing subject")
	}
	if issuer != "" && claims.Issuer != issuer {
		return "", "", errors.New("issuer mismatch")
	}
	if claims.Expires > 0 && time.Now().Unix() > claims.Expires {
		return "", "", errors.New("token expired")
	}
	return claims.Subject, claims.Email, nil
}

func isValidUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

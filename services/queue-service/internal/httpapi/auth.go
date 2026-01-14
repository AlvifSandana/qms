package httpapi

import (
	"context"
	"net/http"
	"strings"

	"qms/queue-service/internal/store"
)

type authContextKey struct{}

type authInfo struct {
	Session  store.Session
	Branches []string
	Services []string
}

func AuthMiddleware(store store.TicketStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicEndpoint(r) {
			next.ServeHTTP(w, r)
			return
		}
		sessionID := sessionIDFromRequest(r)
		if sessionID == "" {
			writeError(w, requestIDFromRequest(r), http.StatusUnauthorized, "unauthorized", "missing session")
			return
		}
		session, err := store.GetSession(r.Context(), sessionID)
		if err != nil {
			if err == store.ErrSessionNotFound {
				writeError(w, requestIDFromRequest(r), http.StatusUnauthorized, "unauthorized", "invalid session")
				return
			}
			writeError(w, requestIDFromRequest(r), http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		branches, services, err := store.GetAccess(r.Context(), session.UserID)
		if err != nil {
			writeError(w, requestIDFromRequest(r), http.StatusInternalServerError, "internal_error", "access lookup failed")
			return
		}
		ctx := context.WithValue(r.Context(), authContextKey{}, authInfo{Session: session, Branches: branches, Services: services})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func sessionFromContext(ctx context.Context) (store.Session, bool) {
	value := ctx.Value(authContextKey{})
	if value == nil {
		return store.Session{}, false
	}
	info, ok := value.(authInfo)
	if !ok {
		return store.Session{}, false
	}
	return info.Session, true
}

func accessFromContext(ctx context.Context) (authInfo, bool) {
	value := ctx.Value(authContextKey{})
	if value == nil {
		return authInfo{}, false
	}
	info, ok := value.(authInfo)
	if !ok {
		return authInfo{}, false
	}
	return info, true
}

func requireTenant(w http.ResponseWriter, r *http.Request, tenantID string) bool {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		writeError(w, requestIDFromRequest(r), http.StatusUnauthorized, "unauthorized", "missing session")
		return false
	}
	if tenantID == "" {
		writeError(w, requestIDFromRequest(r), http.StatusBadRequest, "invalid_request", "tenant_id is required")
		return false
	}
	if session.TenantID != tenantID {
		writeError(w, requestIDFromRequest(r), http.StatusForbidden, "access_denied", "tenant access denied")
		return false
	}
	return true
}

func requireBranchAccess(w http.ResponseWriter, r *http.Request, branchID string) bool {
	info, ok := accessFromContext(r.Context())
	if !ok {
		writeError(w, requestIDFromRequest(r), http.StatusUnauthorized, "unauthorized", "missing session")
		return false
	}
	if branchID == "" {
		writeError(w, requestIDFromRequest(r), http.StatusBadRequest, "invalid_request", "branch_id is required")
		return false
	}
	if len(info.Branches) == 0 {
		return true
	}
	if !contains(info.Branches, branchID) {
		writeError(w, requestIDFromRequest(r), http.StatusForbidden, "access_denied", "branch access denied")
		return false
	}
	return true
}

func requireServiceAccess(w http.ResponseWriter, r *http.Request, serviceID string) bool {
	info, ok := accessFromContext(r.Context())
	if !ok {
		writeError(w, requestIDFromRequest(r), http.StatusUnauthorized, "unauthorized", "missing session")
		return false
	}
	if serviceID == "" {
		writeError(w, requestIDFromRequest(r), http.StatusBadRequest, "invalid_request", "service_id is required")
		return false
	}
	if len(info.Services) == 0 {
		return true
	}
	if !contains(info.Services, serviceID) {
		writeError(w, requestIDFromRequest(r), http.StatusForbidden, "access_denied", "service access denied")
		return false
	}
	return true
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func sessionIDFromRequest(r *http.Request) string {
	if token := bearerToken(r.Header.Get("Authorization")); token != "" {
		return token
	}
	return strings.TrimSpace(r.Header.Get("X-Session-ID"))
}

func requestIDFromRequest(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Request-ID"))
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

func isPublicEndpoint(r *http.Request) bool {
	switch r.URL.Path {
	case "/healthz", "/metrics":
		return true
	case "/api/tickets":
		return r.Method == http.MethodPost
	case "/api/services":
		return r.Method == http.MethodGet
	default:
		return r.Method == http.MethodOptions
	}
}

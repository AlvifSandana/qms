package httpapi

import (
	"context"
	"net/http"
	"strings"

	"qms/admin-service/internal/store"
)

type authContextKey struct{}

type authInfo struct {
	Session store.Session
}

func AuthMiddleware(store store.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicEndpoint(r) {
			next.ServeHTTP(w, r)
			return
		}
		sessionID := sessionIDFromRequest(r)
		if sessionID == "" {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "missing session")
			return
		}
		session, err := store.GetSession(r.Context(), sessionID)
		if err != nil {
			if err == store.ErrSessionNotFound {
				writeError(w, r, http.StatusUnauthorized, "unauthorized", "invalid session")
				return
			}
			writeError(w, r, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		ctx := context.WithValue(r.Context(), authContextKey{}, authInfo{Session: session})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func authFromContext(ctx context.Context) (store.Session, bool) {
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

func requireTenant(w http.ResponseWriter, r *http.Request, tenantID string) bool {
	session, ok := authFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "missing session")
		return false
	}
	if tenantID == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "tenant_id is required")
		return false
	}
	if session.TenantID != tenantID {
		writeError(w, r, http.StatusForbidden, "access_denied", "tenant access denied")
		return false
	}
	return true
}

func sessionIDFromRequest(r *http.Request) string {
	if token := bearerToken(r.Header.Get("Authorization")); token != "" {
		return token
	}
	return strings.TrimSpace(r.Header.Get("X-Session-ID"))
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
	case "/api/devices/config", "/api/devices/status":
		return true
	default:
		return r.Method == http.MethodOptions
	}
}

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"qms/auth-service/internal/models"
	"qms/auth-service/internal/store"
)

type fakeStore struct {
	loginFn   func(ctx context.Context, input store.LoginInput) (store.LoginResult, error)
	sessionFn func(ctx context.Context, sessionID string) (models.Session, models.User, error)
	accessFn  func(ctx context.Context, userID string) ([]string, []string, error)
	ssoFn     func(ctx context.Context, tenantID, provider, subject, email string) (store.LoginResult, error)
}

func (f fakeStore) Login(ctx context.Context, input store.LoginInput) (store.LoginResult, error) {
	if f.loginFn == nil {
		return store.LoginResult{}, nil
	}
	return f.loginFn(ctx, input)
}

func (f fakeStore) GetSession(ctx context.Context, sessionID string) (models.Session, models.User, error) {
	if f.sessionFn == nil {
		return models.Session{}, models.User{}, store.ErrSessionNotFound
	}
	return f.sessionFn(ctx, sessionID)
}

func (f fakeStore) GetAccess(ctx context.Context, userID string) ([]string, []string, error) {
	if f.accessFn == nil {
		return nil, nil, nil
	}
	return f.accessFn(ctx, userID)
}

func (f fakeStore) CreateSession(ctx context.Context, userID string, expiresAt time.Time) (models.Session, error) {
	return models.Session{}, nil
}

func (f fakeStore) SSOLogin(ctx context.Context, tenantID, provider, subject, email string) (store.LoginResult, error) {
	if f.ssoFn == nil {
		return store.LoginResult{}, nil
	}
	return f.ssoFn(ctx, tenantID, provider, subject, email)
}

func TestLoginSuccess(t *testing.T) {
	st := fakeStore{
		loginFn: func(ctx context.Context, input store.LoginInput) (store.LoginResult, error) {
			return store.LoginResult{
				User: models.User{UserID: "user-1", TenantID: input.TenantID, RoleName: "agent", Email: input.Email},
				Session: models.Session{SessionID: "sess-1", ExpiresAt: time.Now().UTC().Add(time.Hour)},
				Branches: []string{"branch-1"},
				Services: []string{"service-1"},
			}, nil
		},
	}
	payload := map[string]string{
		"tenant_id": "11111111-1111-1111-1111-111111111111",
		"email":     "agent@example.com",
		"password":  "secret",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	NewHandler(st).Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	st := fakeStore{
		loginFn: func(ctx context.Context, input store.LoginInput) (store.LoginResult, error) {
			return store.LoginResult{}, store.ErrInvalidCredentials
		},
	}
	payload := map[string]string{
		"tenant_id": "11111111-1111-1111-1111-111111111111",
		"email":     "agent@example.com",
		"password":  "wrong",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	NewHandler(st).Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.Code)
	}
}

func TestMeUnauthorized(t *testing.T) {
	st := fakeStore{}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	resp := httptest.NewRecorder()

	NewHandler(st).Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.Code)
	}
}

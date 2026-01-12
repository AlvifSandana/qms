package store

import (
	"context"
	"time"

	"qms/auth-service/internal/models"
)

type LoginInput struct {
	TenantID string
	Email    string
	Password string
	BranchID string
}

type LoginResult struct {
	User     models.User
	Session  models.Session
	Branches []string
	Services []string
}

type Store interface {
	Login(ctx context.Context, input LoginInput) (LoginResult, error)
	GetSession(ctx context.Context, sessionID string) (models.Session, models.User, error)
	GetAccess(ctx context.Context, userID string) ([]string, []string, error)
	CreateSession(ctx context.Context, userID string, expiresAt time.Time) (models.Session, error)
	SSOLogin(ctx context.Context, tenantID, provider, subject, email string) (LoginResult, error)
}

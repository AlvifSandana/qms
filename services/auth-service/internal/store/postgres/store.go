package postgres

import (
	"context"
	"errors"
	"time"

	"qms/auth-service/internal/models"
	"qms/auth-service/internal/store"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Login(ctx context.Context, input store.LoginInput) (store.LoginResult, error) {
	var result store.LoginResult

	var user models.User
	var passwordHash string
	row := s.pool.QueryRow(ctx, `
		SELECT u.user_id, u.tenant_id, r.name, u.email, u.password_hash, u.created_at
		FROM users u
		JOIN roles r ON r.role_id = u.role_id
		WHERE u.tenant_id = $1 AND lower(u.email) = lower($2) AND u.active = TRUE
	`, input.TenantID, input.Email)
	if err := row.Scan(&user.UserID, &user.TenantID, &user.RoleName, &user.Email, &passwordHash, &user.Created); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.LoginResult{}, store.ErrInvalidCredentials
		}
		return store.LoginResult{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)); err != nil {
		return store.LoginResult{}, store.ErrInvalidCredentials
	}

	branches, services, err := s.GetAccess(ctx, user.UserID)
	if err != nil {
		return store.LoginResult{}, err
	}

	if input.BranchID != "" {
		allowed := false
		for _, branchID := range branches {
			if branchID == input.BranchID {
				allowed = true
				break
			}
		}
		if !allowed {
			return store.LoginResult{}, store.ErrAccessDenied
		}
	}

	session, err := s.CreateSession(ctx, user.UserID, time.Now().UTC().Add(8*time.Hour))
	if err != nil {
		return store.LoginResult{}, err
	}

	result.User = user
	result.Session = session
	result.Branches = branches
	result.Services = services
	return result, nil
}

func (s *Store) GetSession(ctx context.Context, sessionID string) (models.Session, models.User, error) {
	var session models.Session
	var user models.User
	row := s.pool.QueryRow(ctx, `
		SELECT s.session_id, s.user_id, s.expires_at,
		       u.user_id, u.tenant_id, r.name, u.email, u.created_at
		FROM sessions s
		JOIN users u ON u.user_id = s.user_id
		JOIN roles r ON r.role_id = u.role_id
		WHERE s.session_id = $1 AND s.expires_at > NOW()
	`, sessionID)
	if err := row.Scan(&session.SessionID, &session.UserID, &session.ExpiresAt, &user.UserID, &user.TenantID, &user.RoleName, &user.Email, &user.Created); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Session{}, models.User{}, store.ErrSessionNotFound
		}
		return models.Session{}, models.User{}, err
	}
	return session, user, nil
}

func (s *Store) GetAccess(ctx context.Context, userID string) ([]string, []string, error) {
	branchRows, err := s.pool.Query(ctx, `
		SELECT branch_id
		FROM user_branch_access
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer branchRows.Close()

	var branches []string
	for branchRows.Next() {
		var branchID string
		if err := branchRows.Scan(&branchID); err != nil {
			return nil, nil, err
		}
		branches = append(branches, branchID)
	}
	if err := branchRows.Err(); err != nil {
		return nil, nil, err
	}

	serviceRows, err := s.pool.Query(ctx, `
		SELECT service_id
		FROM user_service_access
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer serviceRows.Close()

	var services []string
	for serviceRows.Next() {
		var serviceID string
		if err := serviceRows.Scan(&serviceID); err != nil {
			return nil, nil, err
		}
		services = append(services, serviceID)
	}
	if err := serviceRows.Err(); err != nil {
		return nil, nil, err
	}

	return branches, services, nil
}

func (s *Store) CreateSession(ctx context.Context, userID string, expiresAt time.Time) (models.Session, error) {
	sessionID := uuid.NewString()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (session_id, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, sessionID, userID, expiresAt)
	if err != nil {
		return models.Session{}, err
	}
	return models.Session{SessionID: sessionID, UserID: userID, ExpiresAt: expiresAt}, nil
}

func (s *Store) SSOLogin(ctx context.Context, tenantID, provider, subject, email string) (store.LoginResult, error) {
	var user models.User
	row := s.pool.QueryRow(ctx, `
		SELECT u.user_id, u.tenant_id, r.name, u.email, u.created_at
		FROM user_idp_mappings m
		JOIN users u ON u.user_id = m.user_id
		JOIN roles r ON r.role_id = u.role_id
		WHERE m.tenant_id = $1 AND m.provider = $2 AND m.subject = $3 AND u.active = TRUE
	`, tenantID, provider, subject)
	if err := row.Scan(&user.UserID, &user.TenantID, &user.RoleName, &user.Email, &user.Created); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.createMappedUser(ctx, tenantID, provider, subject, email)
		}
		return store.LoginResult{}, err
	}

	branches, services, err := s.GetAccess(ctx, user.UserID)
	if err != nil {
		return store.LoginResult{}, err
	}
	session, err := s.CreateSession(ctx, user.UserID, time.Now().UTC().Add(8*time.Hour))
	if err != nil {
		return store.LoginResult{}, err
	}
	return store.LoginResult{User: user, Session: session, Branches: branches, Services: services}, nil
}

func (s *Store) createMappedUser(ctx context.Context, tenantID, provider, subject, email string) (store.LoginResult, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return store.LoginResult{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	var roleID string
	row := tx.QueryRow(ctx, `
		SELECT role_id
		FROM roles
		WHERE tenant_id = $1 AND name = 'agent'
		LIMIT 1
	`, tenantID)
	if err = row.Scan(&roleID); err != nil {
		return store.LoginResult{}, err
	}

	userID := uuid.NewString()
	_, err = tx.Exec(ctx, `
		INSERT INTO users (user_id, tenant_id, role_id, email, password_hash, active)
		VALUES ($1, $2, $3, $4, $5, TRUE)
	`, userID, tenantID, roleID, email, "SSO")
	if err != nil {
		return store.LoginResult{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO user_idp_mappings (mapping_id, tenant_id, provider, subject, user_id)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.NewString(), tenantID, provider, subject, userID)
	if err != nil {
		return store.LoginResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return store.LoginResult{}, err
	}

	user := models.User{UserID: userID, TenantID: tenantID, RoleName: "agent", Email: email}
	branches, services, err := s.GetAccess(ctx, userID)
	if err != nil {
		return store.LoginResult{}, err
	}
	session, err := s.CreateSession(ctx, userID, time.Now().UTC().Add(8*time.Hour))
	if err != nil {
		return store.LoginResult{}, err
	}
	return store.LoginResult{User: user, Session: session, Branches: branches, Services: services}, nil
}

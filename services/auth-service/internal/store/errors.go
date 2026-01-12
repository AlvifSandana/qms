package store

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccessDenied       = errors.New("access denied")
	ErrSessionNotFound    = errors.New("session not found")
)

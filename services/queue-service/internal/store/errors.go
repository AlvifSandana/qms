package store

import "errors"

var (
	ErrServiceNotFound    = errors.New("service not found")
	ErrBranchNotFound     = errors.New("branch not found")
	ErrNoTicket           = errors.New("no ticket available")
	ErrTicketNotFound     = errors.New("ticket not found")
	ErrInvalidState       = errors.New("invalid ticket state")
	ErrCounterMismatch    = errors.New("counter mismatch")
	ErrCounterNotFound    = errors.New("counter not found")
	ErrCounterUnavailable = errors.New("counter unavailable")
	ErrAccessDenied       = errors.New("access denied")
	ErrHolidayClosed      = errors.New("holiday closed")
)

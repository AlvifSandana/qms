package store

import "errors"

var (
	ErrBranchHasServices = errors.New("branch has active services")
	ErrApprovalNotFound  = errors.New("approval request not found")
	ErrApprovalNotPending = errors.New("approval request not pending")
	ErrAccessDenied      = errors.New("access denied")
	ErrSessionNotFound   = errors.New("session not found")
)

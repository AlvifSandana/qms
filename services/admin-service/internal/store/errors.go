package store

import "errors"

var (
	ErrBranchHasServices = errors.New("branch has active services")
)

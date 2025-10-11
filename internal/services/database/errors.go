package database

import "errors"

var (
	// ErrOperationNotSupported indicates the requested operation is not allowed for the instance type.
	ErrOperationNotSupported = errors.New("operation not supported for this instance type")
)

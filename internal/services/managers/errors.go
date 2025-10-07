package managers

import "errors"

var (
	// ErrInstanceNotInitialized is returned when the instance is not initialized
	ErrInstanceNotInitialized = errors.New("instance not initialized")

	// ErrInvalidConnection is returned when the connection configuration is invalid
	ErrInvalidConnection = errors.New("invalid connection configuration")

	// ErrConnectionFailed is returned when connection to service fails
	ErrConnectionFailed = errors.New("connection failed")

	// ErrNotConnected is returned when operation requires active connection
	ErrNotConnected = errors.New("not connected to service")

	// ErrInvalidConfig is returned when configuration validation fails
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrHealthCheckFailed is returned when health check fails
	ErrHealthCheckFailed = errors.New("health check failed")

	// ErrMetricsCollectionFailed is returned when metrics collection fails
	ErrMetricsCollectionFailed = errors.New("metrics collection failed")

	// ErrUnsupportedOperation is returned when operation is not supported
	ErrUnsupportedOperation = errors.New("operation not supported")
)

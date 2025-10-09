package webssh

import "errors"

// Common errors
var (
	ErrInvalidMessageType = errors.New("invalid message type")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
	ErrConnectionClosed   = errors.New("connection closed")
	ErrInvalidInput       = errors.New("invalid input")
	ErrMaxSessionsReached = errors.New("maximum sessions reached")
	ErrAuthFailed         = errors.New("authentication failed")
	ErrHostNotFound       = errors.New("host not found")
	ErrAgentOffline       = errors.New("agent offline")
)

// Error codes for client
const (
	ErrCodeSessionNotFound    = "SESSION_NOT_FOUND"
	ErrCodeSessionExpired     = "SESSION_EXPIRED"
	ErrCodeConnectionClosed   = "CONNECTION_CLOSED"
	ErrCodeInvalidInput       = "INVALID_INPUT"
	ErrCodeMaxSessionsReached = "MAX_SESSIONS_REACHED"
	ErrCodeAuthFailed         = "AUTH_FAILED"
	ErrCodeHostNotFound       = "HOST_NOT_FOUND"
	ErrCodeAgentOffline       = "AGENT_OFFLINE"
	ErrCodeInternalError      = "INTERNAL_ERROR"
)

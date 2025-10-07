package performance

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutConfig holds timeout configuration
type TimeoutConfig struct {
	DefaultTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
}

// DefaultTimeoutConfig returns default timeout configuration
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		DefaultTimeout: 30 * time.Second,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
	}
}

// TimeoutMiddleware creates a middleware with request timeout
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Replace request context
		c.Request = c.Request.WithContext(ctx)

		// Channel to track completion
		finished := make(chan struct{})

		go func() {
			c.Next()
			finished <- struct{}{}
		}()

		// Wait for completion or timeout
		select {
		case <-finished:
			// Request completed successfully
			return
		case <-ctx.Done():
			// Request timed out
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"error":   "Request timeout",
				"timeout": timeout.String(),
			})
			return
		}
	}
}

// AdaptiveTimeoutMiddleware creates a middleware with adaptive timeout based on endpoint
func AdaptiveTimeoutMiddleware(timeouts map[string]time.Duration, defaultTimeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		endpoint := c.FullPath()
		timeout := defaultTimeout

		// Check if endpoint has custom timeout
		if customTimeout, exists := timeouts[endpoint]; exists {
			timeout = customTimeout
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Replace request context
		c.Request = c.Request.WithContext(ctx)

		// Channel to track completion
		finished := make(chan struct{})
		timedOut := false

		go func() {
			c.Next()
			if !timedOut {
				finished <- struct{}{}
			}
		}()

		// Wait for completion or timeout
		select {
		case <-finished:
			// Request completed successfully
			return
		case <-ctx.Done():
			// Request timed out
			timedOut = true
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"error":    "Request timeout",
				"endpoint": endpoint,
				"timeout":  timeout.String(),
			})
			return
		}
	}
}

// TimeoutManager manages request timeouts
type TimeoutManager struct {
	endpointTimeouts map[string]time.Duration
	defaultTimeout   time.Duration
	stats            *TimeoutStats
}

// TimeoutStats tracks timeout statistics
type TimeoutStats struct {
	TotalRequests   int64
	TimeoutCount    int64
	TimeoutRate     float64
	AverageWaitTime time.Duration
}

// NewTimeoutManager creates a new timeout manager
func NewTimeoutManager(defaultTimeout time.Duration) *TimeoutManager {
	return &TimeoutManager{
		endpointTimeouts: make(map[string]time.Duration),
		defaultTimeout:   defaultTimeout,
		stats:            &TimeoutStats{},
	}
}

// SetEndpointTimeout sets timeout for a specific endpoint
func (tm *TimeoutManager) SetEndpointTimeout(endpoint string, timeout time.Duration) {
	tm.endpointTimeouts[endpoint] = timeout
}

// GetTimeout retrieves timeout for an endpoint
func (tm *TimeoutManager) GetTimeout(endpoint string) time.Duration {
	if timeout, exists := tm.endpointTimeouts[endpoint]; exists {
		return timeout
	}
	return tm.defaultTimeout
}

// RecordRequest records a request
func (tm *TimeoutManager) RecordRequest(timedOut bool, waitTime time.Duration) {
	tm.stats.TotalRequests++
	if timedOut {
		tm.stats.TimeoutCount++
	}

	// Update timeout rate
	if tm.stats.TotalRequests > 0 {
		tm.stats.TimeoutRate = float64(tm.stats.TimeoutCount) / float64(tm.stats.TotalRequests)
	}

	// Update average wait time (simple moving average)
	totalWaitTime := tm.stats.AverageWaitTime * time.Duration(tm.stats.TotalRequests-1)
	tm.stats.AverageWaitTime = (totalWaitTime + waitTime) / time.Duration(tm.stats.TotalRequests)
}

// GetStats retrieves timeout statistics
func (tm *TimeoutManager) GetStats() TimeoutStats {
	return *tm.stats
}

// ResetStats resets timeout statistics
func (tm *TimeoutManager) ResetStats() {
	tm.stats = &TimeoutStats{}
}

// ContextWithTimeout creates a context with timeout
func ContextWithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// WithTimeout wraps a function with timeout
func WithTimeout(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	errChan := make(chan error, 1)

	go func() {
		errChan <- fn(timeoutCtx)
	}()

	select {
	case err := <-errChan:
		return err
	case <-timeoutCtx.Done():
		return fmt.Errorf("operation timeout after %s", timeout)
	}
}

// RetryWithTimeout retries a function with timeout
func RetryWithTimeout(ctx context.Context, timeout time.Duration, maxRetries int, fn func(context.Context) error) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		err := WithTimeout(ctx, timeout, fn)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if context is cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Exponential backoff
		if i < maxRetries-1 {
			backoff := time.Duration(1<<uint(i)) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// Deadline represents a deadline for a task
type Deadline struct {
	Time      time.Time
	Remaining time.Duration
	Exceeded  bool
}

// GetDeadline retrieves deadline information from context
func GetDeadline(ctx context.Context) *Deadline {
	deadline, ok := ctx.Deadline()
	if !ok {
		return &Deadline{
			Remaining: time.Duration(0),
			Exceeded:  false,
		}
	}

	remaining := time.Until(deadline)
	return &Deadline{
		Time:      deadline,
		Remaining: remaining,
		Exceeded:  remaining <= 0,
	}
}

// CheckTimeout checks if a timeout has occurred
func CheckTimeout(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// TimeoutHandler wraps a handler with timeout protection
type TimeoutHandler struct {
	handler gin.HandlerFunc
	timeout time.Duration
}

// NewTimeoutHandler creates a new timeout handler
func NewTimeoutHandler(handler gin.HandlerFunc, timeout time.Duration) *TimeoutHandler {
	return &TimeoutHandler{
		handler: handler,
		timeout: timeout,
	}
}

// Handle executes the handler with timeout
func (th *TimeoutHandler) Handle(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), th.timeout)
	defer cancel()

	c.Request = c.Request.WithContext(ctx)

	done := make(chan struct{})

	go func() {
		th.handler(c)
		done <- struct{}{}
	}()

	select {
	case <-done:
		return
	case <-ctx.Done():
		c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
			"error":   "Request timeout",
			"timeout": th.timeout.String(),
		})
	}
}

// TimeoutPool manages a pool of timeout handlers
type TimeoutPool struct {
	handlers map[string]*TimeoutHandler
}

// NewTimeoutPool creates a new timeout pool
func NewTimeoutPool() *TimeoutPool {
	return &TimeoutPool{
		handlers: make(map[string]*TimeoutHandler),
	}
}

// Register registers a handler with timeout
func (tp *TimeoutPool) Register(endpoint string, handler gin.HandlerFunc, timeout time.Duration) {
	tp.handlers[endpoint] = NewTimeoutHandler(handler, timeout)
}

// Get retrieves a timeout handler
func (tp *TimeoutPool) Get(endpoint string) (*TimeoutHandler, bool) {
	handler, exists := tp.handlers[endpoint]
	return handler, exists
}

// GracefulTimeout implements graceful timeout with cleanup
func GracefulTimeout(ctx context.Context, timeout time.Duration, workFn func() error, cleanupFn func()) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	errChan := make(chan error, 1)

	go func() {
		errChan <- workFn()
	}()

	select {
	case err := <-errChan:
		return err
	case <-timeoutCtx.Done():
		if cleanupFn != nil {
			cleanupFn()
		}
		return fmt.Errorf("graceful timeout after %s", timeout)
	}
}

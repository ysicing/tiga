package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/golang-lru/v2/expirable"
)

// RateLimitConfig represents rate limit configuration
type RateLimitConfig struct {
	RequestsPerSecond int           // Max requests per second
	Burst             int           // Max burst size
	Duration          time.Duration // Time window
	KeyFunc           func(*gin.Context) string
}

// RateLimiter implements token bucket algorithm
type RateLimiter struct {
	config *RateLimitConfig
	cache  *expirable.LRU[string, *bucket]
	mu     sync.RWMutex
}

// bucket represents a token bucket for rate limiting
type bucket struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	if config.Duration == 0 {
		config.Duration = time.Second
	}

	if config.KeyFunc == nil {
		config.KeyFunc = func(c *gin.Context) string {
			return c.ClientIP()
		}
	}

	// Create LRU cache with 1 hour TTL for buckets
	cache := expirable.NewLRU[string, *bucket](10000, nil, time.Hour)

	return &RateLimiter{
		config: config,
		cache:  cache,
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Get or create bucket
	b, exists := rl.cache.Get(key)
	if !exists {
		b = &bucket{
			tokens:     rl.config.Burst,
			lastRefill: time.Now(),
		}
		rl.cache.Add(key, b)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill)

	// Refill tokens based on elapsed time
	tokensToAdd := int(elapsed.Seconds() * float64(rl.config.RequestsPerSecond))
	if tokensToAdd > 0 {
		b.tokens = min(b.tokens+tokensToAdd, rl.config.Burst)
		b.lastRefill = now
	}

	// Check if we have tokens
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// RateLimit returns a rate limiting middleware
func RateLimit(requestsPerSecond, burst int) gin.HandlerFunc {
	config := &RateLimitConfig{
		RequestsPerSecond: requestsPerSecond,
		Burst:             burst,
		Duration:          time.Second,
	}

	limiter := NewRateLimiter(config)

	return func(c *gin.Context) {
		key := config.KeyFunc(c)

		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": "too many requests",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByUser returns a rate limiting middleware by user
func RateLimitByUser(requestsPerSecond, burst int) gin.HandlerFunc {
	config := &RateLimitConfig{
		RequestsPerSecond: requestsPerSecond,
		Burst:             burst,
		Duration:          time.Second,
		KeyFunc: func(c *gin.Context) string {
			// Use user ID if authenticated, otherwise IP
			if userID, exists := c.Get(string(UserIDKey)); exists {
				return userID.(string)
			}
			return c.ClientIP()
		},
	}

	limiter := NewRateLimiter(config)

	return func(c *gin.Context) {
		key := config.KeyFunc(c)

		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": "too many requests",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByIP returns a rate limiting middleware by IP address
func RateLimitByIP(requestsPerSecond, burst int) gin.HandlerFunc {
	return RateLimit(requestsPerSecond, burst)
}

// RateLimitByEndpoint returns a rate limiting middleware by endpoint
func RateLimitByEndpoint(requestsPerSecond, burst int) gin.HandlerFunc {
	config := &RateLimitConfig{
		RequestsPerSecond: requestsPerSecond,
		Burst:             burst,
		Duration:          time.Second,
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP() + ":" + c.Request.URL.Path
		},
	}

	limiter := NewRateLimiter(config)

	return func(c *gin.Context) {
		key := config.KeyFunc(c)

		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": "too many requests for this endpoint",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Global rate limiter
var globalRateLimiter *RateLimiter

// InitRateLimiter initializes the global rate limiter
func InitRateLimiter(config *RateLimitConfig) {
	globalRateLimiter = NewRateLimiter(config)
}

// GlobalRateLimit returns the global rate limiting middleware
func GlobalRateLimit() gin.HandlerFunc {
	if globalRateLimiter == nil {
		panic("rate limiter not initialized")
	}

	return func(c *gin.Context) {
		key := c.ClientIP()

		if !globalRateLimiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

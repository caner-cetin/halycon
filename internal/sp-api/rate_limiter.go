package sp_api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

// RateLimiterManager manages multiple rate limiters by key
type RateLimiterManager struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

// NewRateLimiterManager creates a new rate limiter manager
func NewRateLimiterManager(limiters map[string]*rate.Limiter) *RateLimiterManager {
	return &RateLimiterManager{
		limiters: limiters,
	}
}

// GetLimiter returns a rate limiter for the given key, creating one if it doesn't exist
func (m *RateLimiterManager) GetLimiter(key string) *rate.Limiter {
	m.mu.RLock()
	limiter, _ := m.limiters[key]
	m.mu.RUnlock()
	return limiter
}

// RateLimiterInterceptor creates a request interceptor function for oapi-codegen
// that applies rate limiting based on a key.
func (m *RateLimiterManager) RateLimiterInterceptor(key string) func(context.Context, *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		limiter := m.GetLimiter(key)

		// Try to reserve a token
		reservation := limiter.Reserve()
		if !reservation.OK() {
			return fmt.Errorf("rate limit exceeded for request to %s (exceeds maximum wait time)",
				req.URL.String())
		}

		delay := reservation.Delay()
		if delay > 0 {
			log.Trace().
				Float64("s", delay.Seconds()).
				Int64("ms", delay.Milliseconds()).
				Str("url", req.URL.String()).
				Msg("rate limit reached")

			select {
			case <-time.After(delay):
				// Waited successfully
			case <-ctx.Done():
				reservation.Cancel()
				return fmt.Errorf("request context cancelled: %w", ctx.Err())
			}
		}

		return nil
	}
}

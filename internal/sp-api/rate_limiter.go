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

type RateLimiterManager struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

func NewRateLimiterManager(limiters map[string]*rate.Limiter) *RateLimiterManager {
	return &RateLimiterManager{
		limiters: limiters,
	}
}

func (m *RateLimiterManager) GetLimiter(key string) *rate.Limiter {
	m.mu.RLock()
	limiter := m.limiters[key]
	m.mu.RUnlock()
	return limiter
}

func (m *RateLimiterManager) RateLimiterInterceptor(key string) func(context.Context, *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		limiter := m.GetLimiter(key)

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
				// I'll just keep waiting
				// You'll just keep waiting
				// In the cold
				// The supplement
				// We lost some friends
				// We drove the bends
				// So small
			case <-ctx.Done():
				reservation.Cancel()
				return fmt.Errorf("request context cancelled: %w", ctx.Err())
			}
		}

		return nil
	}
}

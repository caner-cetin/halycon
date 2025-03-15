package sp_api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

// RateLimitedTransport implements rate limiting for HTTP requests by wrapping
// an existing http.RoundTripper with a token bucket rate limiter.
// It ensures that requests are throttled according to specified rate limits.
// Base is the underlying transport to be rate limited.
// RateLimiter controls the rate at which requests are allowed to proceed.
type RateLimitedTransport struct {
	Base        http.RoundTripper
	RateLimiter *rate.Limiter
}

// RoundTrip implements http.RoundTripper interface and handles rate limiting for HTTP requests.
// It uses a token bucket rate limiter to control request rates.
//
// The function:
//
// 1. Attempts to reserve a token from the rate limiter
//
// 2. If reservation fails (exceeds maximum wait time), returns an error
//
// 3. If there's a delay needed, waits for the required time
//
// 4. Cancels the reservation and returns error if request context is cancelled during wait
//
// 5. Forwards the request to the base transport if rate limiting conditions are met
func (t *RateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reservation := t.RateLimiter.Reserve()
	if !reservation.OK() {
		// %99.99 of time, this block wont run
		// unless you created the RateLimiter with NewLimiter().WithLimit(maxWait), waiting time is infinite
		// and [rate.Reservation.OK] returns whether the limiter can provide the requested number of tokens within the maximum wait time.
		// so unless a maximum wait time is given, this is actually useless.
		return nil, fmt.Errorf("rate limit exceeded for request to %s (exceeds maximum wait time)", req.URL.String())
	}
	delay := reservation.Delay()
	if delay > 0 {
		log.Trace().
			Float64("s", delay.Seconds()).
			Int64("ms", delay.Milliseconds()).
			Str("url", req.URL.String()).
			Msg("rate limit reached")
		timer := time.NewTimer(delay)
		select { // waiting...
		case <-timer.C:
			// done!
		case <-req.Context().Done():
			timer.Stop()
			reservation.Cancel()
			return nil, fmt.Errorf("request context cancelled: %w", req.Context().Err())
		}
	}
	resp, err := t.Base.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("base roundtrip failed: %w", err)
	}
	return resp, nil
}

// NewRateLimitedClient creates and returns a new HTTP client with rate limiting capabilities.
// It wraps the default HTTP transport with a rate limiter to control request frequency.
// The rate limiting is applied at the transport level, affecting all requests made through this client.
func NewRateLimitedClient(rl *rate.Limiter) *http.Client {
	transport := &RateLimitedTransport{
		Base:        http.DefaultTransport,
		RateLimiter: rl,
	}
	return &http.Client{
		Transport: transport,
		// Timeout:   30 * time.Second
	}
}

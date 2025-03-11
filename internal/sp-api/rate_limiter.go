package sp_api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

type RateLimitedTransport struct {
	Base        http.RoundTripper
	RateLimiter *rate.Limiter
}

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
			return nil, req.Context().Err()
		}
	}
	return t.Base.RoundTrip(req)
}

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

package internal

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/client"
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
		log.Trace().Dur("delay", delay).
			Float64("available_tokens", t.RateLimiter.Tokens()).
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

func NewRateLimitedClient(requestsPerSecond float64, burst int) *http.Client {
	limiter := rate.NewLimiter(rate.Every(time.Second*time.Duration(requestsPerSecond)), burst)
	transport := &RateLimitedTransport{
		Base:        http.DefaultTransport,
		RateLimiter: limiter,
	}
	return &http.Client{
		Transport: transport,
		// Timeout:   30 * time.Second
	}
}

func NewRateLimitedRuntime(requestsPerSecond float64, burst int, schemes []string, host string, basePath string) runtime.ClientTransport {
	httpClient := NewRateLimitedClient(requestsPerSecond, burst)
	return client.NewWithClient(host, basePath, schemes, httpClient)
}

func ConfigureRateLimiting(requestsPerSecond float64, burst int) func(op *runtime.ClientOperation) {
	return func(op *runtime.ClientOperation) {
		httpClient := NewRateLimitedClient(requestsPerSecond, burst)
		op.Client = httpClient
	}
}

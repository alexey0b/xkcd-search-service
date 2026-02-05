package middleware

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"
)

// Limit represents the maximum rate of events (events/sec).
type Limit float64

// Inf represents an infinite rate limit (allows all events).
const Inf = Limit(math.MaxFloat64)

// defaultBurst is the default burst size for strict RPS compliance.
const defaultBurst = 1

// RateLimiter implements the Token Bucket algorithm for rate limiting.
// Based on golang.org/x/time/rate implementation.
type RateLimiter struct {
	mu     sync.Mutex
	limit  Limit     // maximum rate (tokens per second)
	burst  int       // maximum burst size (token bucket capacity)
	tokens float64   // current number of available tokens
	last   time.Time // last time tokens were updated
}

// NewRateLimiter creates a rate limiter with specified RPS (requests per second).
// If rate <= 0, all events will wait indefinitely until context cancellation.
func NewRateLimiter(rate int) *RateLimiter {
	return &RateLimiter{
		limit: Limit(rate),
		burst: defaultBurst,
	}
}

// Limit is a middleware that enforces rate limiting using rate limiter.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := rl.wait(r.Context()); err != nil {
			http.Error(w, http.StatusText(http.StatusRequestTimeout), http.StatusRequestTimeout)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// wait blocks until a token is available or context is cancelled.
func (rl *RateLimiter) wait(ctx context.Context) error {
	rl.mu.Lock()

	// refill tokens based on elapsed time
	now := time.Now()
	rl.tokens = rl.tokensAt(now)
	rl.last = now

	// try to consume one
	limit := rl.limit
	tokens := rl.tokens - 1
	var delay time.Duration
	if tokens < 0 {
		// not enough tokens, calculate wait time
		delay = limit.durationFromTokens(-tokens)
	}

	// check if delay would exceed context deadline
	if deadline, ok := ctx.Deadline(); ok {
		if now.Add(delay).After(deadline) {
			rl.mu.Unlock()
			return fmt.Errorf("rate: wait would exceed context deadline")
		}
	}

	rl.tokens = tokens

	rl.mu.Unlock()

	// no delay needed, token available immediately
	if delay <= 0 {
		return nil
	}

	// wait for token to become available
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// tokensAt calculates the number of available tokens at time t.
func (rl *RateLimiter) tokensAt(t time.Time) float64 {
	if rl.limit == Inf {
		return float64(rl.burst)
	}

	// calculate elapsed time since last update
	elapsed := t.Sub(rl.last)
	elapsed = max(elapsed, 0)

	// calculate tokens to add based on elapsed time
	delta := rl.limit.tokensFromDuration(elapsed)
	tokens := rl.tokens + delta

	// number of available tokens must be less or equal burst size
	if burst := float64(rl.burst); tokens > burst {
		tokens = burst
	}
	return tokens
}

// durationFromTokens calculates the time needed to accumulate the given number of tokens.
func (limit Limit) durationFromTokens(tokens float64) time.Duration {
	if limit <= 0 {
		return time.Duration(math.MaxInt64)
	}
	seconds := tokens / float64(limit)
	return time.Duration(float64(time.Second) * seconds)
}

// tokensFromDuration calculates the number of tokens accumulated over duration d.
func (limit Limit) tokensFromDuration(d time.Duration) float64 {
	if limit <= 0 {
		return 0
	}
	return d.Seconds() * float64(limit)
}

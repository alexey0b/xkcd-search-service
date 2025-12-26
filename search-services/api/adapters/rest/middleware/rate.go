package middleware

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"
)

// Limit - максимальная частота событий (event/sec).
type Limit float64

// Inf - бесконечный лимит (allows all events).
const Inf = Limit(math.MaxFloat64)

// defaultBurst - размер defaultBurst по умолчанию для строгого соблюдения RPS.
const defaultBurst = 1

// RateLimiter реализует алгоритм Token Bucket для ограничения скорости запросов.
// Реализация основана на golang.org/x/time/rate.
type RateLimiter struct {
	mu     sync.Mutex
	limit  Limit
	burst  int
	tokens float64
	// last время последнего обновления токенов
	last time.Time
}

// NewRateLimiter создает rate limiter с заданным RPS.
// При rate <= 0 все события бесконечно ожидают, пока не будет отмены внешнего контекста.
func NewRateLimiter(rate int) *RateLimiter {
	return &RateLimiter{
		limit: Limit(rate),
		burst: defaultBurst,
	}
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := rl.wait(r.Context()); err != nil {
			http.Error(w, http.StatusText(http.StatusRequestTimeout), http.StatusRequestTimeout)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) wait(ctx context.Context) error {
	rl.mu.Lock()
	limit := rl.limit
	rl.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	rl.mu.Lock()
	now := time.Now()
	rl.tokens = rl.tokensAt(now)
	rl.last = now

	tokens := rl.tokens - 1
	var delay time.Duration
	if tokens < 0 {
		delay = limit.durationFromTokens(-tokens)
	}

	if deadline, ok := ctx.Deadline(); ok {
		if now.Add(delay).After(deadline) {
			rl.mu.Unlock()
			return fmt.Errorf("rate: Wait would exceed context deadline")
		}
	}

	rl.tokens = tokens
	rl.mu.Unlock()

	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (rl *RateLimiter) tokensAt(t time.Time) float64 {
	if rl.limit == Inf {
		return float64(rl.burst)
	}

	elapsed := t.Sub(rl.last)
	elapsed = max(elapsed, 0)

	delta := rl.limit.tokensFromDuration(elapsed)
	tokens := rl.tokens + delta

	if burst := float64(rl.burst); tokens > burst {
		tokens = burst
	}

	return tokens
}

func (limit Limit) durationFromTokens(tokens float64) time.Duration {
	if limit <= 0 {
		return time.Duration(math.MaxInt64)
	}
	seconds := tokens / float64(limit)
	return time.Duration(float64(time.Second) * seconds)
}

func (limit Limit) tokensFromDuration(d time.Duration) float64 {
	if limit <= 0 {
		return 0
	}
	return d.Seconds() * float64(limit)
}

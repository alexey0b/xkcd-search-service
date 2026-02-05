package middleware

import "net/http"

// ConcurrencyLimiter limits the number of concurrent HTTP requests using a semaphore pattern.
type ConcurrencyLimiter struct {
	sem chan struct{} // buffered channel acts as a semaphore
}

// NewConcurrencyLimiter creates a new limiter with specified maximum concurrent requests.
func NewConcurrencyLimiter(concurrency int) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		sem: make(chan struct{}, concurrency),
	}
}

// Limit is a middleware that enforces concurrency limit.
func (cl *ConcurrencyLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case cl.sem <- struct{}{}: // try to acquire semaphore and release semaphore when done
			defer func() { <-cl.sem }()
			next.ServeHTTP(w, r)
		default: // semaphore full, reject request
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		}
	})
}

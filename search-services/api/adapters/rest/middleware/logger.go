package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logging is a middleware that logs HTTP requests with method, path, and duration.
// Note: health check requests (/api/helth) are logged at DEBUG level.
func Logging(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		if r.URL.Path == "/api/helth" {
			log.Debug("request", "method", r.Method, "path", r.URL.Path, "duration", duration)
		} else {
			log.Info("request", "method", r.Method, "path", r.URL.Path, "duration", duration)
		}
	})
}

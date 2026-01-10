package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		if r.URL.Path == "/api/ping" {
			log.Debug("request", "method", r.Method, "path", r.URL.Path, "duration", duration)
		} else {
			log.Info("request", "method", r.Method, "path", r.URL.Path, "duration", duration)
		}
	})
}

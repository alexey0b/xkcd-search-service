package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, req)
		log.Info("request", "method", req.Method, "path", req.URL.Path, "duration", time.Since(start))
	})
}

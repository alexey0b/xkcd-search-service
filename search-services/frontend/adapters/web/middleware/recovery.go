package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// PanicRecovery is a middleware that recovers from panics in HTTP handlers.
func PanicRecovery(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// log panic with full context and stack trace
				log.Error("panic recovered",
					"error", err,
					"method", req.Method,
					"path", req.URL.Path,
					"stack", string(debug.Stack()),
				)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, req)
	})
}

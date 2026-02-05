package middleware

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// requestsTotal counts total number of HTTP requests by method and path
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "frontend_processed_ops_total",
		Help: "Total number of requests to the Frontend",
	}, []string{"method", "path"})
	// requestDuration tracks request processing time distribution by method and path
	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "frontend_request_duration_seconds",
		Help:    "Processing time of the request to the Frontend",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})
)

// Metrics is a middleware that collects Prometheus metrics for HTTP requests.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		observeRequest(start, r.Method, r.URL.Path)
		incRequest(r.Method, r.URL.Path)
	})
}

// incRequest increments the request counter for given method and path.
func incRequest(method, path string) {
	requestsTotal.WithLabelValues(method, path).Inc()
}

// observeRequest records the request duration for given method and path.
func observeRequest(start time.Time, method, path string) {
	duration := time.Since(start).Seconds()
	requestDuration.WithLabelValues(method, path).Observe(duration)
}

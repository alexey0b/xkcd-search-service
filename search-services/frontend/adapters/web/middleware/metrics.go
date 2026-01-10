package middleware

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "frontend_processed_ops_total",
		Help: "Total number of requests to the Frontend",
	}, []string{"method", "path"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "frontend_request_duration_seconds",
		Help:    "Processing time of the request to the Frontend",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})
)

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		observeRequest(start, r.Method, r.URL.Path)
		incRequest(r.Method, r.URL.Path)
	})
}

func incRequest(method, path string) {
	requestsTotal.WithLabelValues(method, path).Inc()
}

func observeRequest(start time.Time, method, path string) {
	duration := time.Since(start).Seconds()
	requestDuration.WithLabelValues(method, path).Observe(duration)
}

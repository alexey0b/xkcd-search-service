package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusCollector collects and exposes update service metrics.
type PrometheusCollector struct {
	totalComicsFetched  prometheus.Gauge
	lastUpdateTimestamp prometheus.Gauge
	lastUpdateDuration  prometheus.Gauge
}

// NewPrometheusCollector creates a new Prometheus metrics collector.
func NewPrometheusCollector() *PrometheusCollector {
	return &PrometheusCollector{
		totalComicsFetched: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "update_comics_fetched",
			Help: "Number of comics fetched from XKCD per time",
		}),
		lastUpdateTimestamp: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "update_last_update_timestamp",
			Help: "Unix timestamp of last database update",
		}),
		lastUpdateDuration: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "update_last_update_duration_seconds",
			Help: "Duration of last update in seconds",
		}),
	}
}

// SetComicsFetched sets the number of comics fetched metric.
func (pc *PrometheusCollector) SetComicsFetched(count int64) {
	pc.totalComicsFetched.Set(float64(count))
}

// SetLastUpdateTimestamp sets the last update timestamp to current time.
func (pc *PrometheusCollector) SetLastUpdateTimestamp() {
	pc.lastUpdateTimestamp.SetToCurrentTime()
}

// SetLastUpdateDuration sets the last update duration metric.
func (pc *PrometheusCollector) SetLastUpdateDuration(duration time.Duration) {
	pc.lastUpdateDuration.Set(duration.Seconds())
}

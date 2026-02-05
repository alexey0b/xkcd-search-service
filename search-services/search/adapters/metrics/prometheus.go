package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusCollector collects and exposes search service metrics.
type PrometheusCollector struct {
	indexSize       prometheus.Gauge
	indexLastUpdate prometheus.Gauge
}

// NewPrometheusCollector creates a new Prometheus metrics collector.
func NewPrometheusCollector() *PrometheusCollector {
	return &PrometheusCollector{
		indexSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "search_index_size",
			Help: "Number of keywords in search index",
		}),
		indexLastUpdate: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "search_index_last_update_timestamp",
			Help: "Unix timestamp of last index update",
		}),
	}
}

// SetIndexSize sets the current index size metric.
func (pc *PrometheusCollector) SetIndexSize(size int64) {
	pc.indexSize.Set(float64(size))
}

// SetIndexLastUpdateTimestamp sets the index last update timestamp to current time.
func (pc *PrometheusCollector) SetIndexLastUpdateTimestamp() {
	pc.indexLastUpdate.SetToCurrentTime()
}

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusCollector struct {
	indexSize       prometheus.Gauge
	indexLastUpdate prometheus.Gauge
}

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

func (pc *PrometheusCollector) SetIndexSize(size int64) {
	pc.indexSize.Set(float64(size))
}

func (pc *PrometheusCollector) SetIndexLastUpdateTimestamp() {
	pc.indexLastUpdate.SetToCurrentTime()
}

package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusCollector struct {
	totalComicsFetched  prometheus.Gauge
	lastUpdateTimestamp prometheus.Gauge
	lastUpdateDuration  prometheus.Gauge
}

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

func (pc *PrometheusCollector) SetComicsFetched(count int64) {
	pc.totalComicsFetched.Set(float64(count))
}

func (pc *PrometheusCollector) SetLastUpdateTimestamp() {
	pc.lastUpdateTimestamp.SetToCurrentTime()
}

func (pc *PrometheusCollector) SetLastUpdateDuration(duration time.Duration) {
	pc.lastUpdateDuration.Set(duration.Seconds())
}

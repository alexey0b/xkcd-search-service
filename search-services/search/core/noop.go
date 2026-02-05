package core

// NoopMetricsCollector is a no-op implementation of MetricsCollector.
type NoopMetricsCollector struct{}

func (n *NoopMetricsCollector) SetIndexSize(size int64)      {}
func (n *NoopMetricsCollector) SetIndexLastUpdateTimestamp() {}

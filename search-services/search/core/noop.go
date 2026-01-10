package core

type NoopMetricsCollector struct{}

func (n *NoopMetricsCollector) SetIndexSize(size int64)      {}
func (n *NoopMetricsCollector) SetIndexLastUpdateTimestamp() {}

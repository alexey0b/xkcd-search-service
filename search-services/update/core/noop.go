package core

import time "time"

// NoopMetricsCollector is a no-op implementation of MetricsCollector.
type NoopMetricsCollector struct{}

func (n *NoopMetricsCollector) SetComicsFetched(total int64)                 {}
func (n *NoopMetricsCollector) SetTotalWords(total int64)                    {}
func (n *NoopMetricsCollector) SetTotalWordsUnique(total int64)              {}
func (n *NoopMetricsCollector) SetLastUpdateTimestamp()                      {}
func (n *NoopMetricsCollector) SetLastUpdateDuration(duration time.Duration) {}
func (n *NoopMetricsCollector) SetUpdateStatus(status ServiceStatus)         {}

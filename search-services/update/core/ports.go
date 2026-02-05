package core

import (
	"context"
	"time"
)

//go:generate mockgen -source=ports.go -destination=mocks.go -package=core

// Updater manages database updates and provides statistics.
type Updater interface {
	Update(ctx context.Context) error
	Stats(ctx context.Context) (ServiceStats, error)
	Status(ctx context.Context) ServiceStatus
	Drop(ctx context.Context) error
}

// DB provides database operations for comics storage.
type DB interface {
	Add(ctx context.Context, comic ...Comic) error
	Stats(ctx context.Context) (DBStats, error)
	Drop(ctx context.Context) error
	IDs(ctx context.Context) ([]int64, error)
}

// XKCD provides access to XKCD API.
type XKCD interface {
	Get(ctx context.Context, id int64) (XKCDInfo, error)
	LastID(ctx context.Context) (int64, error)
}

// Words provides text normalization.
type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}

// Publisher publishes events to message broker.
type Publisher interface {
	Publish(event EventType) error
}

// MetricsCollector collects and exposes service metrics.
type MetricsCollector interface {
	SetComicsFetched(count int64)
	SetLastUpdateTimestamp()
	SetLastUpdateDuration(duration time.Duration)
}

package core

import (
	"context"
)

//go:generate mockgen -source=ports.go -destination=mocks.go -package=core

// DB provides database operations for comics storage.
type DB interface {
	GetComicsByIds(ctx context.Context, ids []int64) ([]Comic, error)
	GetAllComicsInfo(ctx context.Context) ([]ComicInfo, error)
}

// Words provides text normalization.
type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}

// Searcher performs search operations on comics.
type Searcher interface {
	Search(ctx context.Context, phrase string, limit int64) ([]Comic, error)
	ISearch(ctx context.Context, phrase string, limit int64) ([]Comic, error)
	UpdateIndex(ctx context.Context) error
	ResetIndex()
}

// EventHandler handles events from message broker.
type EventHandler interface {
	HandleEvent(ctx context.Context, eventType EventType) error
}

// MetricsCollector collects and exposes service metrics.
type MetricsCollector interface {
	SetIndexSize(size int64)
	SetIndexLastUpdateTimestamp()
}

package core

import (
	"context"
)

//go:generate mockgen -source=ports.go -destination=mocks.go -package=core

type DB interface {
	GetComicsByIds(ctx context.Context, ids []int64) ([]Comic, error)
	GetAllComicsInfo(ctx context.Context) ([]ComicInfo, error)
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}

type Searcher interface {
	Search(ctx context.Context, phrase string, limit int64) ([]Comic, error)
	ISearch(ctx context.Context, phrase string, limit int64) ([]Comic, error)
	UpdateIndex(ctx context.Context) error
	ResetIndex()
}

type EventHandler interface {
	HandleEvent(ctx context.Context, eventType EventType) error
}

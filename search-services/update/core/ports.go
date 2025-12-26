package core

import (
	"context"
)

//go:generate mockgen -source=ports.go -destination=mocks.go -package=core

type Updater interface {
	Update(ctx context.Context) error
	Stats(ctx context.Context) (ServiceStats, error)
	Status(ctx context.Context) ServiceStatus
	Drop(ctx context.Context) error
}

type DB interface {
	Add(ctx context.Context, comic ...Comic) error
	Stats(ctx context.Context) (DBStats, error)
	Drop(ctx context.Context) error
	IDs(ctx context.Context) ([]int64, error)
}

type XKCD interface {
	Get(ctx context.Context, id int64) (XKCDInfo, error)
	LastID(ctx context.Context) (int64, error)
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}

type Publisher interface {
	Publish(event EventType) error
}

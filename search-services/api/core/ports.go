package core

import (
	"context"
)

//go:generate mockgen -source=ports.go -destination=mocks.go -package=core

type Normalizer interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type Updater interface {
	Update(ctx context.Context) error
	Stats(ctx context.Context) (UpdateStats, error)
	Status(ctx context.Context) (UpdateStatus, error)
	Drop(ctx context.Context) error
}

type Searcher interface {
	Search(ctx context.Context, phrase string, limit int64) ([]Comic, error)
	ISearch(ctx context.Context, phrase string, limit int64) ([]Comic, error)
}

type Authenticator interface {
	CreateToken(name, password string) (string, error)
	ValidateToken(tokenString string) error
}

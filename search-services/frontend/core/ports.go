package core

import (
	"context"
)

//go:generate mockgen -source=ports.go -destination=mocks.go -package=core

type Pinger interface {
	Ping(ctx context.Context) (PingResponse, error)
}

type UpdateStatsProvider interface {
	GetUpdateStats(ctx context.Context) (UpdateStats, error)
	GetUpdateStatus(ctx context.Context) (UpdateStatus, error)
}

type Updater interface {
	Update(ctx context.Context) error
	Drop(ctx context.Context) error
}

type Searcher interface {
	Search(ctx context.Context, phrase string) (SearchResult, error)
}

type Authenticator interface {
	CreateToken(name, password string) (string, error)
	ValidateToken(tokenString string) error
}

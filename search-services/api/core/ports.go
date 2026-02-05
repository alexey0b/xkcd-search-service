package core

import (
	"context"
)

//go:generate mockgen -source=ports.go -destination=mocks.go -package=core

// Normalizer normalizes and stems words in phrases using Snowball algorithm.
type Normalizer interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}

// HealthChecker checks if a service is healthy and serving requests.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// Updater manages database updates and provides statistics.
type Updater interface {
	Update(ctx context.Context) error
	Stats(ctx context.Context) (UpdateStats, error)
	Status(ctx context.Context) (UpdateStatus, error)
	Drop(ctx context.Context) error
}

// Searcher performs search operations on comics.
type Searcher interface {
	Search(ctx context.Context, phrase string, limit int64) ([]Comic, error)  // non indexed phrase search
	ISearch(ctx context.Context, phrase string, limit int64) ([]Comic, error) // indexed phrase search
}

// Authenticator handles JWT token creation and validation for admin access.
type Authenticator interface {
	CreateToken(name, password string) (string, error)
	ValidateToken(tokenString string) error
}

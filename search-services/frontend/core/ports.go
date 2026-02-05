package core

import (
	"context"
)

//go:generate mockgen -source=ports.go -destination=mocks.go -package=core

// HealthChecker checks if a service is healthy and serving requests.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// UpdateStatsProvider provides database statistics and update process status.
type UpdateStatsProvider interface {
	GetUpdateStats(ctx context.Context) (UpdateStats, error)  // returns database statistics
	GetUpdateStatus(ctx context.Context) (UpdateStatus, error) // returns update process status
}

// Updater manages database updates.
type Updater interface {
	Update(ctx context.Context) error // triggers database update to fetch new comics
	Drop(ctx context.Context) error   // removes all comics and indexed words
}

// Searcher performs search operations on comics.
type Searcher interface {
	Search(ctx context.Context, phrase string) (SearchResult, error)
}

// Authenticator handles JWT token creation and validation for admin access.
type Authenticator interface {
	CreateToken(name, password string) (string, error)
	ValidateToken(tokenString string) error
}

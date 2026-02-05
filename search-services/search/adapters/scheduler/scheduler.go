package scheduler

import (
	"context"
	"log/slog"
	"search-service/search/core"
	"time"
)

// SearcherScheduler periodically updates the search index.
type SearcherScheduler struct {
	log      *slog.Logger
	searcher core.Searcher
	interval time.Duration
}

// NewSearcherScheduler creates a new scheduler for periodic index updates.
func NewSearcherScheduler(log *slog.Logger, searcher core.Searcher, interval time.Duration) *SearcherScheduler {
	return &SearcherScheduler{
		log:      log,
		searcher: searcher,
		interval: interval,
	}
}

// Start begins periodic index updates at configured interval.
func (s *SearcherScheduler) Start(ctx context.Context) error {
	s.log.Info("start searcher scheduler")
	if err := s.searcher.UpdateIndex(ctx); err != nil {
		return err
	}
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.searcher.UpdateIndex(ctx); err != nil {
					s.log.Error("failed to update index", "error", err)
				}
			case <-ctx.Done():
				s.log.Info("index updater stopped")
				return
			}
		}
	}()
	return nil
}

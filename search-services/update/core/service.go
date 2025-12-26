package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"
)

type Service struct {
	log         *slog.Logger
	db          DB
	xkcd        XKCD
	words       Words
	publisher   Publisher
	concurrency int
	inProgress  atomic.Bool
}

func NewService(
	log *slog.Logger, db DB, xkcd XKCD, words Words, publisher Publisher, concurrency int,
) (*Service, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("wrong concurrency specified: %d", concurrency)
	}
	return &Service{
		log:         log,
		db:          db,
		xkcd:        xkcd,
		words:       words,
		publisher:   publisher,
		concurrency: concurrency,
	}, nil
}

func (s *Service) Stats(ctx context.Context) (ServiceStats, error) {
	stats, err := s.db.Stats(ctx)
	if err != nil {
		s.log.Error("failed to get database stats", "error", err)
		return ServiceStats{}, fmt.Errorf("failed to get database stats: %w", err)
	}
	lastID, err := s.xkcd.LastID(ctx)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			s.log.Warn("last comic not found from xkcd API")
		} else {
			s.log.Error("failed to get last comic from xkcd API", "error", err)
		}
		return ServiceStats{}, fmt.Errorf("failed to get last comic from xkcd API: %w", err)
	}
	return ServiceStats{
		DBStats:     stats,
		ComicsTotal: lastID,
	}, nil
}

func (s *Service) Status(ctx context.Context) ServiceStatus {
	if s.inProgress.Load() {
		return StatusRunning
	}
	return StatusIdle
}

func (s *Service) Update(ctx context.Context) error {
	if !s.inProgress.CompareAndSwap(false, true) {
		return ErrAlreadyExists
	}
	defer s.inProgress.Store(false)

	s.log.Info("update started")
	defer func(start time.Time) {
		s.log.Info("update finished", "duration", time.Since(start))
	}(time.Now())

	// get existing IDs in DB
	IDs, err := s.db.IDs(ctx)
	if err != nil {
		s.log.Error("failed to get existing IDs in DB", "error", err)
		return fmt.Errorf("failed to get existing IDs in DB: %w", err)
	}
	s.log.Debug("existing comics in DB", "count", len(IDs))
	exists := make(map[int64]bool, len(IDs))
	for _, id := range IDs {
		exists[id] = true
	}

	// get last comics ID
	lastID, err := s.xkcd.LastID(ctx)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			s.log.Warn("last comic not found from xkcd API")
		} else {
			s.log.Error("failed to get last comic from xkcd API", "error", err)
		}
		return fmt.Errorf("failed to get last comic from xkcd API: %w", err)
	}
	s.log.Debug("last comics ID in XKCD", "id", lastID)

	jobs := make(chan int64, lastID)
	results := make(chan *Comic, lastID)

	for w := 1; w <= s.concurrency; w++ {
		go s.worker(ctx, jobs, results)
	}

	var jobCount int64
	for id := int64(1); id <= lastID; id++ {
		if !exists[id] {
			jobs <- id
			jobCount++
		}
	}
	close(jobs)

	var comics []Comic
	for range jobCount {
		comic := <-results
		if comic != nil {
			comics = append(comics, *comic)
		}
	}

	if len(comics) == 0 {
		s.log.Debug("no new comics to add")
		return nil
	}

	// batch-запись извлеченных комиксов
	if err := s.db.Add(ctx, comics...); err != nil {
		s.log.Error("failed to add comics", "error", err)
		return fmt.Errorf("failed to add comics: %w", err)
	}
	s.log.Debug("added new comics", "counter", len(comics))

	// отправка сообщения через брокер-Nats после успешного обновления
	if err := s.publisher.Publish(EventUpdate); err != nil {
		s.log.Error("failed to publish", "error", err)
	}
	return nil
}

func (s *Service) worker(ctx context.Context, jobs <-chan int64, results chan<- *Comic) {
	for id := range jobs {
		// special case
		if id == 404 {
			results <- &Comic{
				ID:    id,
				Words: []string{},
			}
			continue
		}

		info, err := s.xkcd.Get(ctx, id)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				s.log.Debug("comic not found", "comic_id", id)
			} else {
				s.log.Error("failed to get XKCDInfo", "comic_id", id, "error", err)
			}
			results <- nil
			continue
		}

		keywords, err := s.words.Norm(ctx, makeDescription(info))
		if err != nil {
			s.log.Error("failed to normalize comic description", "comic_id", id, "error", err)
			results <- nil
			continue
		}
		results <- &Comic{
			ID:    info.ID,
			URL:   info.URL,
			Words: keywords,
		}
	}
}

func makeDescription(info XKCDInfo) string {
	return strings.Join([]string{
		info.SafeTitle,
		info.Title,
		info.Transcript,
		info.Alt,
	}, " ")
}

func (s *Service) Drop(ctx context.Context) error {
	if !s.inProgress.CompareAndSwap(false, true) {
		return ErrAlreadyExists
	}
	defer s.inProgress.Store(false)

	err := s.db.Drop(ctx)
	if err != nil {
		s.log.Error("failed to drop db entries", "error", err)
		return fmt.Errorf("failed to drop db entries: %w", err)
	}
	// отправка сообщения через брокер-Nats после успешного "обнулениия" базы
	if err := s.publisher.Publish(EventReset); err != nil {
		s.log.Error("failed to publish", "error", err)
	}
	return nil
}

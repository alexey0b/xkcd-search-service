package core

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

// Service implements search functionality with in-memory inverted index.
type Service struct {
	log     *slog.Logger
	metrics MetricsCollector

	db    DB
	words Words

	index map[string][]int64 // inverted index: word -> comic IDs
	lock  sync.RWMutex
}

// comicRank holds comic ranking information for search results.
type comicRank struct {
	Comic
	matched int64 // number of matched keywords
	total   int64 // total number of words in comic
}

// NewService creates a new search service instance.
func NewService(
	log *slog.Logger, metrics MetricsCollector, db DB, words Words,
) (*Service, error) {
	if metrics == nil {
		metrics = &NoopMetricsCollector{}
	}
	return &Service{
		log:     log,
		metrics: metrics,
		db:      db,
		words:   words,
		index:   map[string][]int64{},
	}, nil
}

// Search performs ranked search from all comics data.
func (s *Service) Search(ctx context.Context, phrase string, limit int64) ([]Comic, error) {
	if phrase == "" || limit <= 0 {
		return nil, ErrBadArguments
	}

	s.log.Info("search started")
	defer func(start time.Time) {
		s.log.Info("search finished", "duration", time.Since(start))
	}(time.Now())

	keywords, err := s.words.Norm(ctx, phrase)
	if err != nil {
		s.log.Error("failed to normalized phrase", "error", err)
		return nil, fmt.Errorf("failed to normalized phrase: %w", err)
	}
	setOfPhrase := map[string]bool{}
	for _, word := range keywords {
		setOfPhrase[word] = true
	}

	comicsInfo, err := s.db.GetAllComicsInfo(ctx)
	if err != nil {
		s.log.Error("failed to get all comics info", "error", err)
		return nil, fmt.Errorf("failed to get all comics: %w", err)
	}
	return s.rankedSearch(comicsInfo, setOfPhrase, limit), nil
}

// rankedSearch ranks comics by keyword matches and returns top results.
func (s *Service) rankedSearch(comicsInfo []ComicInfo, setOfPhrase map[string]bool, limit int64) []Comic {
	if len(comicsInfo) == 0 {
		return []Comic{}
	}

	var comicsRanks []comicRank
	for _, comic := range comicsInfo {
		var matched int64
		for _, word := range comic.Words {
			if setOfPhrase[word] {
				matched++
			}
		}
		if matched == 0 {
			continue
		}
		comicsRanks = append(comicsRanks, comicRank{
			Comic:   comic.Comic,
			matched: matched,
			total:   int64(len(comic.Words)),
		})
	}
	if len(comicsRanks) == 0 {
		return []Comic{}
	}

	// sort by descending priority:
	// 1. number of absolute matches
	// 2. matched/total ratio
	sort.Slice(comicsRanks, func(i, j int) bool {
		if comicsRanks[i].matched != comicsRanks[j].matched {
			return comicsRanks[i].matched > comicsRanks[j].matched
		}
		crossI := comicsRanks[i].matched * comicsRanks[j].total
		crossJ := comicsRanks[j].matched * comicsRanks[i].total
		return crossI > crossJ
	})

	limit = min(int64(len(comicsRanks)), limit)
	rankedComics := make([]Comic, limit)
	for i, comicRank := range comicsRanks[:limit] {
		rankedComics[i] = comicRank.Comic
	}
	s.log.Debug("search results",
		"relevant", len(comicsRanks),
		"returned", limit,
	)
	return rankedComics
}

// ISearch performs indexed search using inverted index.
func (s *Service) ISearch(ctx context.Context, phrase string, limit int64) ([]Comic, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if phrase == "" || limit <= 0 {
		return nil, ErrBadArguments
	}

	s.log.Info("isearch started")
	defer func(start time.Time) {
		s.log.Info("isearch finished", "duration", time.Since(start))
	}(time.Now())

	keywords, err := s.words.Norm(ctx, phrase)
	if err != nil {
		s.log.Error("failed to normalized phrase", "error", err)
		return nil, fmt.Errorf("failed to normalized phrase: %w", err)
	}

	scores := map[int64]int{}
	uniqueIDs := []int64{}
	for _, keyword := range keywords {
		for _, id := range s.index[keyword] {
			if _, ok := scores[id]; !ok {
				uniqueIDs = append(uniqueIDs, id)
			}
			scores[id]++
		}
		s.log.Debug("found comic ids for keyword", "keyword", keyword, "count", len(s.index[keyword]))
	}

	comics, err := s.db.GetComicsByIds(ctx, uniqueIDs)
	if err != nil {
		s.log.Error("failed to get comics by comics ids", "error", err)
		return nil, fmt.Errorf("failed to get comics by comics ids: %w", err)
	}

	// sort by descending number of matches
	sort.Slice(comics, func(i, j int) bool {
		return scores[comics[i].ID] > scores[comics[j].ID]
	})

	limit = min(int64(len(comics)), limit)
	s.log.Debug("isearch results",
		"relevant", len(comics),
		"returned", limit,
	)
	return comics[:limit], nil
}

// UpdateIndex rebuilds the inverted index from database.
func (s *Service) UpdateIndex(ctx context.Context) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.log.Info("update index started")
	defer func(start time.Time) {
		s.log.Info("update index finished", "duration", time.Since(start))
	}(time.Now())

	comicsInfo, err := s.db.GetAllComicsInfo(ctx)
	if err != nil {
		s.log.Error("failed to get all comics", "error", err)
		return fmt.Errorf("failed to get all comics info: %w", err)
	}

	clear(s.index)

	for _, comicInfo := range comicsInfo {
		for _, keyword := range comicInfo.Words {
			s.index[keyword] = append(s.index[keyword], comicInfo.ID)
		}
	}

	s.metrics.SetIndexSize(int64(len(s.index)))
	s.metrics.SetIndexLastUpdateTimestamp()
	return nil
}

// ResetIndex clears the inverted index.
func (s *Service) ResetIndex() {
	s.lock.Lock()
	defer s.lock.Unlock()
	clear(s.index)
	s.log.Info("index has been reset")

	s.metrics.SetIndexSize(0)
	s.metrics.SetIndexLastUpdateTimestamp()
}

// HandleEvent processes events from message broker.
func (s *Service) HandleEvent(ctx context.Context, eventType EventType) error {
	switch eventType {
	case EventUpdate:
		if err := s.UpdateIndex(ctx); err != nil {
			return fmt.Errorf("failed to update index: %w", err)
		}
	case EventReset:
		s.ResetIndex()
	default:
		s.log.Warn("unknown event type", "event", string(eventType))
	}
	return nil
}

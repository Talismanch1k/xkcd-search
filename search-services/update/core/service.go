package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"unicode/utf8"
)

const maxMessageSize = 4 * 1024

type Service struct {
	running     atomic.Bool
	db          DB
	xkcd        XKCD
	words       Words
	notifier    Notifier
	concurrency int
}

func NewService(db DB, xkcd XKCD, words Words, notifier Notifier, concurrency int) (*Service, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("wrong concurrency specified: %d", concurrency)
	}
	return &Service{
		db:          db,
		xkcd:        xkcd,
		words:       words,
		notifier:    notifier,
		concurrency: concurrency,
	}, nil
}

func (s *Service) Update(ctx context.Context) (err error) {
	if !s.running.CompareAndSwap(false, true) {
		return ErrAlreadyExists
	}
	defer s.running.Store(false)

	lastID, err := s.xkcd.LastID(ctx)
	if err != nil {
		return fmt.Errorf("get last id: %w", err)
	}

	existingIDs, err := s.db.IDs(ctx)
	if err != nil {
		return fmt.Errorf("get existing ids: %w", err)
	}

	existing := make(map[int]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existing[id] = struct{}{}
	}

	var missingIDs []int
	for id := 1; id <= lastID; id++ {
		if _, ok := existing[id]; !ok {
			missingIDs = append(missingIDs, id)
		}
	}
	if len(missingIDs) == 0 {
		slog.Info("database is up to date")
		return nil
	}

	slog.Info("starting update", "missing", len(missingIDs), "total", lastID)

	// worker pool ^-^
	jobs := make(chan int, s.concurrency)

	var wg sync.WaitGroup
	for range s.concurrency {
		wg.Go(func() {
			for id := range jobs {
				if ctx.Err() != nil {
					return
				}

				if err := s.processComic(ctx, id); err != nil {
					if errors.Is(err, ErrNotFound) {
						if id == 404 {
							slog.Debug("comic 404 not found, save as empty 0_0", "id", id)
							err := s.db.Add(ctx, Comics{
								ID:         id,
								URL:        "https://xkcd.com/404/",
								Title:      []string{"404"},
								Alt:        []string{"404"},
								Transcript: []string{"404"},
							})
							if err != nil {
								slog.Error("save empty comic", "id", id, "err", err)
							}
						} else {
							slog.Debug("comic not found, skip", "id", id)
						}
						continue
					}
					slog.Error("process comic", "id", id, "err", err)
				}
			}
		})
	}

	for _, id := range missingIDs {
		select {
		case jobs <- id:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return ctx.Err()
		}
	}

	close(jobs)
	wg.Wait()

	slog.Info("update finished")

	if err := s.notifier.DBUpdated(ctx); err != nil {
		slog.Error("notify db updated", "err", err)
	}

	return nil
}

func (s *Service) processComic(ctx context.Context, id int) error {
	info, err := s.xkcd.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("get comic %d: %w", id, err)
	}

	// maybe make parallel requests?
	normTitle, err := s.words.Norm(ctx, trim(info.Title, maxMessageSize))
	if err != nil {
		return fmt.Errorf("normalize comic %d: %w", id, err)
	}
	normTitle = cleanWords(normTitle)

	normTranscript, err := s.words.Norm(ctx, trim(info.Transcript, maxMessageSize))
	if err != nil {
		return fmt.Errorf("normalize comic %d: %w", id, err)
	}
	normTranscript = cleanWords(normTranscript)

	normAlt, err := s.words.Norm(ctx, trim(info.Alt, maxMessageSize))
	if err != nil {
		return fmt.Errorf("normalize comic %d: %w", id, err)
	}
	normAlt = cleanWords(normAlt)

	err = s.db.Add(ctx, Comics{
		ID:         info.ID,
		URL:        info.URL,
		Title:      normTitle,
		Alt:        normAlt,
		Transcript: normTranscript,
	})
	if err != nil {
		return fmt.Errorf("save comic %d: %w", id, err)
	}

	slog.Debug("processed comic", "id", id)
	return nil
}

func (s *Service) Stats(ctx context.Context) (ServiceStats, error) {
	dbStats, err := s.db.Stats(ctx)
	if err != nil {
		return ServiceStats{}, fmt.Errorf("get db stats: %w", err)
	}

	lastID, err := s.xkcd.LastID(ctx)
	if err != nil {
		return ServiceStats{}, fmt.Errorf("get last id: %w", err)
	}

	return ServiceStats{
		DBStats:     dbStats,
		ComicsTotal: lastID,
	}, nil
}

func (s *Service) Status(ctx context.Context) ServiceStatus {
	if s.running.Load() {
		return StatusRunning
	}
	return StatusIdle
}

func (s *Service) Drop(ctx context.Context) error {
	if err := s.db.Drop(ctx); err != nil {
		return fmt.Errorf("drop db: %w", err)
	}

	if err := s.notifier.DBDropped(ctx); err != nil {
		slog.Error("notify db dropped", "err", err)
	}

	return nil
}

// trim - trims the string to a specified number of bytes
func trim(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	// prevent cutting half symbol
	for limit > 0 && !utf8.RuneStart(s[limit]) {
		limit--
	}
	return s[:limit]
}

func cleanWords(words []string) []string {
	if words == nil {
		return []string{}
	}
	return words
}

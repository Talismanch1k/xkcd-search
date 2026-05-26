package core

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
"slices"
	"sync"
	"unicode/utf8"
)

const maxMessageSize = 4 * 1024

type Service struct {
	db     DB
	words  Words
	scorer Scorer

	lastID int
	mu     sync.RWMutex
	index  map[string][]entry
	comics map[int]Comic

	// reusing ram for scores
	scorePools sync.Pool
	idPools    sync.Pool
}

type entry struct {
	ID    int
	Score int
}

type scoredComic struct {
	comic Comic
	score int
}

func NewService(db DB, words Words, scorer Scorer) *Service {
	return &Service{
		db:     db,
		words:  words,
		scorer: scorer,
		index:  make(map[string][]entry),
		comics: make(map[int]Comic),
		scorePools: sync.Pool{
			New: func() any {
				s := make([]int, 0, 1024)
				return &s
			},
		},
		idPools: sync.Pool{
			New: func() any {
				s := make([]int, 0, 1024)
				return &s
			},
		},
	}
}

func (s *Service) Search(ctx context.Context, query string, limit int32) ([]Comic, error) {
	if limit < 1 {
		return nil, ErrBadArguments
	}
	normQuery, err := s.words.Norm(ctx, trim(query, maxMessageSize))
	if err != nil {
		return nil, fmt.Errorf("norm query: %w", err)
	}
	if len(normQuery) == 0 {
		return nil, nil
	}

	matches, err := s.db.Search(ctx, normQuery)
	if err != nil {
		return nil, fmt.Errorf("search in database: %w", err)
	}
	if len(matches) == 0 {
		return nil, nil
	}

	scored := make([]scoredComic, 0, len(matches))
	for _, c := range matches {
		scored = append(scored,
			scoredComic{comic: c, score: s.scorer.Score(c, normQuery)},
		)
	}

	slices.SortFunc(scored, func(a, b scoredComic) int {
		if a.score != b.score {
			return cmp.Compare(b.score, a.score)
		}
		return cmp.Compare(b.comic.ID, a.comic.ID)
	})

	if int32(len(scored)) > limit {
		scored = scored[:limit]
	}

	res := make([]Comic, 0, len(scored))
	for _, sc := range scored {
		res = append(res, sc.comic)
	}

	return res, nil
}

func (s *Service) ISearch(ctx context.Context, query string, limit int32) ([]Comic, error) {
	if limit < 1 {
		return nil, ErrBadArguments
	}
	normQuery, err := s.words.Norm(ctx, trim(query, maxMessageSize))
	if err != nil {
		return nil, fmt.Errorf("norm query: %w", err)
	}
	if len(normQuery) == 0 {
		return nil, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	maxID := s.lastID + 1

	scoresPtr := s.acquireScores(maxID)
	scores := *scoresPtr

	modifiedIDsPtr := s.acquireIDs()
	defer s.releaseIDs(modifiedIDsPtr)
	modifiedIDs := *modifiedIDsPtr

	defer func() {
		for _, id := range *modifiedIDsPtr {
			scores[id] = 0
		}
		s.releaseScores(scoresPtr)
	}()

	for _, w := range normQuery {
		for _, e := range s.index[w] {
			if scores[e.ID] == 0 {
				modifiedIDs = append(modifiedIDs, e.ID)
			}
			scores[e.ID] += e.Score
		}
	}

	// update if reallocated
	*modifiedIDsPtr = modifiedIDs

	if len(modifiedIDs) == 0 {
		return nil, nil
	}

	// minheap ?
	slices.SortFunc(modifiedIDs, func(a, b int) int {
		if scores[a] != scores[b] {
			return cmp.Compare(scores[b], scores[a])
		}
		return cmp.Compare(b, a)
	})

	if int32(len(modifiedIDs)) > limit {
		modifiedIDs = modifiedIDs[:limit]
	}

	res := make([]Comic, 0, len(modifiedIDs))
	for _, id := range modifiedIDs {
		res = append(res, s.comics[id])
	}

	return res, nil
}

func (s *Service) GetComic(_ context.Context, id int) (Comic, error) {
	s.mu.RLock()
	c, ok := s.comics[id]
	s.mu.RUnlock()
	if !ok {
		return Comic{}, ErrNotFound
	}
	return c, nil
}

// BuildIndex - incrementally update index
func (s *Service) BuildIndex(ctx context.Context) error {
	ids, err := s.db.IDs(ctx)
	if err != nil {
		return fmt.Errorf("get db ids: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	var missingIDs []int
	for _, id := range ids {
		if _, ok := s.comics[id]; !ok {
			missingIDs = append(missingIDs, id)
		}
	}

	if len(missingIDs) == 0 {
		return nil
	}

	comics, err := s.db.ListIDs(ctx, missingIDs)
	if err != nil {
		return fmt.Errorf("fetch missing comics: %w", err)
	}

	for _, c := range comics {
		s.comics[c.ID] = c

		termScores := s.scorer.ExtractScores(c)

		for w, sc := range termScores {
			s.index[w] = append(s.index[w], entry{ID: c.ID, Score: sc})
		}

		s.lastID = max(s.lastID, c.ID)
	}
	slog.Info("index updated", "new comics", len(comics), "total comics", len(s.comics))
	return nil
}

func (s *Service) DropIndex() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.index = make(map[string][]entry)
	s.comics = make(map[int]Comic)
	s.lastID = 0

	slog.Info("search index cleared")
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

func (s *Service) acquireScores(size int) *[]int {
	ptr := s.scorePools.Get().(*[]int)
	arr := *ptr

	if cap(arr) < size {
		arr = make([]int, size)
	} else {
		arr = arr[:size]
	}

	*ptr = arr
	return ptr
}

func (s *Service) releaseScores(ptr *[]int) {
	s.scorePools.Put(ptr)
}

func (s *Service) acquireIDs() *[]int {
	ptr := s.idPools.Get().(*[]int)
	arr := (*ptr)[:0]
	*ptr = arr
	return ptr
}

func (s *Service) releaseIDs(ptr *[]int) {
	s.idPools.Put(ptr)
}

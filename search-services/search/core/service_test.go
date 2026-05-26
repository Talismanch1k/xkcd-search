package core_test

import (
	"context"
	"errors"
	"testing"

	"yadro.com/course/search/core"
)

type mockDB struct {
	searchFn  func(ctx context.Context, words []string) ([]core.Comic, error)
	idsFn     func(ctx context.Context) ([]int, error)
	listIDsFn func(ctx context.Context, ids []int) ([]core.Comic, error)
}

func (m *mockDB) Search(ctx context.Context, words []string) ([]core.Comic, error) {
	return m.searchFn(ctx, words)
}

func (m *mockDB) IDs(ctx context.Context) ([]int, error) {
	return m.idsFn(ctx)
}

func (m *mockDB) ListIDs(ctx context.Context, ids []int) ([]core.Comic, error) {
	return m.listIDsFn(ctx, ids)
}

type mockWords struct {
	normFn func(ctx context.Context, phrase string) ([]string, error)
}

func (m *mockWords) Norm(ctx context.Context, phrase string) ([]string, error) {
	return m.normFn(ctx, phrase)
}

type mockScorer struct {
	scoreFn         func(c core.Comic, words []string) int
	extractScoresFn func(c core.Comic) map[string]int
}

func (m *mockScorer) Score(c core.Comic, words []string) int {
	return m.scoreFn(c, words)
}

func (m *mockScorer) ExtractScores(c core.Comic) map[string]int {
	return m.extractScoresFn(c)
}

func comicIDs(comics []core.Comic) []int {
	ids := make([]int, 0, len(comics))
	for _, c := range comics {
		ids = append(ids, c.ID)
	}
	return ids
}

func TestSearch_BadLimit(t *testing.T) {
	t.Parallel()

	svc := core.NewService(nil, nil, nil)
	tests := []struct {
		name  string
		limit int32
	}{
		{"zero", 0},
		{"negative one", -1},
		{"negative hundred", -100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.Search(t.Context(), "hello", tt.limit)
			if !errors.Is(err, core.ErrBadArguments) {
				t.Errorf("want %v, got %v", core.ErrBadArguments, err)
			}
		})
	}
}

func TestSearch_NormError(t *testing.T) {
	t.Parallel()

	normErr := errors.New("norm failed")
	svc := core.NewService(
		nil,
		&mockWords{normFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, normErr
		}},
		nil,
	)

	_, err := svc.Search(t.Context(), "query", 10)
	if !errors.Is(err, normErr) {
		t.Errorf("want wrapped normErr, got %v", err)
	}
}

func TestSearch_EmptyNormResult(t *testing.T) {
	t.Parallel()

	svc := core.NewService(
		nil,
		&mockWords{normFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{}, nil
		}},
		nil,
	)

	res, err := svc.Search(t.Context(), "   ", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Errorf("want %v, got %v", nil, res)
	}
}

func TestSearch_DBError(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("db error")
	svc := core.NewService(
		&mockDB{searchFn: func(_ context.Context, _ []string) ([]core.Comic, error) {
			return nil, dbErr
		}},
		&mockWords{normFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"hello"}, nil
		}},
		nil,
	)

	_, err := svc.Search(t.Context(), "hello", 10)
	if !errors.Is(err, dbErr) {
		t.Errorf("want wrapped dbErr, got %v", err)
	}
}

func TestSearch_EmptyDBResult(t *testing.T) {
	t.Parallel()

	svc := core.NewService(
		&mockDB{searchFn: func(_ context.Context, _ []string) ([]core.Comic, error) {
			return []core.Comic{}, nil
		}},
		&mockWords{normFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"hello"}, nil
		}},
		nil,
	)

	res, err := svc.Search(t.Context(), "hello", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Errorf("want nil, got %v", res)
	}
}

func TestSearch_SortedByScoreThenIDDesc(t *testing.T) {
	t.Parallel()

	comics := []core.Comic{
		{ID: 1}, {ID: 2}, {ID: 3},
	}

	scores := map[int]int{1: 5, 2: 10, 3: 10}

	svc := core.NewService(
		&mockDB{searchFn: func(_ context.Context, _ []string) ([]core.Comic, error) {
			return comics, nil
		}},
		&mockWords{normFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"w"}, nil
		}},
		&mockScorer{scoreFn: func(c core.Comic, _ []string) int {
			return scores[c.ID]
		}},
	)

	res, err := svc.Search(t.Context(), "query", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantOrder := []int{3, 2, 1}
	got := comicIDs(res)
	if len(got) != len(wantOrder) {
		t.Fatalf("want %d results, got %d", len(wantOrder), len(got))
	}
	for i, want := range wantOrder {
		if got[i] != want {
			t.Errorf("pos %d: want ID=%d, got ID=%d", i, want, got[i])
		}
	}
}

func TestSearch_LimitTruncates(t *testing.T) {
	t.Parallel()

	comics := []core.Comic{{ID: 1}, {ID: 2}, {ID: 3}}

	svc := core.NewService(
		&mockDB{searchFn: func(_ context.Context, _ []string) ([]core.Comic, error) {
			return comics, nil
		}},
		&mockWords{normFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"w"}, nil
		}},
		&mockScorer{scoreFn: func(c core.Comic, _ []string) int { return 1 }},
	)

	res, err := svc.Search(t.Context(), "query", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("want %d results, got %d", 2, len(res))
	}
}

func TestISearch_BadLimit(t *testing.T) {
	t.Parallel()

	svc := core.NewService(nil, nil, nil)
	tests := []struct {
		name  string
		limit int32
	}{
		{"zero", 0},
		{"negative one", -1},
		{"negative hundred", -100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.ISearch(t.Context(), "hello", tt.limit)
			if !errors.Is(err, core.ErrBadArguments) {
				t.Errorf("want %v, got %v", core.ErrBadArguments, err)
			}
		})
	}
}

func TestISearch_NormError(t *testing.T) {
	t.Parallel()

	normErr := errors.New("norm failed")
	svc := core.NewService(
		nil,
		&mockWords{normFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, normErr
		}},
		nil,
	)
	_, err := svc.ISearch(t.Context(), "hello", 10)
	if !errors.Is(err, normErr) {
		t.Errorf("want wrapped normErr, got %v", err)
	}
}

func TestISearch_EmptyNormResult(t *testing.T) {
	t.Parallel()

	svc := core.NewService(
		nil,
		&mockWords{normFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, nil
		}},
		nil,
	)
	res, err := svc.ISearch(t.Context(), "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Errorf("want nil, got %v", res)
	}
}

func TestISearch_NoMatchInIndex(t *testing.T) {
	t.Parallel()

	svc := core.NewService(
		nil,
		&mockWords{normFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{"missing"}, nil
		}},
		nil,
	)
	res, err := svc.ISearch(t.Context(), "missing", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Errorf("want nil, got %v", res)
	}
}

func TestISearch_ReturnsIndexedComics(t *testing.T) {
	t.Parallel()

	comics := []core.Comic{
		{ID: 1, Title: []string{"hello"}},
		{ID: 2, Title: []string{"hello", "world"}},
		{ID: 3, Title: []string{"world"}},
	}
	termScores := map[int]map[string]int{
		1: {"hello": 5},
		2: {"hello": 5, "world": 3},
		3: {"world": 3},
	}

	db := &mockDB{
		idsFn: func(_ context.Context) ([]int, error) {
			return []int{1, 2, 3}, nil
		},
		listIDsFn: func(_ context.Context, ids []int) ([]core.Comic, error) {
			var res []core.Comic
			for _, c := range comics {
				for _, id := range ids {
					if c.ID == id {
						res = append(res, c)
					}
				}
			}
			return res, nil
		},
	}
	scorer := &mockScorer{
		extractScoresFn: func(c core.Comic) map[string]int {
			return termScores[c.ID]
		},
	}
	words := &mockWords{normFn: func(_ context.Context, phrase string) ([]string, error) {
		if phrase == "hello" {
			return []string{"hello"}, nil
		}
		return []string{phrase}, nil
	}}

	svc := core.NewService(db, words, scorer)
	if err := svc.BuildIndex(t.Context()); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	res, err := svc.ISearch(t.Context(), "hello", 10)
	if err != nil {
		t.Fatalf("ISearch: %v", err)
	}

	if len(res) != 2 {
		t.Fatalf("want %d comics, got %d", 2, len(res))
	}

	if res[0].ID != 2 || res[1].ID != 1 {
		t.Errorf("want order %v, got %v", []int{2, 1}, comicIDs(res))
	}
}

func TestISearch_LimitTruncates(t *testing.T) {
	t.Parallel()

	comics := []core.Comic{
		{ID: 1, Title: []string{"hello"}},
		{ID: 2, Title: []string{"hello"}},
		{ID: 3, Title: []string{"hello"}},
	}
	termScores := map[int]map[string]int{
		1: {"hello": 5},
		2: {"hello": 5},
		3: {"hello": 5},
	}

	db := &mockDB{
		idsFn:     func(_ context.Context) ([]int, error) { return []int{1, 2, 3}, nil },
		listIDsFn: func(_ context.Context, _ []int) ([]core.Comic, error) { return comics, nil },
	}
	scorer := &mockScorer{
		extractScoresFn: func(c core.Comic) map[string]int { return termScores[c.ID] },
	}
	words := &mockWords{normFn: func(_ context.Context, _ string) ([]string, error) {
		return []string{"hello"}, nil
	}}

	svc := core.NewService(db, words, scorer)
	if err := svc.BuildIndex(t.Context()); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	res, err := svc.ISearch(t.Context(), "hello", 2)
	if err != nil {
		t.Fatalf("ISearch: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("want %d, got %d", 2, len(res))
	}
}

func TestBuildIndex_IDsError(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("ids error")
	svc := core.NewService(
		&mockDB{idsFn: func(_ context.Context) ([]int, error) { return nil, dbErr }},
		nil, nil,
	)
	err := svc.BuildIndex(t.Context())
	if !errors.Is(err, dbErr) {
		t.Errorf("want wrapped dbErr, got %v", err)
	}
}

func TestBuildIndex_ListIDsError(t *testing.T) {
	t.Parallel()

	listErr := errors.New("list error")
	svc := core.NewService(
		&mockDB{
			idsFn:     func(_ context.Context) ([]int, error) { return []int{1}, nil },
			listIDsFn: func(_ context.Context, _ []int) ([]core.Comic, error) { return nil, listErr },
		},
		nil, nil,
	)

	err := svc.BuildIndex(t.Context())
	if !errors.Is(err, listErr) {
		t.Errorf("want wrapped listErr, got %v", err)
	}
}

func TestBuildIndex_IncrementalUpdate(t *testing.T) {
	t.Parallel()

	firstBatch := []core.Comic{{ID: 1, Title: []string{"first"}}}
	secondBatch := []core.Comic{{ID: 2, Title: []string{"second"}}}

	callCount := 0
	db := &mockDB{
		idsFn: func(_ context.Context) ([]int, error) {
			callCount++
			if callCount == 1 {
				return []int{1}, nil
			}
			return []int{1, 2}, nil
		},
		listIDsFn: func(_ context.Context, ids []int) ([]core.Comic, error) {
			if len(ids) == 1 && ids[0] == 1 {
				return firstBatch, nil
			}
			return secondBatch, nil
		},
	}
	scorer := &mockScorer{
		extractScoresFn: func(c core.Comic) map[string]int {
			if c.ID == 1 {
				return map[string]int{"first": 5}
			}
			return map[string]int{"second": 5}
		},
	}
	words := &mockWords{normFn: func(_ context.Context, phrase string) ([]string, error) {
		return []string{phrase}, nil
	}}

	svc := core.NewService(db, words, scorer)

	if err := svc.BuildIndex(t.Context()); err != nil {
		t.Fatalf("first BuildIndex: %v", err)
	}
	res, err := svc.ISearch(t.Context(), "first", 10)
	if err != nil || len(res) != 1 {
		t.Fatalf("after first build: want %d comic, got %v (err=%v)", 1, res, err)
	}

	if err := svc.BuildIndex(t.Context()); err != nil {
		t.Fatalf("second BuildIndex: %v", err)
	}
	res, err = svc.ISearch(t.Context(), "second", 10)
	if err != nil || len(res) != 1 {
		t.Fatalf("after second build: want %d comic for 'second', got %v (err=%v)", 1, res, err)
	}
}

func TestBuildIndex_NothingToUpdate(t *testing.T) {
	t.Parallel()

	listCalled := false
	db := &mockDB{
		idsFn:     func(_ context.Context) ([]int, error) { return []int{}, nil },
		listIDsFn: func(_ context.Context, _ []int) ([]core.Comic, error) { listCalled = true; return nil, nil },
	}

	svc := core.NewService(db, nil, nil)
	if err := svc.BuildIndex(t.Context()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if listCalled {
		t.Error("ListIDs should not be called when there is nothing to update")
	}
}

func TestDropIndex_ClearsIndex(t *testing.T) {
	t.Parallel()

	comics := []core.Comic{{ID: 1, Title: []string{"hello"}}}
	db := &mockDB{
		idsFn:     func(_ context.Context) ([]int, error) { return []int{1}, nil },
		listIDsFn: func(_ context.Context, _ []int) ([]core.Comic, error) { return comics, nil },
	}
	scorer := &mockScorer{
		extractScoresFn: func(c core.Comic) map[string]int { return map[string]int{"hello": 5} },
	}
	words := &mockWords{normFn: func(_ context.Context, phrase string) ([]string, error) {
		return []string{phrase}, nil
	}}

	svc := core.NewService(db, words, scorer)
	if err := svc.BuildIndex(t.Context()); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	res, err := svc.ISearch(t.Context(), "hello", 10)
	if err != nil || len(res) != 1 {
		t.Fatalf("before drop: want %d comic, got %v (err=%v)", 1, res, err)
	}

	svc.DropIndex()

	res, err = svc.ISearch(t.Context(), "hello", 10)
	if err != nil {
		t.Fatalf("after drop: unexpected error: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("after drop: want %d, got %v", 0, res)
	}
}

func TestDropIndex_ThenRebuild(t *testing.T) {
	t.Parallel()

	comics := []core.Comic{{ID: 1, Title: []string{"hello"}}}
	db := &mockDB{
		idsFn:     func(_ context.Context) ([]int, error) { return []int{1}, nil },
		listIDsFn: func(_ context.Context, _ []int) ([]core.Comic, error) { return comics, nil },
	}
	scorer := &mockScorer{
		extractScoresFn: func(c core.Comic) map[string]int { return map[string]int{"hello": 5} },
	}
	words := &mockWords{normFn: func(_ context.Context, phrase string) ([]string, error) {
		return []string{phrase}, nil
	}}

	svc := core.NewService(db, words, scorer)

	if err := svc.BuildIndex(t.Context()); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	svc.DropIndex()
	if err := svc.BuildIndex(t.Context()); err != nil {
		t.Fatalf("BuildIndex after drop: %v", err)
	}

	res, err := svc.ISearch(t.Context(), "hello", 10)
	if err != nil || len(res) != 1 {
		t.Errorf("after rebuild: want %d comic, got %v (err=%v)", 1, res, err)
	}
}

func TestISearch_SortedByScore(t *testing.T) {
	t.Parallel()

	comics := []core.Comic{
		{ID: 1},
		{ID: 2},
	}
	termScores := map[int]map[string]int{
		1: {"hello": 5},
		2: {"hello": 10},
	}

	db := &mockDB{
		idsFn:     func(_ context.Context) ([]int, error) { return []int{1, 2}, nil },
		listIDsFn: func(_ context.Context, _ []int) ([]core.Comic, error) { return comics, nil },
	}
	scorer := &mockScorer{
		extractScoresFn: func(c core.Comic) map[string]int { return termScores[c.ID] },
	}
	words := &mockWords{normFn: func(_ context.Context, phrase string) ([]string, error) {
		return []string{phrase}, nil
	}}

	svc := core.NewService(db, words, scorer)
	if err := svc.BuildIndex(t.Context()); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	res, err := svc.ISearch(t.Context(), "hello", 10)
	if err != nil {
		t.Fatalf("ISearch: %v", err)
	}

	if len(res) != 2 {
		t.Fatalf("want %d results, got %d", 2, len(res))
	}
	if res[0].ID != 2 || res[1].ID != 1 {
		t.Errorf("want order %v by score, got %v", []int{2, 1}, comicIDs(res))
	}
}

func TestSearch_TrimUTF8(t *testing.T) {
	t.Parallel()

	// 'д' is 0xD0 0xB4
	prefix := make([]byte, 4095)
	for i := range prefix {
		prefix[i] = 'a'
	}
	query := string(prefix) + "д" // total 4097 bytes
	// s[4095] = 0xD0, s[4096] = 0xB4
	// trim(s, 4096) should see s[4096] is a continuation byte and decrement limit to 4095.

	var capturedQuery string
	words := &mockWords{normFn: func(_ context.Context, phrase string) ([]string, error) {
		capturedQuery = phrase
		return nil, nil
	}}

	svc := core.NewService(nil, words, nil)
	_, _ = svc.Search(t.Context(), query, 10)

	if len(capturedQuery) != 4095 {
		t.Errorf("want length %d, got %d", 4095, len(capturedQuery))
	}
	if capturedQuery[len(capturedQuery)-1] != 'a' {
		t.Errorf("last byte should be '%c', got %v", 'a', capturedQuery[len(capturedQuery)-1])
	}
}

func TestAcquireScores_Reallocate(t *testing.T) {
	t.Parallel()

	comics := []core.Comic{{ID: 2000}}
	db := &mockDB{
		idsFn:     func(_ context.Context) ([]int, error) { return []int{2000}, nil },
		listIDsFn: func(_ context.Context, _ []int) ([]core.Comic, error) { return comics, nil },
	}
	scorer := &mockScorer{
		extractScoresFn: func(c core.Comic) map[string]int { return map[string]int{"w": 1} },
	}
	words := &mockWords{normFn: func(_ context.Context, _ string) ([]string, error) {
		return []string{"w"}, nil
	}}

	svc := core.NewService(db, words, scorer)
	if err := svc.BuildIndex(t.Context()); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	// lastID is 2000, so acquireScores(2001) will be called
	res, err := svc.ISearch(t.Context(), "w", 10)
	if err != nil {
		t.Fatalf("ISearch: %v", err)
	}
	if len(res) != 1 || res[0].ID != 2000 {
		t.Errorf("want ID %d, got %v", 2000, comicIDs(res))
	}
}


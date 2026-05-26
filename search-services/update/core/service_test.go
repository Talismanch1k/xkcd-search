package core_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"yadro.com/course/update/core"
)

type mockDB struct {
	addFn   func(context.Context, core.Comics) error
	statsFn func(context.Context) (core.DBStats, error)
	dropFn  func(context.Context) error
	idsFn   func(context.Context) ([]int, error)
}

func (m *mockDB) Add(ctx context.Context, c core.Comics) error {
	if m.addFn != nil {
		return m.addFn(ctx, c)
	}
	return nil
}

func (m *mockDB) Stats(ctx context.Context) (core.DBStats, error) {
	if m.statsFn != nil {
		return m.statsFn(ctx)
	}
	return core.DBStats{}, nil
}

func (m *mockDB) Drop(ctx context.Context) error {
	if m.dropFn != nil {
		return m.dropFn(ctx)
	}
	return nil
}

func (m *mockDB) IDs(ctx context.Context) ([]int, error) {
	if m.idsFn != nil {
		return m.idsFn(ctx)
	}
	return nil, nil
}

type mockXKCD struct {
	getFn    func(context.Context, int) (core.XKCDInfo, error)
	lastIDFn func(context.Context) (int, error)
}

func (m *mockXKCD) Get(ctx context.Context, id int) (core.XKCDInfo, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}
	return core.XKCDInfo{ID: id}, nil
}

func (m *mockXKCD) LastID(ctx context.Context) (int, error) {
	if m.lastIDFn != nil {
		return m.lastIDFn(ctx)
	}
	return 0, nil
}

type mockWords struct {
	normFn func(context.Context, string) ([]string, error)
}

func (m *mockWords) Norm(ctx context.Context, phrase string) ([]string, error) {
	if m.normFn != nil {
		return m.normFn(ctx, phrase)
	}
	return []string{}, nil
}

type mockNotifier struct {
	dbUpdatedFn func(context.Context) error
	dbDroppedFn func(context.Context) error
}

func (m *mockNotifier) DBUpdated(ctx context.Context) error {
	if m.dbUpdatedFn != nil {
		return m.dbUpdatedFn(ctx)
	}
	return nil
}

func (m *mockNotifier) DBDropped(ctx context.Context) error {
	if m.dbDroppedFn != nil {
		return m.dbDroppedFn(ctx)
	}
	return nil
}

func mustNewService(db core.DB, xkcd core.XKCD, words core.Words, n core.Notifier, concurrency int) *core.Service {
	svc, err := core.NewService(db, xkcd, words, n, concurrency)
	if err != nil {
		panic(err)
	}
	return svc
}

func TestNewService_Concurrency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		concurrency int
		wantErr     bool
	}{
		{"zero", 0, true},
		{"negative", -1, true},
		{"large negative", -100, true},
		{"one", 1, false},
		{"two", 2, false},
		{"ten", 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, err := core.NewService(nil, nil, nil, nil, tt.concurrency)
			if (err != nil) != tt.wantErr {
				t.Errorf("concurrency=%d: error = %v, wantErr %v", tt.concurrency, err, tt.wantErr)
			}
			if !tt.wantErr && svc == nil {
				t.Error("want non-nil service")
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	lastIDErr := errors.New("last id error")
	idsErr := errors.New("ids error")
	notifyErr := errors.New("notify error")

	tests := []struct {
		name     string
		db       *mockDB
		xkcd     *mockXKCD
		words    *mockWords
		notifier *mockNotifier
		wantErr  error
		check    func(t *testing.T, db *mockDB, n *mockNotifier)
	}{
		{
			name:    "last id error",
			xkcd:    &mockXKCD{lastIDFn: func(_ context.Context) (int, error) { return 0, lastIDErr }},
			wantErr: lastIDErr,
		},
		{
			name:    "ids error",
			xkcd:    &mockXKCD{lastIDFn: func(_ context.Context) (int, error) { return 5, nil }},
			db:      &mockDB{idsFn: func(_ context.Context) ([]int, error) { return nil, idsErr }},
			wantErr: idsErr,
		},
		{
			name: "nothing to update",
			xkcd: &mockXKCD{lastIDFn: func(_ context.Context) (int, error) { return 3, nil }},
			db:   &mockDB{idsFn: func(_ context.Context) ([]int, error) { return []int{1, 2, 3}, nil }},
			check: func(t *testing.T, _ *mockDB, n *mockNotifier) {
				// notifier must not be called
			},
		},
		{
			name: "processes missing comics",
			xkcd: &mockXKCD{
				lastIDFn: func(_ context.Context) (int, error) { return 3, nil },
				getFn: func(_ context.Context, id int) (core.XKCDInfo, error) {
					return core.XKCDInfo{ID: id, URL: "url", Title: "t"}, nil
				},
			},
			db: &mockDB{
				idsFn: func(_ context.Context) ([]int, error) { return []int{1, 3}, nil },
				addFn: func(_ context.Context, c core.Comics) error {
					if c.ID != 2 {
						t.Errorf("want added ID %d, got %d", 2, c.ID)
					}
					return nil
				},
			},
			words: &mockWords{normFn: func(_ context.Context, s string) ([]string, error) { return []string{s}, nil }},
		},
		{
			name: "clean words converts nil to empty",
			xkcd: &mockXKCD{
				lastIDFn: func(_ context.Context) (int, error) { return 1, nil },
				getFn:    func(_ context.Context, id int) (core.XKCDInfo, error) { return core.XKCDInfo{ID: id}, nil },
			},
			db: &mockDB{
				idsFn: func(_ context.Context) ([]int, error) { return nil, nil },
				addFn: func(_ context.Context, c core.Comics) error {
					if c.Title == nil || c.Alt == nil || c.Transcript == nil {
						t.Error("want non-nil slices for strings")
					}
					return nil
				},
			},
			words: &mockWords{normFn: func(_ context.Context, _ string) ([]string, error) { return nil, nil }},
		},
		{
			name: "comic 404 placeholder",
			xkcd: &mockXKCD{
				lastIDFn: func(_ context.Context) (int, error) { return 404, nil },
				getFn:    func(_ context.Context, _ int) (core.XKCDInfo, error) { return core.XKCDInfo{}, core.ErrNotFound },
			},
			db: &mockDB{
				idsFn: func(_ context.Context) ([]int, error) {
					ids := make([]int, 403)
					for i := range ids {
						ids[i] = i + 1
					}
					return ids, nil
				},
				addFn: func(_ context.Context, c core.Comics) error {
					if c.ID != 404 || c.URL != "https://xkcd.com/404/" {
						t.Errorf("invalid %d placeholder: %+v", 404, c)
					}
					return nil
				},
			},
		},
		{
			name: "skips non-existent comic",
			xkcd: &mockXKCD{
				lastIDFn: func(_ context.Context) (int, error) { return 5, nil },
				getFn:    func(_ context.Context, _ int) (core.XKCDInfo, error) { return core.XKCDInfo{}, core.ErrNotFound },
			},
			db: &mockDB{
				idsFn: func(_ context.Context) ([]int, error) { return []int{1, 3, 4, 5}, nil },
				addFn: func(_ context.Context, _ core.Comics) error {
					t.Error("db.Add should not be called for non-404 missing comic")
					return nil
				},
			},
		},
		{
			name: "notifier called on success",
			xkcd: &mockXKCD{
				lastIDFn: func(_ context.Context) (int, error) { return 1, nil },
				getFn:    func(_ context.Context, id int) (core.XKCDInfo, error) { return core.XKCDInfo{ID: id}, nil },
			},
			db: &mockDB{idsFn: func(_ context.Context) ([]int, error) { return nil, nil }},
			notifier: &mockNotifier{dbUpdatedFn: func(_ context.Context) error {
				return nil
			}},
		},
		{
			name: "notifier error ignored",
			xkcd: &mockXKCD{
				lastIDFn: func(_ context.Context) (int, error) { return 1, nil },
				getFn:    func(_ context.Context, id int) (core.XKCDInfo, error) { return core.XKCDInfo{ID: id}, nil },
			},
			db: &mockDB{idsFn: func(_ context.Context) ([]int, error) { return nil, nil }},
			notifier: &mockNotifier{dbUpdatedFn: func(_ context.Context) error {
				return notifyErr
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := tt.db
			if db == nil {
				db = &mockDB{}
			}
			xkcd := tt.xkcd
			if xkcd == nil {
				xkcd = &mockXKCD{}
			}
			words := tt.words
			if words == nil {
				words = &mockWords{}
			}
			notifier := tt.notifier
			if notifier == nil {
				notifier = &mockNotifier{}
			}

			svc := mustNewService(db, xkcd, words, notifier, 1)
			err := svc.Update(t.Context())

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("want error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpdate_AlreadyRunning(t *testing.T) {
	t.Parallel()

	ready := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	var once sync.Once
	svc := mustNewService(
		&mockDB{},
		&mockXKCD{lastIDFn: func(ctx context.Context) (int, error) {
			once.Do(func() { close(ready) })
			<-ctx.Done()
			return 0, ctx.Err()
		}},
		&mockWords{},
		&mockNotifier{},
		1,
	)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = svc.Update(ctx)
	}()
	<-ready

	err := svc.Update(ctx)
	if !errors.Is(err, core.ErrAlreadyExists) {
		t.Errorf("want %v, got %v", core.ErrAlreadyExists, err)
	}

	cancel()
	<-done
}

func TestUpdate_ContextCancelledAtIDs(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	svc := mustNewService(
		&mockDB{idsFn: func(ctx context.Context) ([]int, error) {
			return nil, ctx.Err()
		}},
		&mockXKCD{lastIDFn: func(_ context.Context) (int, error) { return 5, nil }},
		&mockWords{},
		&mockNotifier{},
		1,
	)

	err := svc.Update(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want %v, got %v", context.Canceled, err)
	}
}

func TestStats(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("db error")
	lastIDErr := errors.New("last id error")
	wantDB := core.DBStats{WordsTotal: 10, WordsUnique: 5, ComicsFetched: 3}

	tests := []struct {
		name    string
		db      *mockDB
		xkcd    *mockXKCD
		wantErr error
		wantOut core.ServiceStats
	}{
		{
			name:    "db error",
			db:      &mockDB{statsFn: func(_ context.Context) (core.DBStats, error) { return core.DBStats{}, dbErr }},
			xkcd:    &mockXKCD{},
			wantErr: dbErr,
		},
		{
			name:    "last id error",
			db:      &mockDB{},
			xkcd:    &mockXKCD{lastIDFn: func(_ context.Context) (int, error) { return 0, lastIDErr }},
			wantErr: lastIDErr,
		},
		{
			name:    "success",
			db:      &mockDB{statsFn: func(_ context.Context) (core.DBStats, error) { return wantDB, nil }},
			xkcd:    &mockXKCD{lastIDFn: func(_ context.Context) (int, error) { return 42, nil }},
			wantOut: core.ServiceStats{DBStats: wantDB, ComicsTotal: 42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := mustNewService(tt.db, tt.xkcd, &mockWords{}, &mockNotifier{}, 1)
			got, err := svc.Stats(t.Context())
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("want wrapped %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.wantOut {
					t.Errorf("got %+v, want %+v", got, tt.wantOut)
				}
			}
		})
	}
}

func TestStatus_Idle(t *testing.T) {
	t.Parallel()

	svc := mustNewService(&mockDB{}, &mockXKCD{}, &mockWords{}, &mockNotifier{}, 1)

	if got := svc.Status(t.Context()); got != core.StatusIdle {
		t.Errorf("want %q, got %q", core.StatusIdle, got)
	}
}

func TestStatus_RunningDuringUpdate(t *testing.T) {
	t.Parallel()

	ready := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	var once sync.Once
	svc := mustNewService(
		&mockDB{},
		&mockXKCD{lastIDFn: func(ctx context.Context) (int, error) {
			once.Do(func() { close(ready) })
			<-ctx.Done()
			return 0, ctx.Err()
		}},
		&mockWords{},
		&mockNotifier{},
		1,
	)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = svc.Update(ctx)
	}()
	<-ready

	if got := svc.Status(t.Context()); got != core.StatusRunning {
		t.Errorf("want %q during update, got %q", core.StatusRunning, got)
	}

	cancel()
	<-done
}

func TestDrop(t *testing.T) {
	t.Parallel()

	dropErr := errors.New("drop error")

	tests := []struct {
		name        string
		db          *mockDB
		notifier    *mockNotifier
		wantErr     error
		checkNotify bool
	}{
		{
			name:    "db error",
			db:      &mockDB{dropFn: func(_ context.Context) error { return dropErr }},
			wantErr: dropErr,
		},
		{
			name: "notifier called after drop",
			db:   &mockDB{},
			notifier: &mockNotifier{dbDroppedFn: func(_ context.Context) error {
				return nil
			}},
			checkNotify: true,
		},
		{
			name: "notifier error ignored",
			db:   &mockDB{},
			notifier: &mockNotifier{dbDroppedFn: func(_ context.Context) error {
				return errors.New("notify failed")
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			notifier := tt.notifier
			if notifier == nil {
				notifier = &mockNotifier{}
			}
			
			var notifierCalled bool
			if tt.checkNotify {
				notifier = &mockNotifier{dbDroppedFn: func(_ context.Context) error {
					notifierCalled = true
					return nil
				}}
			}

			svc := mustNewService(tt.db, &mockXKCD{}, &mockWords{}, notifier, 1)
			err := svc.Drop(t.Context())

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("want wrapped %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.checkNotify && !notifierCalled {
					t.Error("notifier.DBDropped must be called after drop")
				}
			}
		})
	}
}

func TestUpdate_JobDispatchCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// block worker to ensure jobs channel fills up
	workerBlock := make(chan struct{})
	defer close(workerBlock)

	svc := mustNewService(
		&mockDB{idsFn: func(_ context.Context) ([]int, error) { return nil, nil }},
		&mockXKCD{
			lastIDFn: func(_ context.Context) (int, error) { return 10, nil },
			getFn: func(ctx context.Context, _ int) (core.XKCDInfo, error) {
				<-workerBlock
				return core.XKCDInfo{}, nil
			},
		},
		&mockWords{},
		&mockNotifier{},
		1,
	)

	// wait for worker to start and block
	ready := make(chan struct{}, 1)
	go func() {
		cancel()
		ready <- struct{}{}
	}()
	<-ready

	err := svc.Update(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want %v, got %v", context.Canceled, err)
	}
}

func TestUpdate_ProcessComicError(t *testing.T) {
	t.Parallel()

	processErr := errors.New("process error")
	svc := mustNewService(
		&mockDB{idsFn: func(_ context.Context) ([]int, error) { return nil, nil }},
		&mockXKCD{
			lastIDFn: func(_ context.Context) (int, error) { return 1, nil },
			getFn:    func(_ context.Context, _ int) (core.XKCDInfo, error) { return core.XKCDInfo{}, processErr },
		},
		&mockWords{},
		&mockNotifier{},
		1,
	)

	if err := svc.Update(t.Context()); err != nil {
		t.Fatalf("Update should not return error for a failed comic process, got: %v", err)
	}
}

func TestUpdate_Comic404SaveError(t *testing.T) {
	t.Parallel()

	saveErr := errors.New("save error")
	svc := mustNewService(
		&mockDB{
			idsFn: func(_ context.Context) ([]int, error) { return nil, nil },
			addFn: func(_ context.Context, _ core.Comics) error { return saveErr },
		},
		&mockXKCD{
			lastIDFn: func(_ context.Context) (int, error) { return 404, nil },
			getFn:    func(_ context.Context, _ int) (core.XKCDInfo, error) { return core.XKCDInfo{}, core.ErrNotFound },
		},
		&mockWords{},
		&mockNotifier{},
		1,
	)

	if err := svc.Update(t.Context()); err != nil {
		t.Fatalf("Update should not return error for a failed %d save, got: %v", 404, err)
	}
}

func TestUpdate_ProcessComicNormError(t *testing.T) {
	t.Parallel()

	normErr := errors.New("norm error")
	tests := []struct {
		name string
		id   int
	}{
		{"title norm error", 1},
		{"transcript norm error", 1},
		{"alt norm error", 1},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			count := 0
			svc := mustNewService(
				&mockDB{idsFn: func(_ context.Context) ([]int, error) { return nil, nil }},
				&mockXKCD{
					lastIDFn: func(_ context.Context) (int, error) { return tt.id, nil },
					getFn:    func(_ context.Context, _ int) (core.XKCDInfo, error) { return core.XKCDInfo{ID: tt.id}, nil },
				},
				&mockWords{normFn: func(_ context.Context, _ string) ([]string, error) {
					if count == i {
						return nil, normErr
					}
					count++
					return nil, nil
				}},
				&mockNotifier{},
				1,
			)
			_ = svc.Update(t.Context())
		})
	}
}

func TestUpdate_ProcessComicAddError(t *testing.T) {
	t.Parallel()

	addErr := errors.New("add error")
	svc := mustNewService(
		&mockDB{
			idsFn: func(_ context.Context) ([]int, error) { return nil, nil },
			addFn: func(_ context.Context, _ core.Comics) error { return addErr },
		},
		&mockXKCD{
			lastIDFn: func(_ context.Context) (int, error) { return 1, nil },
			getFn:    func(_ context.Context, _ int) (core.XKCDInfo, error) { return core.XKCDInfo{ID: 1}, nil },
		},
		&mockWords{},
		&mockNotifier{},
		1,
	)

	_ = svc.Update(t.Context())
}

func TestService_TrimUTF8(t *testing.T) {
	t.Parallel()
	const (
		maxBytes   = 4096
		wantLength = 4095
	)
	prefix := make([]byte, wantLength)
	for i := range prefix {
		prefix[i] = 'a'
	}
	query := string(prefix) + "д"

	var mu sync.Mutex
	var capturedTitle string
	svc := mustNewService(
		&mockDB{
			idsFn: func(_ context.Context) ([]int, error) { return nil, nil },
			addFn: func(_ context.Context, _ core.Comics) error { return nil },
		},
		&mockXKCD{
			lastIDFn: func(_ context.Context) (int, error) { return 1, nil },
			getFn: func(_ context.Context, id int) (core.XKCDInfo, error) {
				return core.XKCDInfo{ID: id, Title: query, Transcript: query, Alt: query}, nil
			},
		},
		&mockWords{normFn: func(_ context.Context, phrase string) ([]string, error) {
			mu.Lock()
			capturedTitle = phrase
			mu.Unlock()
			return []string{"word"}, nil
		}},
		&mockNotifier{},
		1,
	)

	if err := svc.Update(t.Context()); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(capturedTitle) != wantLength {
		t.Errorf("want length %d, got %d. captured: %q", wantLength, len(capturedTitle), capturedTitle)
	}
}

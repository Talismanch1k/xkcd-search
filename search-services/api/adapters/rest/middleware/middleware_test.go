package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"yadro.com/course/api/adapters/rest/middleware"
)

type mockVerifier struct {
	err    error
	called bool
	token  string
}

func (m *mockVerifier) Verify(token string) error {
	m.called = true
	m.token = token
	return m.err
}

func nextOK(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

func newGet(t *testing.T, target string) *http.Request {
	return httptest.NewRequest(http.MethodGet, target, nil).WithContext(t.Context())
}

func TestAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		header           string
		verifyErr        error
		wantStatus       int
		wantNextCalled   bool
		wantVerifyCalled bool
		wantToken        string
	}{
		{
			name:       "no Authorization header",
			header:     "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Bearer prefix is rejected",
			header:     "Bearer sometoken",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Basic prefix is rejected",
			header:     "Basic dXNlcjpwYXNz",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:             "Token prefix but verifier fails",
			header:           "Token badtoken",
			verifyErr:        errors.New("invalid"),
			wantStatus:       http.StatusUnauthorized,
			wantVerifyCalled: true,
			wantToken:        "badtoken",
		},
		{
			name:             "valid Token header",
			header:           "Token goodtoken",
			wantStatus:       http.StatusOK,
			wantNextCalled:   true,
			wantVerifyCalled: true,
			wantToken:        "goodtoken",
		},
		{
			name:             "Token with empty value, verifier passes",
			header:           "Token ",
			wantStatus:       http.StatusOK,
			wantNextCalled:   true,
			wantVerifyCalled: true,
			wantToken:        "",
		},
		{
			name:             "Token with empty value, verifier fails",
			header:           "Token ",
			verifyErr:        errors.New("empty token"),
			wantStatus:       http.StatusUnauthorized,
			wantVerifyCalled: true,
			wantToken:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var nextCalled bool
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			verifier := &mockVerifier{err: tt.verifyErr}
			h := middleware.Auth(next, verifier)

			w := httptest.NewRecorder()
			r := newGet(t, "/")
			if tt.header != "" {
				r.Header.Set("Authorization", tt.header)
			}

			h(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", w.Code, tt.wantStatus)
			}
			if nextCalled != tt.wantNextCalled {
				t.Errorf("next called: got %v, want %v", nextCalled, tt.wantNextCalled)
			}
			if verifier.called != tt.wantVerifyCalled {
				t.Errorf("verifier called: got %v, want %v", verifier.called, tt.wantVerifyCalled)
			}
			if tt.wantVerifyCalled && verifier.token != tt.wantToken {
				t.Errorf("token passed to verifier: got %q, want %q", verifier.token, tt.wantToken)
			}
		})
	}
}

func TestConcurrency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "single request passes through",
			fn: func(t *testing.T) {
				h := middleware.Concurrency(http.HandlerFunc(nextOK), 1)
				w := httptest.NewRecorder()
				h(w, newGet(t, "/"))

				if w.Code != http.StatusOK {
					t.Errorf("got %d, want %d", w.Code, http.StatusOK)
				}
			},
		},
		{
			name: "sequential requests all pass",
			fn: func(t *testing.T) {
				h := middleware.Concurrency(http.HandlerFunc(nextOK), 1)
				for i := range 3 {
					w := httptest.NewRecorder()
					h(w, newGet(t, "/"))
					if w.Code != http.StatusOK {
						t.Errorf("request %d: got %d, want %d", i, w.Code, http.StatusOK)
					}
				}
			},
		},
		{
			name: "second concurrent request rejected when limit=1",
			fn: func(t *testing.T) {
				blocked := make(chan struct{})
				proceed := make(chan struct{})

				h := middleware.Concurrency(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					close(blocked)
					<-proceed
					w.WriteHeader(http.StatusOK)
				}), 1)

				firstDone := make(chan struct{})
				go func() {
					defer close(firstDone)
					h(httptest.NewRecorder(), newGet(t, "/"))
				}()

				<-blocked

				w := httptest.NewRecorder()
				h(w, newGet(t, "/"))

				if w.Code != http.StatusServiceUnavailable {
					t.Errorf("got %d, want %d", w.Code, http.StatusServiceUnavailable)
				}

				close(proceed)
				<-firstDone
			},
		},
		{
			name: "slot freed after request completes",
			fn: func(t *testing.T) {
				h := middleware.Concurrency(http.HandlerFunc(nextOK), 1)

				h(httptest.NewRecorder(), newGet(t, "/"))

				w := httptest.NewRecorder()
				h(w, newGet(t, "/"))

				if w.Code != http.StatusOK {
					t.Errorf("got %d, want %d after slot freed", w.Code, http.StatusOK)
				}
			},
		},
		{
			name: "limit=2 allows two concurrent requests",
			fn: func(t *testing.T) {
				var (
					mu    sync.Mutex
					count int
				)

				bothInside := make(chan struct{})
				proceed := make(chan struct{})

				h := middleware.Concurrency(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					count++
					inside := count
					mu.Unlock()

					if inside == 2 {
						close(bothInside)
					}
					<-proceed
				}), 2)

				var wg sync.WaitGroup
				for range 2 {
					wg.Go(func() { h(httptest.NewRecorder(), newGet(t, "/")) })
				}

				<-bothInside

				close(proceed)
				wg.Wait()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.fn(t)
		})
	}
}

func TestRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		ctx        func(t *testing.T) context.Context
		rps        int
		wantStatus int
	}{
		{
			name:       "request within limit passes",
			ctx:        func(t *testing.T) context.Context { return t.Context() },
			rps:        1000,
			wantStatus: http.StatusOK,
		},
		{
			name:       "cancelled context returns 429",
			ctx:        cancelledCtx,
			rps:        1,
			wantStatus: http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := middleware.Rate(http.HandlerFunc(nextOK), tt.rps)

			w := httptest.NewRecorder()
			r := newGet(t, "/").WithContext(tt.ctx(t))
			h(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func cancelledCtx(t *testing.T) context.Context {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

func TestWithMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
	}{
		{
			name:       "default 200 when WriteHeader not called",
			handler:    func(w http.ResponseWriter, r *http.Request) {},
			wantStatus: http.StatusOK,
		},
		{
			name:       "captures 404",
			handler:    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) },
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "captures 500",
			handler:    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "captures 201",
			handler:    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) },
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var nextCalled bool
			wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				tt.handler(w, r)
			})

			h := middleware.WithMetrics(wrapped)

			w := httptest.NewRecorder()
			h.ServeHTTP(w, newGet(t, "/test"))

			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d", w.Code, tt.wantStatus)
			}
			if !nextCalled {
				t.Error("want next handler to be called")
			}
		})
	}
}

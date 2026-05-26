package rest_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"yadro.com/course/api/adapters/rest"
	"yadro.com/course/api/core"
)

type mockUpdater struct {
	stats  core.UpdateStats
	status core.UpdateStatus
	err    error
}

func (m *mockUpdater) Update(ctx context.Context) error                      { return m.err }
func (m *mockUpdater) Stats(ctx context.Context) (core.UpdateStats, error)   { return m.stats, m.err }
func (m *mockUpdater) Status(ctx context.Context) (core.UpdateStatus, error) { return m.status, m.err }
func (m *mockUpdater) Drop(ctx context.Context) error                        { return m.err }

func TestNewUpdateHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mockErr    error
		wantStatus int
	}{
		{
			name:       "success",
			mockErr:    nil,
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "already exists (update already running)",
			mockErr:    core.ErrAlreadyExists,
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "internal service error",
			mockErr:    errors.New("redis connection lost"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			updater := &mockUpdater{err: tt.mockErr}
			handler := rest.NewUpdateHandler(updater)

			req := httptest.NewRequest(http.MethodPost, "/update", nil).WithContext(t.Context())
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestNewStatsHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mockStats  core.UpdateStats
		mockErr    error
		wantStatus int
	}{
		{
			name: "success",
			mockStats: core.UpdateStats{
				WordsTotal:    10,
				WordsUnique:   5,
				ComicsFetched: 2,
				ComicsTotal:   100,
			},
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "error",
			mockStats:  core.UpdateStats{},
			mockErr:    errors.New("failed"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			updater := &mockUpdater{stats: tt.mockStats, err: tt.mockErr}
			handler := rest.NewStatsHandler(updater)

			req := httptest.NewRequest(http.MethodGet, "/stats", nil).WithContext(t.Context())
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestNewStatusHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mockStatus core.UpdateStatus
		mockErr    error
		wantStatus int
	}{
		{
			name:       "success idle",
			mockStatus: core.StatusUpdateIdle,
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "error",
			mockStatus: core.StatusUpdateUnknown,
			mockErr:    errors.New("failed"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			updater := &mockUpdater{status: tt.mockStatus, err: tt.mockErr}
			handler := rest.NewStatusHandler(updater)

			req := httptest.NewRequest(http.MethodGet, "/status", nil).WithContext(t.Context())
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

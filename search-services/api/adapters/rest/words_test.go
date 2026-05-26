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

type mockNormalizer struct {
	words []string
	err   error
}

func (m *mockNormalizer) Norm(ctx context.Context, phrase string) ([]string, error) {
	return m.words, m.err
}

func TestNewNormHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		queryURL   string
		mockWords  []string
		mockErr    error
		wantStatus int
	}{
		{
			name:       "success",
			queryURL:   "/norm?phrase=hello",
			mockWords:  []string{"hello"},
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing phrase",
			queryURL:   "/norm",
			mockWords:  nil,
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad arguments error",
			queryURL:   "/norm?phrase=xyz",
			mockWords:  nil,
			mockErr:    core.ErrBadArguments,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "internal error",
			queryURL:   "/norm?phrase=hello",
			mockWords:  nil,
			mockErr:    errors.New("db error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			norm := &mockNormalizer{words: tt.mockWords, err: tt.mockErr}
			handler := rest.NewNormHandler(norm)

			req := httptest.NewRequest(http.MethodGet, tt.queryURL, nil).WithContext(t.Context())
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

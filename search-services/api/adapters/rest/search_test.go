package rest_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"yadro.com/course/api/adapters/rest"
	"yadro.com/course/api/core"
)

type mockSearcher struct {
	res core.SearchResult
	err error
}

func (m *mockSearcher) Search(ctx context.Context, query string, limit int32) (core.SearchResult, error) {
	return m.res, m.err
}

func (m *mockSearcher) ISearch(ctx context.Context, query string, limit int32) (core.SearchResult, error) {
	return m.res, m.err
}

func (m *mockSearcher) GetComic(ctx context.Context, id int) (core.ComicInfo, error) {
	return core.ComicInfo{}, m.err
}

func (m *mockSearcher) Drop(ctx context.Context) error {
	return m.err
}

func TestNewSearchHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		queryURL   string
		mockRes    core.SearchResult
		mockErr    error
		wantStatus int
	}{
		{
			name:     "success with valid phrase",
			queryURL: "/search?phrase=apple",
			mockRes: core.SearchResult{
				Comics: []core.Comic{{ID: 1, URL: "http://example.com/1"}},
				Total:  1,
			},
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name:     "success with valid limit",
			queryURL: "/search?phrase=apple&limit=5",
			mockRes: core.SearchResult{
				Comics: []core.Comic{{ID: 1, URL: "http://example.com/1"}},
				Total:  1,
			},
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing phrase",
			queryURL:   "/search",
			mockRes:    core.SearchResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty phrase",
			queryURL:   "/search?phrase=",
			mockRes:    core.SearchResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid limit (not a number)",
			queryURL:   "/search?phrase=apple&limit=abc",
			mockRes:    core.SearchResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid limit (negative)",
			queryURL:   "/search?phrase=apple&limit=-5",
			mockRes:    core.SearchResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "internal service error",
			queryURL:   "/search?phrase=apple",
			mockRes:    core.SearchResult{},
			mockErr:    errors.New("database connection lost"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			searcher := &mockSearcher{
				res: tt.mockRes,
				err: tt.mockErr,
			}
			handler := rest.NewSearchHandler(searcher)

			req := httptest.NewRequest(http.MethodGet, tt.queryURL, nil).WithContext(t.Context())
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rr.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("failed to parse response JSON: %v", err)
				}
				if _, ok := response["comics"]; !ok {
					t.Error("expected 'comics' field in response")
				}
				if _, ok := response["total"]; !ok {
					t.Error("expected 'total' field in response")
				}
			}
		})
	}
}

func TestNewISearchHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		queryURL   string
		mockRes    core.SearchResult
		mockErr    error
		wantStatus int
	}{
		{
			name:     "success",
			queryURL: "/isearch?phrase=apple",
			mockRes: core.SearchResult{
				Comics: []core.Comic{{ID: 1, URL: "http://example.com/1"}},
				Total:  1,
			},
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name:     "success with valid limit",
			queryURL: "/isearch?phrase=apple&limit=5",
			mockRes: core.SearchResult{
				Comics: []core.Comic{{ID: 1, URL: "http://example.com/1"}},
				Total:  1,
			},
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing phrase",
			queryURL:   "/isearch",
			mockRes:    core.SearchResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "internal service error",
			queryURL:   "/isearch?phrase=apple",
			mockRes:    core.SearchResult{},
			mockErr:    errors.New("service error"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid limit (not a number)",
			queryURL:   "/isearch?phrase=apple&limit=abc",
			mockRes:    core.SearchResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid limit (negative)",
			queryURL:   "/isearch?phrase=apple&limit=-5",
			mockRes:    core.SearchResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			searcher := &mockSearcher{
				res: tt.mockRes,
				err: tt.mockErr,
			}
			handler := rest.NewISearchHandler(searcher)

			req := httptest.NewRequest(http.MethodGet, tt.queryURL, nil).WithContext(t.Context())
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

package rest_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"yadro.com/course/api/adapters/rest"
	"yadro.com/course/api/core"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error { return m.err }

func TestNewPingHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pingers    map[string]core.Pinger
		wantStatus int
	}{
		{
			name: "all ok",
			pingers: map[string]core.Pinger{
				"db":    &mockPinger{err: nil},
				"cache": &mockPinger{err: nil},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "one unavailable",
			pingers: map[string]core.Pinger{
				"db":    &mockPinger{err: errors.New("timeout")},
				"cache": &mockPinger{err: nil},
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := rest.NewPingHandler(tt.pingers, time.Second)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil).WithContext(t.Context())
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rr.Code, tt.wantStatus)
			}

			var resp map[string]map[string]string
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode json: %v", err)
			}
			if len(resp["replies"]) != len(tt.pingers) {
				t.Errorf("got %d replies, want %d", len(resp["replies"]), len(tt.pingers))
			}
		})
	}
}

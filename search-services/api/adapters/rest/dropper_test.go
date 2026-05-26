package rest_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"yadro.com/course/api/adapters/rest"
)

type mockDropper struct {
	err error
}

func (m *mockDropper) Drop(ctx context.Context) error { return m.err }

func TestNewDropHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		dropperErr error
		wantStatus int
	}{
		{
			name:       "success",
			dropperErr: nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "error",
			dropperErr: errors.New("drop failed"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &mockDropper{err: tt.dropperErr}
			handler := rest.NewDropHandler(d)

			req := httptest.NewRequest(http.MethodDelete, "/drop", nil).WithContext(t.Context())
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

package rest_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"yadro.com/course/api/adapters/rest"
)

type mockAuthenticator struct {
	token string
	err   error
}

func (m *mockAuthenticator) Login(name, password string) (string, error) {
	return m.token, m.err
}

func TestNewLoginHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		mockToken  string
		mockErr    error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"name":"admin", "password":"password"}`,
			mockToken:  "token123",
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "bad request json",
			body:       `{"name":}`,
			mockToken:  "",
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthorized",
			body:       `{"name":"admin", "password":"wrong"}`,
			mockToken:  "",
			mockErr:    errors.New("invalid credentials"),
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			auth := &mockAuthenticator{token: tt.mockToken, err: tt.mockErr}
			handler := rest.NewLoginHandler(auth)

			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(tt.body)).WithContext(t.Context())
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

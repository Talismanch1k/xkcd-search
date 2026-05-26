package rest_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"yadro.com/course/api/adapters/rest"
)

func TestNewMetricsHandler(t *testing.T) {
	t.Parallel()

	handler := rest.NewMetricsHandler()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil).WithContext(t.Context())
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusOK)
	}
}

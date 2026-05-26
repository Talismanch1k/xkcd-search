package httputil_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"yadro.com/course/api/internal/httputil"
)

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code int
		data any
		want string
	}{
		{
			name: "write simple map",
			code: http.StatusOK,
			data: map[string]string{"foo": "bar"},
			want: `{"foo":"bar"}`,
		},
		{
			name: "write struct",
			code: http.StatusCreated,
			data: struct {
				ID int `json:"id"`
			}{ID: 42},
			want: `{"id":42}`,
		},
		{
			name: "failed encoding",
			code: http.StatusInternalServerError,
			data: func() {},
			want: `{"error":"failed to encode json response"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			httputil.WriteJSON(w, tt.code, tt.data)

			if w.Code != tt.code {
				t.Errorf("got status %d, want %d", w.Code, tt.code)
			}

			if got := w.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Errorf("got content-type %q, want %q", got, "application/json; charset=utf-8")
			}

			if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("got x-content-type-options %q, want %q", got, "nosniff")
			}

			var got any
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			var want any
			if err := json.Unmarshal([]byte(tt.want), &want); err != nil {
				t.Fatalf("failed to unmarshal want string: %v", err)
			}

			gotBytes, _ := json.Marshal(got)
			wantBytes, _ := json.Marshal(want)

			if string(gotBytes) != string(wantBytes) {
				t.Errorf("got body %s, want %s", string(gotBytes), string(wantBytes))
			}
		})
	}
}

func TestWriteJSONOk(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	data := map[string]int{"status": 1}
	httputil.WriteJSONOk(w, data)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}

	var got map[string]int
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if got["status"] != 1 {
		t.Errorf("got status %d, want %d", got["status"], 1)
	}
}

func TestWriteJSONError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  string
		code int
	}{
		{
			name: "bad request",
			msg:  "invalid input",
			code: http.StatusBadRequest,
		},
		{
			name: "internal error",
			msg:  "something went wrong",
			code: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			httputil.WriteJSONError(w, tt.msg, tt.code)

			if w.Code != tt.code {
				t.Errorf("got status %d, want %d", w.Code, tt.code)
			}

			var got map[string]string
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("failed to decode body: %v", err)
			}

			if got["error"] != tt.msg {
				t.Errorf("got message %q, want %q", got["error"], tt.msg)
			}
		})
	}
}

type errorWriter struct{}

func (e *errorWriter) Header() http.Header         { return http.Header{} }
func (e *errorWriter) WriteHeader(statusCode int) {}
func (e *errorWriter) Write(p []byte) (int, error) {
	return 0, errors.New("simulated write error")
}

func TestWriteJSON_WriteError(t *testing.T) {
	t.Parallel()
	httputil.WriteJSON(&errorWriter{}, http.StatusOK, map[string]string{"ok": "true"})
}

func TestWriteJSON_EncodingAndWriteError(t *testing.T) {
	t.Parallel()
	httputil.WriteJSON(&errorWriter{}, http.StatusOK, func() {})
}

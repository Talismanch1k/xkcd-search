package xkcd_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"yadro.com/course/update/adapters/xkcd"
	"yadro.com/course/update/core"
)

func TestNewClient(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "valid URL",
			url:     "http://example.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, err := xkcd.NewClient(tt.url, time.Second)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("got nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c == nil {
				t.Fatal("got nil client")
			}
		})
	}
}

func TestClient_Get(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/info.0.json" {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"num": 1, "safe_title": "title", "transcript": "transcript", "alt": "alt", "img": "img"}`)); err != nil {
				t.Errorf("w.Write failed: %v", err)
			}
			return
		}
		if r.URL.Path == "/404/info.0.json" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path == "/429/info.0.json" {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if r.URL.Path == "/500/info.0.json" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)

	client, err := xkcd.NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	tests := []struct {
		name     string
		comicID  int
		wantInfo core.XKCDInfo
		wantErr  error
	}{
		{
			name:    "success",
			comicID: 1,
			wantInfo: core.XKCDInfo{
				ID:         1,
				URL:        "img",
				Title:      "title",
				Alt:        "alt",
				Transcript: "transcript",
			},
			wantErr: nil,
		},
		{
			name:     "not found",
			comicID:  404,
			wantInfo: core.XKCDInfo{},
			wantErr:  core.ErrNotFound,
		},
		{
			name:     "rate limit",
			comicID:  429,
			wantInfo: core.XKCDInfo{},
			wantErr:  core.ErrRateLimit,
		},
		{
			name:     "internal error",
			comicID:  500,
			wantInfo: core.XKCDInfo{},
			wantErr:  errors.New("unexpected status"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			info, err := client.Get(t.Context(), tt.comicID)

			if tt.wantErr != nil {
				if tt.comicID == 500 {
					if err == nil {
						t.Fatal("got nil, want error")
					}
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("got %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if info != tt.wantInfo {
				t.Errorf("got %+v, want %+v", info, tt.wantInfo)
			}
		})
	}
}

func TestClient_LastID(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/info.0.json" {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"num": 100}`)); err != nil {
				t.Errorf("w.Write failed: %v", err)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client, err := xkcd.NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	id, err := client.LastID(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantID := 100
	if id != wantID {
		t.Errorf("got %d, want %d", id, wantID)
	}
}

func TestClient_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		setup   func(t *testing.T) (*xkcd.Client, int, bool)
		wantErr bool
	}{
		{
			name: "new request error",
			setup: func(t *testing.T) (*xkcd.Client, int, bool) {
				c, _ := xkcd.NewClient(":", time.Second)
				return c, 1, false
			},
			wantErr: true,
		},
		{
			name: "execute request error",
			setup: func(t *testing.T) (*xkcd.Client, int, bool) {
				c, _ := xkcd.NewClient("http://localhost:1", time.Second)
				return c, 1, false
			},
			wantErr: true,
		},
		{
			name: "decode error",
			setup: func(t *testing.T) (*xkcd.Client, int, bool) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					if _, err := w.Write([]byte(`{invalid json`)); err != nil {
						t.Errorf("w.Write failed: %v", err)
					}
				}))
				t.Cleanup(server.Close)
				c, _ := xkcd.NewClient(server.URL, time.Second)
				return c, 1, false
			},
			wantErr: true,
		},
		{
			name: "last id fetch error",
			setup: func(t *testing.T) (*xkcd.Client, int, bool) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
				t.Cleanup(server.Close)
				c, _ := xkcd.NewClient(server.URL, time.Second)
				return c, 0, true
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, id, isLastID := tt.setup(t)
			var err error
			if isLastID {
				_, err = client.LastID(t.Context())
			} else {
				_, err = client.Get(t.Context(), id)
			}
			if tt.wantErr && err == nil {
				t.Fatal("got nil, want error")
			}
		})
	}
}

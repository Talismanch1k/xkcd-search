package search

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	searchpb "yadro.com/course/proto/search"
)

type mockSearchClient struct {
	pingErr    error
	searchRes  *searchpb.SearchResponse
	searchErr  error
	isearchRes *searchpb.SearchResponse
	isearchErr error
	dropErr    error
}

func (m *mockSearchClient) Ping(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, m.pingErr
}

func (m *mockSearchClient) Search(ctx context.Context, in *searchpb.SearchRequest, opts ...grpc.CallOption) (*searchpb.SearchResponse, error) {
	return m.searchRes, m.searchErr
}

func (m *mockSearchClient) ISearch(ctx context.Context, in *searchpb.SearchRequest, opts ...grpc.CallOption) (*searchpb.SearchResponse, error) {
	return m.isearchRes, m.isearchErr
}

func (m *mockSearchClient) GetComic(ctx context.Context, in *searchpb.ComicRequest, opts ...grpc.CallOption) (*searchpb.ComicInfo, error) {
	return nil, nil
}

func (m *mockSearchClient) Drop(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, m.dropErr
}

func TestClient_Ping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockErr error
		wantErr bool
	}{
		{"success", nil, false},
		{"error", errors.New("error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := &Client{client: &mockSearchClient{pingErr: tt.mockErr}}
			err := client.Ping(t.Context())
			if (err != nil) != tt.wantErr {
				t.Errorf("Ping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Search(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		mockRes   *searchpb.SearchResponse
		mockErr   error
		wantErr   bool
		wantTotal int
		wantLen   int
	}{
		{
			name: "success",
			mockRes: &searchpb.SearchResponse{
				Comics: []*searchpb.Comic{{Id: 1, Url: "url"}},
				Total:  1,
			},
			wantErr:   false,
			wantTotal: 1,
			wantLen:   1,
		},
		{
			name:    "error",
			mockErr: errors.New("error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := &Client{
				client: &mockSearchClient{
					searchRes: tt.mockRes,
					searchErr: tt.mockErr,
				},
			}
			res, err := client.Search(t.Context(), "test", 5)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.Total != tt.wantTotal || len(res.Comics) != tt.wantLen {
					t.Errorf("unexpected result: %+v", res)
				}
			}
		})
	}
}

func TestClient_ISearch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		mockRes   *searchpb.SearchResponse
		mockErr   error
		wantErr   bool
		wantTotal int
		wantLen   int
	}{
		{
			name: "success",
			mockRes: &searchpb.SearchResponse{
				Comics: []*searchpb.Comic{{Id: 1, Url: "url"}},
				Total:  1,
			},
			wantErr:   false,
			wantTotal: 1,
			wantLen:   1,
		},
		{
			name:    "error",
			mockErr: errors.New("error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := &Client{
				client: &mockSearchClient{
					isearchRes: tt.mockRes,
					isearchErr: tt.mockErr,
				},
			}
			res, err := client.ISearch(t.Context(), "test", 5)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.Total != tt.wantTotal || len(res.Comics) != tt.wantLen {
					t.Errorf("unexpected result: %+v", res)
				}
			}
		})
	}
}

func TestClient_Drop(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockErr error
		wantErr bool
	}{
		{"success", nil, false},
		{"error", errors.New("error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := &Client{client: &mockSearchClient{dropErr: tt.mockErr}}
			err := client.Drop(t.Context())
			if (err != nil) != tt.wantErr {
				t.Errorf("Drop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewClient_And_Close(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{
			name:    "success",
			address: "127.0.0.1:1234",
			wantErr: false,
		},
		{
			name:    "error",
			address: "passthrough://\x00",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewClient(tt.address)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if client == nil {
				t.Fatal("expected client, got nil")
			}
			if client.conn == nil {
				t.Fatal("expected conn, got nil")
			}

			if err := client.Close(); err != nil {
				t.Errorf("unexpected error on Close: %v", err)
			}
		})
	}
}

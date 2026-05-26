package grpc_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/adapters/grpc"
	"yadro.com/course/search/core"
)

type mockSearchService struct {
	searchRes    []core.Comic
	searchErr    error
	isearchRes   []core.Comic
	isearchErr   error
	getComicRes  core.Comic
	getComicErr  error
	dropped      bool
}

func (m *mockSearchService) Search(ctx context.Context, query string, limit int32) ([]core.Comic, error) {
	return m.searchRes, m.searchErr
}

func (m *mockSearchService) ISearch(ctx context.Context, query string, limit int32) ([]core.Comic, error) {
	return m.isearchRes, m.isearchErr
}

func (m *mockSearchService) GetComic(_ context.Context, id int) (core.Comic, error) {
	return m.getComicRes, m.getComicErr
}

func (m *mockSearchService) DropIndex() {
	m.dropped = true
}

func TestServer_Ping(t *testing.T) {
	t.Parallel()
	srv := grpc.NewServer(&mockSearchService{})
	_, err := srv.Ping(t.Context(), &emptypb.Empty{})
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestServer_Search(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		req     *searchpb.SearchRequest
		mockRes []core.Comic
		mockErr error
		wantErr codes.Code
	}{
		{
			name: "success",
			req:  &searchpb.SearchRequest{Query: "test", Limit: 5},
			mockRes: []core.Comic{
				{ID: 1, URL: "url1"},
			},
			wantErr: codes.OK,
		},
		{
			name:    "invalid limit",
			req:     &searchpb.SearchRequest{Query: "test", Limit: 0},
			wantErr: codes.InvalidArgument,
		},
		{
			name:    "internal error",
			req:     &searchpb.SearchRequest{Query: "test", Limit: 5},
			mockErr: errors.New("db error"),
			wantErr: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := grpc.NewServer(&mockSearchService{
				searchRes: tt.mockRes,
				searchErr: tt.mockErr,
			})

			res, err := srv.Search(t.Context(), tt.req)

			if tt.wantErr == codes.OK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(res.Comics) != len(tt.mockRes) {
					t.Errorf("got %d comics, want %d", len(res.Comics), len(tt.mockRes))
				}
			} else {
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got %v", err)
				}
				if st.Code() != tt.wantErr {
					t.Errorf("got code %v, want %v", st.Code(), tt.wantErr)
				}
			}
		})
	}
}

func TestServer_ISearch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		req     *searchpb.SearchRequest
		mockRes []core.Comic
		mockErr error
		wantErr codes.Code
	}{
		{
			name: "success",
			req:  &searchpb.SearchRequest{Query: "test", Limit: 5},
			mockRes: []core.Comic{
				{ID: 1, URL: "url1"},
			},
			wantErr: codes.OK,
		},
		{
			name:    "invalid limit",
			req:     &searchpb.SearchRequest{Query: "test", Limit: 0},
			wantErr: codes.InvalidArgument,
		},
		{
			name:    "internal error",
			req:     &searchpb.SearchRequest{Query: "test", Limit: 5},
			mockErr: errors.New("db error"),
			wantErr: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := grpc.NewServer(&mockSearchService{
				isearchRes: tt.mockRes,
				isearchErr: tt.mockErr,
			})

			res, err := srv.ISearch(t.Context(), tt.req)

			if tt.wantErr == codes.OK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(res.Comics) != len(tt.mockRes) {
					t.Errorf("got %d comics, want %d", len(res.Comics), len(tt.mockRes))
				}
			} else {
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got %v", err)
				}
				if st.Code() != tt.wantErr {
					t.Errorf("got code %v, want %v", st.Code(), tt.wantErr)
				}
			}
		})
	}
}

func TestServer_Drop(t *testing.T) {
	t.Parallel()
	srv := grpc.NewServer(&mockSearchService{})
	_, err := srv.Drop(t.Context(), &emptypb.Empty{})
	if err != nil {
		t.Fatalf("Drop failed: %v", err)
	}
}

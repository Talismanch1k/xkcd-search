package grpc_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	updatepb "yadro.com/course/proto/update"
	"yadro.com/course/update/adapters/grpc"
	"yadro.com/course/update/core"
)

type mockUpdater struct {
	status core.ServiceStatus
	stats  core.ServiceStats
	err    error
}

func (m *mockUpdater) Status(ctx context.Context) core.ServiceStatus {
	return m.status
}

func (m *mockUpdater) Update(ctx context.Context) error {
	return m.err
}

func (m *mockUpdater) Stats(ctx context.Context) (core.ServiceStats, error) {
	return m.stats, m.err
}

func (m *mockUpdater) Drop(ctx context.Context) error {
	return m.err
}

func TestServer_Ping(t *testing.T) {
	t.Parallel()
	srv := grpc.NewServer(&mockUpdater{})
	_, err := srv.Ping(t.Context(), &emptypb.Empty{})
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestServer_Status(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockStatus core.ServiceStatus
		wantStatus updatepb.Status
	}{
		{
			name:       "idle",
			mockStatus: core.StatusIdle,
			wantStatus: updatepb.Status_STATUS_IDLE,
		},
		{
			name:       "running",
			mockStatus: core.StatusRunning,
			wantStatus: updatepb.Status_STATUS_RUNNING,
		},
		{
			name:       "unknown",
			mockStatus: core.ServiceStatus("unknown"),
			wantStatus: updatepb.Status_STATUS_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := grpc.NewServer(&mockUpdater{status: tt.mockStatus})
			res, err := srv.Status(t.Context(), &emptypb.Empty{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Status != tt.wantStatus {
				t.Errorf("got %v, want %v", res.Status, tt.wantStatus)
			}
		})
	}
}

func TestServer_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockStatus core.ServiceStatus
		wantErr    codes.Code
	}{
		{
			name:       "success",
			mockStatus: core.StatusIdle,
			wantErr:    codes.OK,
		},
		{
			name:       "already exists",
			mockStatus: core.StatusRunning,
			wantErr:    codes.AlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := grpc.NewServer(&mockUpdater{status: tt.mockStatus})
			_, err := srv.Update(t.Context(), &emptypb.Empty{})
			if tt.wantErr == codes.OK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
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

func TestServer_Stats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		mockStats core.ServiceStats
		mockErr   error
		wantErr   codes.Code
	}{
		{
			name: "success",
			mockStats: core.ServiceStats{
				DBStats: core.DBStats{
					WordsTotal:    1,
					WordsUnique:   2,
					ComicsFetched: 3,
				},
				ComicsTotal: 4,
			},
			mockErr: nil,
			wantErr: codes.OK,
		},
		{
			name:    "error",
			mockErr: errors.New("db error"),
			wantErr: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := grpc.NewServer(&mockUpdater{stats: tt.mockStats, err: tt.mockErr})
			res, err := srv.Stats(t.Context(), &emptypb.Empty{})
			if tt.wantErr == codes.OK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.WordsTotal != int64(tt.mockStats.WordsTotal) {
					t.Errorf("got %d, want %d", res.WordsTotal, tt.mockStats.WordsTotal)
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
	tests := []struct {
		name    string
		mockErr error
		wantErr codes.Code
	}{
		{
			name:    "success",
			mockErr: nil,
			wantErr: codes.OK,
		},
		{
			name:    "error",
			mockErr: errors.New("db error"),
			wantErr: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := grpc.NewServer(&mockUpdater{err: tt.mockErr})
			_, err := srv.Drop(t.Context(), &emptypb.Empty{})
			if tt.wantErr == codes.OK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
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

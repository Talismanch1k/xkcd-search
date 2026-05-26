package update

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/core"
	updatepb "yadro.com/course/proto/update"
)

type mockUpdateClient struct {
	pingErr   error
	statusRes *updatepb.StatusReply
	statusErr error
	statsRes  *updatepb.StatsReply
	statsErr  error
	updateErr error
	dropErr   error
}

func (m *mockUpdateClient) Ping(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, m.pingErr
}

func (m *mockUpdateClient) Status(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*updatepb.StatusReply, error) {
	return m.statusRes, m.statusErr
}

func (m *mockUpdateClient) Stats(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*updatepb.StatsReply, error) {
	return m.statsRes, m.statsErr
}

func (m *mockUpdateClient) Update(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, m.updateErr
}

func (m *mockUpdateClient) Drop(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error) {
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
			client := &Client{client: &mockUpdateClient{pingErr: tt.mockErr}}
			err := client.Ping(t.Context())
			if (err != nil) != tt.wantErr {
				t.Errorf("Ping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Status(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockRes *updatepb.StatusReply
		mockErr error
		wantRes core.UpdateStatus
		wantErr bool
	}{
		{
			name:    "idle",
			mockRes: &updatepb.StatusReply{Status: updatepb.Status_STATUS_IDLE},
			wantRes: core.StatusUpdateIdle,
		},
		{
			name:    "running",
			mockRes: &updatepb.StatusReply{Status: updatepb.Status_STATUS_RUNNING},
			wantRes: core.StatusUpdateRunning,
		},
		{
			name:    "unknown",
			mockRes: &updatepb.StatusReply{Status: updatepb.Status_STATUS_UNSPECIFIED},
			wantRes: core.StatusUpdateUnknown,
		},
		{
			name:    "error",
			mockErr: errors.New("error"),
			wantRes: core.StatusUpdateUnknown,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := &Client{client: &mockUpdateClient{statusRes: tt.mockRes, statusErr: tt.mockErr}}
			res, err := client.Status(t.Context())
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if res != tt.wantRes {
					t.Errorf("got %v, want %v", res, tt.wantRes)
				}
			}
		})
	}
}

func TestClient_Stats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockRes *updatepb.StatsReply
		mockErr error
		wantErr bool
	}{
		{
			name: "success",
			mockRes: &updatepb.StatsReply{
				WordsTotal:    1,
				WordsUnique:   2,
				ComicsFetched: 3,
				ComicsTotal:   4,
			},
			wantErr: false,
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
			client := &Client{client: &mockUpdateClient{statsRes: tt.mockRes, statsErr: tt.mockErr}}
			res, err := client.Stats(t.Context())
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.WordsTotal != 1 || res.WordsUnique != 2 || res.ComicsFetched != 3 || res.ComicsTotal != 4 {
					t.Errorf("unexpected result: %+v", res)
				}
			}
		})
	}
}

func TestClient_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		mockErr   error
		wantErr   bool
		errTarget error
	}{
		{"success", nil, false, nil},
		{"already exists", status.Error(codes.AlreadyExists, "exists"), true, core.ErrAlreadyExists},
		{"error", errors.New("error"), true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := &Client{client: &mockUpdateClient{updateErr: tt.mockErr}}
			err := client.Update(t.Context())
			if tt.wantErr {
				if tt.errTarget != nil {
					if err != tt.errTarget {
						t.Errorf("expected %v, got %v", tt.errTarget, err)
					}
				} else if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
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
			client := &Client{client: &mockUpdateClient{dropErr: tt.mockErr}}
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

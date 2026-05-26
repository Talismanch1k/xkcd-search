package words

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	wordspb "yadro.com/course/proto/words"
	"yadro.com/course/update/core"
)

type mockWordsClient struct {
	normRes *wordspb.WordsReply
	normErr error
	pingErr error
}

func (m *mockWordsClient) Ping(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, m.pingErr
}

func (m *mockWordsClient) Norm(ctx context.Context, in *wordspb.WordsRequest, opts ...grpc.CallOption) (*wordspb.WordsReply, error) {
	return m.normRes, m.normErr
}

func TestClient_Norm(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockRes *wordspb.WordsReply
		mockErr error
		wantErr error
		wantLen int
	}{
		{
			name:    "success",
			mockRes: &wordspb.WordsReply{Words: []string{"test"}},
			wantErr: nil,
			wantLen: 1,
		},
		{
			name:    "resource exhausted",
			mockErr: status.Error(codes.ResourceExhausted, "too large"),
			wantErr: core.ErrBadArguments,
		},
		{
			name:    "internal error",
			mockErr: errors.New("internal"),
			wantErr: errors.New("internal"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := &Client{
				client: &mockWordsClient{
					normRes: tt.mockRes,
					normErr: tt.mockErr,
				},
			}
			res, err := client.Norm(t.Context(), "test phrase")
			switch {
			case tt.wantErr == core.ErrBadArguments:
				if err != core.ErrBadArguments {
					t.Errorf("got %v, want %v", err, core.ErrBadArguments)
				}
			case tt.wantErr != nil:
				if err == nil {
					t.Errorf("got nil, want error %v", tt.wantErr)
				}
			default:
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(res) != tt.wantLen {
					t.Errorf("got %d words, want %d", len(res), tt.wantLen)
				}
			}
		})
	}
}

func TestClient_Ping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pingErr error
		wantErr bool
	}{
		{
			name:    "success",
			pingErr: nil,
			wantErr: false,
		},
		{
			name:    "error",
			pingErr: errors.New("error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := &Client{
				client: &mockWordsClient{pingErr: tt.pingErr},
			}
			err := client.Ping(t.Context())
			if tt.wantErr {
				if err == nil {
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

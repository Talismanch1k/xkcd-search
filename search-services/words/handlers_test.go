package main

import (
	"context"
	"slices"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	wordspb "yadro.com/course/proto/words"
)

type mockStemmer struct{ words []string }

func (m *mockStemmer) Normalize(_ string) []string { return m.words }

func TestPing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{
			name: "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &server{}
			resp, err := s.Ping(t.Context(), &emptypb.Empty{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp == nil {
				t.Error("expected non-nil responce")
			}
		})
	}
}

func TestNorm(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		phrase    string
		ctx       func(*testing.T) context.Context
		stemWords []string
		wantWords []string
		wantCode  codes.Code
	}{
		{
			name:      "normal phrase",
			phrase:    "hello world",
			ctx:       func(t *testing.T) context.Context { return t.Context() },
			stemWords: []string{"hello", "world"},
			wantWords: []string{"hello", "world"},
			wantCode:  codes.OK,
		},
		{
			name:     "too large message",
			phrase:   strings.Repeat("a", maxMessageSize+1),
			ctx:      func(t *testing.T) context.Context { return t.Context() },
			wantCode: codes.ResourceExhausted,
		},
		{
			name:     "cancelled context",
			phrase:   "hello world",
			ctx:      cancelledCtx,
			wantCode: codes.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &server{stemmer: &mockStemmer{words: tt.stemWords}}
			resp, err := s.Norm(tt.ctx(t), &wordspb.WordsRequest{Phrase: tt.phrase})

			if tt.wantCode == codes.OK {

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if !slices.Equal(resp.Words, tt.wantWords) {
					t.Errorf("got words %v, want %v", resp.Words, tt.wantWords)
				}

			} else {

				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got %v", err)
				}

				if st.Code() != tt.wantCode {
					t.Errorf("got code %v, want %v", st.Code(), tt.wantCode)
				}

			}
		})
	}
}

func cancelledCtx(t *testing.T) context.Context {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

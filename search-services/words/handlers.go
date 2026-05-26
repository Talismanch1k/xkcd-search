package main

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	wordspb "yadro.com/course/proto/words"
)

const maxMessageSize = 4 * 1024 // bytes

type Stemmer interface {
	Normalize(str string) []string
}

type server struct {
	wordspb.UnimplementedWordsServer
	stemmer Stemmer
}

func (s *server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *server) Norm(ctx context.Context, r *wordspb.WordsRequest) (*wordspb.WordsReply, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(ctx.Err()).Err()
	}
	if len(r.GetPhrase()) > maxMessageSize {
		return nil, status.Error(codes.ResourceExhausted, "message exceeds 4 KiB limit")
	}

	resp := wordspb.WordsReply{Words: s.stemmer.Normalize(r.GetPhrase())}

	return &resp, nil
}

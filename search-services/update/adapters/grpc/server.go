package grpc

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	updatepb "yadro.com/course/proto/update"
	"yadro.com/course/update/core"
)

func NewServer(service core.Updater) *Server {
	return &Server{service: service}
}

type Server struct {
	updatepb.UnimplementedUpdateServer
	service core.Updater
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *Server) Status(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatusReply, error) {
	st := s.service.Status(ctx)

	switch st {
	case core.StatusIdle:
		return &updatepb.StatusReply{Status: updatepb.Status_STATUS_IDLE}, nil

	case core.StatusRunning:
		return &updatepb.StatusReply{Status: updatepb.Status_STATUS_RUNNING}, nil

	default:
		return &updatepb.StatusReply{Status: updatepb.Status_STATUS_UNSPECIFIED}, nil
	}
}

func (s *Server) Update(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if s.service.Status(ctx) == core.StatusRunning {
		return nil, status.Error(codes.AlreadyExists, core.ErrAlreadyExists.Error())
	}

	go func() {
		bgCtx := context.WithoutCancel(ctx)
		if err := s.service.Update(bgCtx); err != nil {
			if !errors.Is(err, core.ErrAlreadyExists) {
				slog.Error("manual update", "error", err)
			}
		}
	}()

	return &emptypb.Empty{}, nil
}

func (s *Server) Stats(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatsReply, error) {
	st, err := s.service.Stats(ctx)
	if err != nil {
		slog.Error("get stats", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &updatepb.StatsReply{
		WordsTotal:    int64(st.WordsTotal),
		WordsUnique:   int64(st.WordsUnique),
		ComicsFetched: int64(st.ComicsFetched),
		ComicsTotal:   int64(st.ComicsTotal),
	}, nil
}

func (s *Server) Drop(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Drop(ctx); err != nil {
		slog.Error("drop database", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

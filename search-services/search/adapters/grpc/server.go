package grpc

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/core"
)

type Server struct {
	searchpb.UnimplementedSearchServiceServer
	service core.SearchService
}

func NewServer(s core.SearchService) *Server {
	return &Server{service: s}
}

func (s *Server) Ping(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *Server) Search(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
	limit := req.GetLimit()
	if limit < 1 {
		return nil, status.Error(codes.InvalidArgument, "limit must be positive")
	}

	comics, err := s.service.Search(ctx, req.GetQuery(), limit)
	if err != nil {
		slog.Error("during search", "err", err, "query", req.GetQuery())
		return nil, status.Errorf(codes.Internal, "internal search error: %v", err)
	}

	pbComics := make([]*searchpb.Comic, 0, len(comics))
	for _, c := range comics {
		pbComics = append(pbComics,
			&searchpb.Comic{
				Id:  int32(c.ID),
				Url: c.URL,
			},
		)
	}

	return &searchpb.SearchResponse{
		Comics: pbComics,
		Total:  int32(len(comics)),
	}, nil
}

func (s *Server) ISearch(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
	limit := req.GetLimit()
	if limit < 1 {
		return nil, status.Error(codes.InvalidArgument, "limit must be positive")
	}

	comics, err := s.service.ISearch(ctx, req.GetQuery(), limit)
	if err != nil {
		slog.Error("during isearch", "err", err, "query", req.GetQuery())
		return nil, status.Errorf(codes.Internal, "internal search error: %v", err)
	}

	pbComics := make([]*searchpb.Comic, 0, len(comics))
	for _, c := range comics {
		pbComics = append(pbComics,
			&searchpb.Comic{
				Id:  int32(c.ID),
				Url: c.URL,
			},
		)
	}

	return &searchpb.SearchResponse{
		Comics: pbComics,
		Total:  int32(len(comics)),
	}, nil
}

func (s *Server) GetComic(ctx context.Context, req *searchpb.ComicRequest) (*searchpb.ComicInfo, error) {
	comic, err := s.service.GetComic(ctx, int(req.GetId()))
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "comic %d not found", req.GetId())
		}
		slog.Error("get comic", "id", req.GetId(), "err", err)
		return nil, status.Errorf(codes.Internal, "get comic: %v", err)
	}
	return &searchpb.ComicInfo{
		Id:          int32(comic.ID),
		Url:         comic.URL,
		Titles:      comic.Title,
		Alts:        comic.Alt,
		Transcripts: comic.Transcript,
	}, nil
}

func (s *Server) Drop(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	s.service.DropIndex()
	return &emptypb.Empty{}, nil
}

package search

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/core"
	searchpb "yadro.com/course/proto/search"
)

type Client struct {
	client searchpb.SearchServiceClient
	conn   *grpc.ClientConn
}

func NewClient(address string) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connect to search grpc: %w", err)
	}
	return &Client{
		client: searchpb.NewSearchServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	return err
}

func (c *Client) Search(ctx context.Context, query string, limit int32) (core.SearchResult, error) {
	resp, err := c.client.Search(ctx, &searchpb.SearchRequest{
		Query: query,
		Limit: int32(limit),
	})
	if err != nil {
		return core.SearchResult{}, fmt.Errorf("search grpc: %w", err)
	}

	comics := make([]core.Comic, 0, len(resp.GetComics()))
	for _, c := range resp.GetComics() {
		comics = append(comics, core.Comic{
			ID:  int(c.GetId()),
			URL: c.GetUrl(),
		})
	}

	return core.SearchResult{
		Comics: comics,
		Total:  int(resp.GetTotal()),
	}, nil
}

// somewhere DRY crying...

func (c *Client) ISearch(ctx context.Context, query string, limit int32) (core.SearchResult, error) {
	resp, err := c.client.ISearch(ctx, &searchpb.SearchRequest{
		Query: query,
		Limit: int32(limit),
	})
	if err != nil {
		return core.SearchResult{}, fmt.Errorf("search grpc: %w", err)
	}

	comics := make([]core.Comic, 0, len(resp.GetComics()))
	for _, c := range resp.GetComics() {
		comics = append(comics, core.Comic{
			ID:  int(c.GetId()),
			URL: c.GetUrl(),
		})
	}

	return core.SearchResult{
		Comics: comics,
		Total:  int(resp.GetTotal()),
	}, nil
}

func (c *Client) GetComic(ctx context.Context, id int) (core.ComicInfo, error) {
	resp, err := c.client.GetComic(ctx, &searchpb.ComicRequest{Id: int32(id)})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return core.ComicInfo{}, core.ErrNotFound
		}
		return core.ComicInfo{}, fmt.Errorf("get comic grpc: %w", err)
	}
	return core.ComicInfo{
		ID:         int(resp.GetId()),
		URL:        resp.GetUrl(),
		Title:      resp.GetTitles(),
		Alt:        resp.GetAlts(),
		Transcript: resp.GetTranscripts(),
	}, nil
}

func (c *Client) Drop(ctx context.Context) error {
	_, err := c.client.Drop(ctx, &emptypb.Empty{})
	return err
}

func (c *Client) Close() error {
	return c.conn.Close()
}

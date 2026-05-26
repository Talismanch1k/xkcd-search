package update

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/core"
	updatepb "yadro.com/course/proto/update"
)

type Client struct {
	client updatepb.UpdateClient
	conn   *grpc.ClientConn
}

func NewClient(address string) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connect to update: %w", err)
	}
	return &Client{
		client: updatepb.NewUpdateClient(conn),
		conn:   conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	return err
}

func (c *Client) Status(ctx context.Context) (core.UpdateStatus, error) {
	resp, err := c.client.Status(ctx, &emptypb.Empty{})
	if err != nil {
		return core.StatusUpdateUnknown, fmt.Errorf("get status: %w", err)
	}

	switch resp.GetStatus() {
	case updatepb.Status_STATUS_IDLE:
		return core.StatusUpdateIdle, nil

	case updatepb.Status_STATUS_RUNNING:
		return core.StatusUpdateRunning, nil

	default:
		return core.StatusUpdateUnknown, nil
	}
}

func (c *Client) Stats(ctx context.Context) (core.UpdateStats, error) {
	resp, err := c.client.Stats(ctx, &emptypb.Empty{})
	if err != nil {
		return core.UpdateStats{}, fmt.Errorf("get stats: %w", err)
	}

	return core.UpdateStats{
		WordsTotal:    int(resp.GetWordsTotal()),
		WordsUnique:   int(resp.GetWordsUnique()),
		ComicsFetched: int(resp.GetComicsFetched()),
		ComicsTotal:   int(resp.GetComicsTotal()),
	}, nil
}

func (c *Client) Update(ctx context.Context) error {
	_, err := c.client.Update(ctx, &emptypb.Empty{})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return core.ErrAlreadyExists
		}
		return fmt.Errorf("update: %w", err)
	}

	return nil
}

func (c *Client) Drop(ctx context.Context) error {
	_, err := c.client.Drop(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("drop: %w", err)
	}

	return nil
}

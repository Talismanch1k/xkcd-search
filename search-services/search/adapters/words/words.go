package words

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	wordspb "yadro.com/course/proto/words"
	"yadro.com/course/search/core"
)

type Client struct {
	client wordspb.WordsClient
	conn   *grpc.ClientConn
}

func NewClient(address string) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connect to words %s: %w", address, err)
	}

	return &Client{
		client: wordspb.NewWordsClient(conn),
		conn:   conn,
	}, nil
}

func (c *Client) Norm(ctx context.Context, phrase string) ([]string, error) {
	norm, err := c.client.Norm(ctx, &wordspb.WordsRequest{Phrase: phrase})
	if err != nil {
		if status.Code(err) == codes.ResourceExhausted {
			return nil, core.ErrBadArguments
		}
		return nil, err
	}
	return norm.GetWords(), nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	return err
}

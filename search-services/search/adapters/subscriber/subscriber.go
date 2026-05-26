package subscriber

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"yadro.com/course/search/core"
)

const (
	SubjectDBUpdated = "xkcd.db.updated"
	SubjectDBDropped = "xkcd.db.dropped"
)

type Client struct {
	nc      *nats.Conn
	indexer core.Indexer
}

func New(addr string, indexer core.Indexer) (*Client, error) {
	nc, err := nats.Connect(addr,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			slog.Warn("nats disconnected", "err", err)
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			slog.Info("nats reconnected", "url", c.ConnectedUrl())
		}),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			slog.Error("nats async error", "err", err)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("connect nats: %w", err)
	}

	return &Client{nc: nc, indexer: indexer}, nil
}

func (c *Client) Start(ctx context.Context) error {
	_, err := c.nc.Subscribe(SubjectDBUpdated, func(msg *nats.Msg) {
		slog.Info("event received", "subject", msg.Subject)

		if ctx.Err() != nil {
			return
		}

		if err := c.indexer.BuildIndex(ctx); err != nil {
			slog.Error("build index on event", "subject", msg.Subject, "err", err)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", SubjectDBUpdated, err)
	}

	_, err = c.nc.Subscribe(SubjectDBDropped, func(msg *nats.Msg) {
		slog.Info("event received", "subject", msg.Subject)

		if ctx.Err() != nil {
			return
		}

		c.indexer.DropIndex()
	})
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", SubjectDBDropped, err)
	}

	slog.Info("subscribed to broker events")

	return nil
}

func (c *Client) Close() error {
	return c.nc.Drain()
}

package publisher

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	SubjectDBUpdated = "xkcd.db.updated"
	SubjectDBDropped = "xkcd.db.dropped"
	retryAttempts    = 5
	retryTimeout     = 5 * time.Second
)

type Client struct {
	nc *nats.Conn
}

func New(addr string) (*Client, error) {
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

	return &Client{nc: nc}, nil
}

func (c *Client) DBUpdated(ctx context.Context) error {
	return retry(ctx, retryAttempts, func(ctx context.Context) error {
		if err := c.nc.Publish(SubjectDBUpdated, nil); err != nil {
			return fmt.Errorf("publish %s: %w", SubjectDBUpdated, err)
		}

		return c.nc.FlushWithContext(ctx)
	})
}

func (c *Client) DBDropped(ctx context.Context) error {
	return retry(ctx, retryAttempts, func(ctx context.Context) error {
		if err := c.nc.Publish(SubjectDBDropped, nil); err != nil {
			return fmt.Errorf("publish %s: %w", SubjectDBDropped, err)
		}

		return c.nc.FlushWithContext(ctx)
	})
}

func (c *Client) Close() error {
	return c.nc.Drain()
}

func retry(ctx context.Context, attempts uint, f func(ctx context.Context) error) error {
	var err error

	for i := range attempts {

		attemptCtx, cancel := context.WithTimeout(ctx, retryTimeout)
		err = f(attemptCtx)
		cancel()

		if err == nil {
			return nil
		}

		base := time.Duration(100<<i) * time.Millisecond
		jitter := time.Duration(rand.Int64N(int64(base) / 2))
		hold := base + jitter

		select {
		case <-time.After(hold):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return err
}

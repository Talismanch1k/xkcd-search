package interceptors

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func Logging() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		slog.DebugContext(ctx, "grpc request", "method", info.FullMethod)

		resp, err := handler(ctx, req)

		attrs := []any{
			"method", info.FullMethod,
			"code", status.Code(err).String(),
			"duration_ms", time.Since(start).Milliseconds(),
		}

		if err != nil {
			attrs = append(attrs, "err", err)
			slog.ErrorContext(ctx, "grpc response", attrs...)
		} else {
			slog.DebugContext(ctx, "grpc response", attrs...)
		}

		return resp, err
	}
}

package utils

import (
	"context"
	"io"
	"log/slog"
	"time"
)

func CloseOnContext(ctx context.Context, closer io.Closer) {
	<-ctx.Done()
	err := closer.Close()
	if err != nil {
		slog.Error("failed to close io.Closer", slog.Any("error", err))
	}
}

type Closer interface {
	Close(context.Context) error
}

func CloseOnContextWithContext(ctx context.Context, closer Closer, timeout time.Duration) {
	<-ctx.Done()
	ctx2, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := closer.Close(ctx2)
	if err != nil {
		slog.Error("failed to close io.Closer", slog.Any("error", err))
	}
}

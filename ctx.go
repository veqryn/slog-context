package slogcontext

import (
	"context"
	"log/slog"

	"github.com/go-logr/logr"
)

// ToCtx returns a copy of ctx with the logger attached.
// The parent context will be unaffected.
func ToCtx(parent context.Context, logger *slog.Logger) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	return logr.NewContextWithSlogLogger(parent, logger)
}

// Logger returns the slog.Logger associated with the ctx.
// If no logger is associated, or the logger or ctx are nil,
// slog.Default() is returned.
func Logger(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}

	l := logr.FromContextAsSlogLogger(ctx)
	if l == nil {
		return slog.Default()
	}

	return l
}

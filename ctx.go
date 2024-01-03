package slogctx

import (
	"context"
	"log/slog"

	"github.com/go-logr/logr"
)

// NewCtx returns a copy of ctx with the logger attached.
// The parent context will be unaffected.
func NewCtx(parent context.Context, logger *slog.Logger) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	return logr.NewContextWithSlogLogger(parent, logger)
}

// FromCtx returns the slog.FromCtx associated with the ctx.
// If no logger is associated, or the logger or ctx are nil,
// slog.Default() is returned.
// This function will convert a logr.Logger to a *slog.Logger only if necessary.
func FromCtx(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}

	l := logr.FromContextAsSlogLogger(ctx)
	if l == nil {
		return slog.Default()
	}

	return l
}

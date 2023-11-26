package slogcontext

import (
	"context"
	"log/slog"

	"github.com/veqryn/slog-context/internal"
)

// ToCtx returns a copy of ctx with the logger attached.
// The parent context will be unaffected.
// Passing in a nil logger will force future calls of Logger(ctx) on the
// returned context to return the slog.Default() logger.
func ToCtx(parent context.Context, logger *slog.Logger) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, internal.CtxKey{}, internal.LoggerCtxVal{Logger: logger})
}

// Logger returns the slog.Logger associated with the ctx.
// If no logger is associated, or the logger or ctx are nil,
// slog.Default() is returned.
func Logger(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	if l, ok := ctx.Value(internal.CtxKey{}).(internal.LoggerCtxVal); ok && l.Logger != nil {
		return l.Logger
	}
	return slog.Default()
}

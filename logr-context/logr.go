package logrcontext

import (
	"context"
	"log/slog"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/slogr"
	"github.com/veqryn/slog-context/internal"
)

func LogrToCtx(parent context.Context, logger logr.Logger) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, internal.CtxKey{}, internal.LoggerCtxVal{
		Logger: slog.New(slogr.NewSlogHandler(logger)),
		IterOp: logger,
	})
}

func SlogToCtx(parent context.Context, logger *slog.Logger) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	if logger == nil {
		return context.WithValue(parent, internal.CtxKey{}, internal.LoggerCtxVal{Logger: logger})
	}
	return context.WithValue(parent, internal.CtxKey{}, internal.LoggerCtxVal{
		Logger: logger,
		IterOp: slogr.NewLogr(logger.Handler()),
	})
}

func LogrFromCtx(ctx context.Context) logr.Logger {
	if ctx == nil {
		return slogr.NewLogr(slog.Default().Handler())
	}
	if l, ok := ctx.Value(internal.CtxKey{}).(internal.LoggerCtxVal); ok {
		if lgr, hasLogr := l.IterOp.(logr.Logger); hasLogr {
			return lgr
		}
		if l.Logger != nil {
			return slogr.NewLogr(l.Logger.Handler())
		}
	}
	return slogr.NewLogr(slog.Default().Handler())
}

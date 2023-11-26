package internal

import "log/slog"

// CtxKey Logger key for context.valueCtx
type CtxKey struct{}

// LoggerCtxVal is just a bucket containing a *slog.Logger
// and potentially other logger interfaces for interoperability.
type LoggerCtxVal struct {
	Logger *slog.Logger
	IterOp any
}

package slogctx

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/veqryn/slog-context/internal/attr"
)

// Add key for context.valueCtx
type addKey struct{}

// Add adds the attribute arguments at the root level, and not in any groups.
func Add(parent context.Context, args ...any) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	if v, ok := parent.Value(addKey{}).([]slog.Attr); ok {
		// Clip to ensure this is a scoped copy
		return context.WithValue(parent, addKey{}, append(slices.Clip(v), attr.ArgsToAttrSlice(args)...))
	}
	return context.WithValue(parent, addKey{}, attr.ArgsToAttrSlice(args))
}

// extractAdded returns the added attributes stored in the context.
// The returned slice should not be appended to or modified in any way. Doing so will cause a race condition.
func extractAdded(ctx context.Context, _ time.Time, _ slog.Level, _ string) []slog.Attr {
	if v, ok := ctx.Value(addKey{}).([]slog.Attr); ok {
		return v
	}
	return nil
}

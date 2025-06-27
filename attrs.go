package slogctx

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/pazams/yasctx/internal/attr"
)

type addKey struct{}
type addToGroupKey struct{}

// Add adds the attribute arguments at the root level
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

// Add adds the attribute arguments at a group level
// If the future log line does not use the group, it will default to the root level.
func AddToGroup(parent context.Context, group string, args ...any) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	if v, ok := parent.Value(addToGroupKey{}).(map[string][]slog.Attr); ok {
		v[group] = append(slices.Clip(v[group]), attr.ArgsToAttrSlice(args)...)
		return context.WithValue(parent, addToGroupKey{}, v)
	}
	return context.WithValue(parent, addToGroupKey{}, map[string][]slog.Attr{
		group: attr.ArgsToAttrSlice(args),
	})
}

// extractAdded returns the added attributes stored in the context.
// The returned slice should not be appended to or modified in any way. Doing so will cause a race condition.
func extractAdded(ctx context.Context, _ time.Time, _ slog.Level, _ string) []slog.Attr {
	if v, ok := ctx.Value(addKey{}).([]slog.Attr); ok {
		return v
	}
	return nil
}

func extractAddedToGroup(ctx context.Context, _ time.Time, _ slog.Level, _ string) map[string][]slog.Attr {
	if v, ok := ctx.Value(addToGroupKey{}).(map[string][]slog.Attr); ok {
		return v
	}
	return nil
}

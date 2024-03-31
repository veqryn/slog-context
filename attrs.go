package slogctx

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/veqryn/slog-context/internal/attr"
)

// Prepend key for context.valueCtx
type prependKey struct{}

// Append key for context.valueCtx
type appendKey struct{}

// Prepend adds the attribute arguments to the end of the group that will be
// prepended to the start of the log record when it is handled.
// This means that these attributes will be at the root level, and not in any
// groups.
func Prepend(parent context.Context, args ...any) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	if v, ok := parent.Value(prependKey{}).([]slog.Attr); ok {
		// Clip to ensure this is a scoped copy
		return context.WithValue(parent, prependKey{}, append(slices.Clip(v), attr.ArgsToAttrSlice(args)...))
	}
	return context.WithValue(parent, prependKey{}, attr.ArgsToAttrSlice(args))
}

// ExtractPrepended is an AttrExtractor that returns the prepended attributes
// stored in the context. The returned slice should not be appended to or
// modified in any way. Doing so will cause a race condition.
func ExtractPrepended(ctx context.Context, _ time.Time, _ slog.Level, _ string) []slog.Attr {
	if v, ok := ctx.Value(prependKey{}).([]slog.Attr); ok {
		return v
	}
	return nil
}

// Append adds the attribute arguments to the end of the group that will be
// appended to the end of the log record when it is handled.
// This means that the attributes could be in a group or sub-group, if the log
// has used WithGroup at some point.
func Append(parent context.Context, args ...any) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	if v, ok := parent.Value(appendKey{}).([]slog.Attr); ok {
		// Clip to ensure this is a scoped copy
		return context.WithValue(parent, appendKey{}, append(slices.Clip(v), attr.ArgsToAttrSlice(args)...))
	}
	return context.WithValue(parent, appendKey{}, attr.ArgsToAttrSlice(args))
}

// ExtractAppended is an AttrExtractor that returns the appended attributes
// stored in the context. The returned slice should not be appended to or
// modified in any way. Doing so will cause a race condition.
func ExtractAppended(ctx context.Context, _ time.Time, _ slog.Level, _ string) []slog.Attr {
	if v, ok := ctx.Value(appendKey{}).([]slog.Attr); ok {
		return v
	}
	return nil
}

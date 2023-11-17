package slogcontext

import (
	"context"
	"log/slog"
	"slices"
	"time"
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
		return context.WithValue(parent, prependKey{}, append(slices.Clip(v), argsToAttrSlice(args)...))
	}
	return context.WithValue(parent, prependKey{}, argsToAttrSlice(args))
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
		return context.WithValue(parent, appendKey{}, append(slices.Clip(v), argsToAttrSlice(args)...))
	}
	return context.WithValue(parent, appendKey{}, argsToAttrSlice(args))
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

// Turn a slice of arguments, some of which pairs of primitives,
// some might be attributes already, into a slice of attributes.
// This is copied from golang sdk.
func argsToAttrSlice(args []any) []slog.Attr {
	var (
		attr  slog.Attr
		attrs []slog.Attr
	)
	for len(args) > 0 {
		attr, args = argsToAttr(args)
		attrs = append(attrs, attr)
	}
	return attrs
}

// This is copied from golang sdk.
const badKey = "!BADKEY"

// argsToAttr turns a prefix of the nonempty args slice into an Attr
// and returns the unconsumed portion of the slice.
// If args[0] is an Attr, it returns it.
// If args[0] is a string, it treats the first two elements as
// a key-value pair.
// Otherwise, it treats args[0] as a value with a missing key.
// This is copied from golang sdk.
func argsToAttr(args []any) (slog.Attr, []any) {
	switch x := args[0].(type) {
	case string:
		if len(args) == 1 {
			return slog.String(badKey, x), nil
		}
		return slog.Any(x, args[1]), args[2:]

	case slog.Attr:
		return x, args[1:]

	default:
		return slog.Any(badKey, x), args[1:]
	}
}

package slogcontext

import (
	"context"
	"log/slog"
	"slices"
)

// key for context.valueCtx
type attrsKey struct{}

// bucket with our different lists of attributes
type attrsValue struct {
	prepended []slog.Attr // list of attributes to add to the start of a log record
	appended  []slog.Attr // list of attributes to add to the end of a log record
}

// prependedAttrs iterates through all prepend attributes
func prependedAttrs(ctx context.Context, f func(slog.Attr) bool) {
	if v, ok := ctx.Value(attrsKey{}).(attrsValue); ok {
		for _, attr := range v.prepended {
			if !f(attr) {
				break
			}
		}
	}
}

// appendedAttrs iterates through all append attributes
func appendedAttrs(ctx context.Context, f func(slog.Attr) bool) {
	if v, ok := ctx.Value(attrsKey{}).(attrsValue); ok {
		for _, attr := range v.appended {
			if !f(attr) {
				break
			}
		}
	}
}

// Prepend adds the attribute arguments to the end of the group that will be
// prepended to the start of the log record when it is handled.
// This means that these attributes will be at the root level, and not in any
// groups.
func Prepend(parent context.Context, args ...any) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	if v, ok := parent.Value(attrsKey{}).(attrsValue); ok {
		// by being a value, instead of pointer, v is already a shallow copy for the new context, to keep it scoped
		v.prepended = append(slices.Clip(v.prepended), argsToAttrSlice(args)...)
		return context.WithValue(parent, attrsKey{}, v)
	}

	return context.WithValue(parent, attrsKey{}, attrsValue{
		prepended: argsToAttrSlice(args),
	})
}

// Append adds the attribute arguments to the end of the group that will be
// appended to the end of the log record when it is handled.
// This means that the attributes could be in a group or sub-group, if the log
// has used WithGroup at some point.
func Append(parent context.Context, args ...any) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	if v, ok := parent.Value(attrsKey{}).(attrsValue); ok {
		// by being a value, instead of pointer, v is already a shallow copy for the new context, to keep it scoped
		v.appended = append(slices.Clip(v.appended), argsToAttrSlice(args)...)
		return context.WithValue(parent, attrsKey{}, v)
	}

	return context.WithValue(parent, attrsKey{}, attrsValue{
		appended: argsToAttrSlice(args),
	})
}

// This is copied from golang sdk.
// Turn a slice of arguments, some of which pairs of primitives,
// some might be attributes already, into a slice of attributes.
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

// This is copied from golang sdk.
// argsToAttr turns a prefix of the nonempty args slice into an Attr
// and returns the unconsumed portion of the slice.
// If args[0] is an Attr, it returns it.
// If args[0] is a string, it treats the first two elements as
// a key-value pair.
// Otherwise, it treats args[0] as a value with a missing key.
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

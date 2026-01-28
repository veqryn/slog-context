package propagate

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/veqryn/slog-context/internal/attr"
)

// Init initializes a context that allows propagating attributes from child context back to parents.
// Essentially, it lets you collect slog attributes that are discovered later in
// the stack (such as authentication and user ID's, derived values, attributes
// only discovered halfway-through the final request handler after several db
// queries, etc), and be able to have them be included in the log lines of other
// middlewares (such as a middleware that logs all requests that come in).
// For a ready-to-use http middleware that implements this feature, see package github.com/veqryn/slog-context/http
func Init(parent context.Context) context.Context {
	if fromCtx(parent) != nil {
		// If we already have a collector in the context, return it
		return parent
	}

	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, ctxKey{}, &syncAttrs{})
}

// With adds the provided attributes to the context and propagates them to parent contextes.
// If propagation wasn't initialized on the context via a Init(), it will initialize at this point,
// followed by adding the attributes to it.
func With(ctx context.Context, args ...any) context.Context {
	m := fromCtx(ctx)
	if m == nil {
		if ctx == nil {
			ctx = context.Background()
		}
		// Initialize if it doesn't exist, and save the attributes to it
		return context.WithValue(ctx, ctxKey{}, &syncAttrs{attrs: attr.ArgsToAttrSlice(args)})
	}

	// Convert args to a slice of slog.Attr
	attrs := attr.ArgsToAttrSlice(args)
	if len(attrs) == 0 {
		return ctx
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Append to the slice
	m.attrs = append(m.attrs, attrs...)
	return ctx
}

// ExtractAttrs is a slogctx Extractor that must be used with a
// slogctx.Handler (via slogctx.HandlerOptions) as Prependers or Appenders.
// It will cause the Handler to add the Attributes added by slogctx.Add() to all
// log lines using that same context.
func ExtractAttrs(ctx context.Context, _ time.Time, _ slog.Level, _ string) []slog.Attr {
	m := fromCtx(ctx)
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Add the attributes, in the order defined
	attrs := make([]slog.Attr, len(m.attrs))
	copy(attrs, m.attrs)
	return attrs
}

// syncAttrs is an append-only synchronized ordered slice
type syncAttrs struct {
	mu    sync.RWMutex
	attrs []slog.Attr
}

// ctxKey is how we find our attribute collector data structure in the context
type ctxKey struct{}

// fromCtx returns the collector data structure if it is found within the
// context, or nil.
func fromCtx(ctx context.Context) *syncAttrs {
	if ctx == nil {
		return nil
	}

	m := ctx.Value(ctxKey{})
	if m == nil {
		return nil
	}

	return m.(*syncAttrs)
}

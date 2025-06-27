package yasctx

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/pazams/yasctx/internal/attr"
)

// InitPropagation initializes a context that allows propegating attributes from child context back to parents.
// Essentially, it lets you collect slog attributes that are discovered later in
// the stack (such as authentication and user ID's, derived values, attributes
// only discovered halfway-through the final request handler after several db
// queries, etc), and be able to have them be included in the log lines of other
// middlewares (such as a middleware that logs all requests that come in).
func InitPropagation(parent context.Context) context.Context {
	if fromCtx(parent) != nil {
		// If we already have a collector in the context, return it
		return parent
	}

	m := &syncOrderedMap{
		kv: map[string]slog.Attr{},
	}
	if parent == nil {
		parent = context.Background()
	}

	return context.WithValue(parent, ctxKey{}, m)
}

// AddWithPropagation adds the provided attributes to the context and propagates them to parent contextes.
// If propagation wasn't initialized on the context via a InitPropagation(), it falls back to performing a normal Add() operation.
func AddWithPropagation(ctx context.Context, args ...any) context.Context {
	// Convert args to a slice of slog.Attr
	attrs := attr.ArgsToAttrSlice(args)
	if len(attrs) == 0 {
		return ctx
	}

	m := fromCtx(ctx)
	if m == nil {
		// Someone is using this package outside of a request.
		// As a feature for utility methods that could be used both in requests
		// and outside of requests, the most useful thing to do is to return the
		// context with the attributes added. That way the attributes will still
		// end up on log lines using this context, which is the goal in both cases.
		return Add(ctx, args...)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// For each attribute, if it is not already in the map, append it to the end
	// of our ordered list. Then add/update it in the map.
	for _, attr := range attrs {
		if _, ok := m.kv[attr.Key]; !ok {
			// Does not yet exist in the append-only ordered list
			m.order = append(m.order, attr.Key)
		}
		m.kv[attr.Key] = attr
	}
	return ctx
}

// extractPropagatedAttrs is a yasctx Extractor that must be used with a
// yasctx.Handler (via yasctx.HandlerOptions) as Prependers or Appenders.
// It will cause the Handler to add the Attributes added by sloghttp.With to all
// log lines using that same context.
func extractPropagatedAttrs(ctx context.Context, _ time.Time, _ slog.Level, _ string) []slog.Attr {
	m := fromCtx(ctx)
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Add the attributes, in the order defined
	var attrs []slog.Attr
	for _, key := range m.order {
		attrs = append(attrs, m.kv[key])
	}
	return attrs
}

// syncOrderedMap is an append-only synchronized ordered map
type syncOrderedMap struct {
	mu    sync.RWMutex
	kv    map[string]slog.Attr
	order []string
}

// ctxKey is how we find our attribute collector data structure in the context
type ctxKey struct{}

// fromCtx returns the collector data structure if it is found within the
// context, or nil.
func fromCtx(ctx context.Context) *syncOrderedMap {
	if ctx == nil {
		return nil
	}

	m := ctx.Value(ctxKey{})
	if m == nil {
		return nil
	}

	return m.(*syncOrderedMap)
}

package sloghttp

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	slogctx "github.com/veqryn/slog-context"
	"github.com/veqryn/slog-context/internal/attr"
)

// AttrCollectorMiddleware is an http middleware that collects slog.Attr from
// any and all later middlewares and the final http request handler, and makes
// them available to all middlewares and the request handler.
// Essentially, it lets you collect slog attributes that are discovered later in
// the stack (such as authentication and user ID's, derived values, attributes
// only discovered halfway-through the final request handler after several db
// queries, etc), and be able to have them be included in the log lines of other
// middlewares (such as a middleware that logs all requests that come in).
//
// Requires the use of slogctx.Handler, as a wrapper or middleware around your
// slog formatter/sink.
//
// Attributes are added by calls to sloghttp.With, and the attributes are then
// stored inside the context. All calls log that include the context will
// automatically have all the attributes included (ex: slogctx.Info, or
// slog.InfoContext).
func AttrCollectorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the context already contains our map, we don't need to create a new one
		if ctx := r.Context(); fromCtx(ctx) == nil {
			r = r.WithContext(newCtx(ctx, newSyncOrderedMap()))
		}
		next.ServeHTTP(w, r)
	})
}

// With adds the provided slog.Attr's to the context. If used with
// sloghttp.AttrCollectorMiddleware it will add them to the context in a way
// that is visible to all intermediate middlewares and functions between the
// collector middleware and the call to With.
func With(ctx context.Context, args ...any) context.Context {
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
		return slogctx.With(ctx, args...)
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

// AttrCollectorExtractor is a slogctx Extractor that must be used with a
// slogctx.Handler (via slogctx.HandlerOptions) as Prependers or Appenders.
// It will cause the Handler to add the Attributes added by sloghttp.With to all
// log lines using that same context.
func AttrCollectorExtractor(ctx context.Context, _ time.Time, _ slog.Level, _ string) []slog.Attr {
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

// newSyncOrderedMap creates a usable initialized *syncOrderedMap
func newSyncOrderedMap() *syncOrderedMap {
	return &syncOrderedMap{
		kv: map[string]slog.Attr{},
	}
}

// ctxKey is how we find our attribute collector data structure in the context
type ctxKey struct{}

// newCtx returns a copy of the parent context with the collector data structure
// attached. The parent context will be unaffected.
func newCtx(parent context.Context, m *syncOrderedMap) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	return context.WithValue(parent, ctxKey{}, m)
}

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

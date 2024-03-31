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

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the context already contains our map, we don't need to create a new one
		if ctx := r.Context(); fromCtx(ctx) == nil {
			r = r.WithContext(newCtx(ctx, newSyncOrderedMap()))
		}
		next.ServeHTTP(w, r)
	})
}

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

func Extractor(ctx context.Context, _ time.Time, _ slog.Level, _ string) []slog.Attr {
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

func newSyncOrderedMap() *syncOrderedMap {
	return &syncOrderedMap{
		kv: map[string]slog.Attr{},
	}
}

type ctxKey struct{}

func newCtx(parent context.Context, m *syncOrderedMap) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	return context.WithValue(parent, ctxKey{}, m)
}

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

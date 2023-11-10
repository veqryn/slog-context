package slogcontext

import (
	"context"
	"log/slog"
	"slices"
)

// HandlerOptions are options for a Handler
type HandlerOptions struct{}

// Handler is a slog.Handler middleware that will ...
type Handler struct {
	next slog.Handler
	goa  *groupOrAttrs
}

var _ slog.Handler = &Handler{} // Assert conformance with interface

// NewHandler creates a Handler slog.Handler middleware that will ...
// If opts is nil, the default options are used.
func NewHandler(next slog.Handler, opts *HandlerOptions) *Handler {
	if opts == nil {
		opts = &HandlerOptions{}
	}

	return &Handler{
		next: next,
	}
}

// Enabled reports whether the next handler handles records at the given level.
// The handler ignores records whose level is lower.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle de-duplicates all attributes and groups, then passes the new set of attributes to the next handler.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	// Collect all attributes from the record (which is the most recent attribute set).
	// These attributes are ordered from oldest to newest, and our collection will be too.
	finalAttrs := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		finalAttrs = append(finalAttrs, a)
		return true
	})

	// Add our 'appended' context attributes to the end
	if v, ok := ctx.Value(attrsKey{}).(attrsValue); ok {
		finalAttrs = append(finalAttrs, v.appended...)
	}

	// Iterate through the goa (group Or Attributes) linked list, which is ordered from newest to oldest
	for g := h.goa; g != nil; g = g.next {
		if g.group != "" {
			// If a group, but all the previous attributes (the newest ones) in it
			finalAttrs = []slog.Attr{{
				Key:   g.group,
				Value: slog.GroupValue(finalAttrs...),
			}}
		} else {
			// Prepend to the front of finalAttrs, thereby making finalAttrs ordered from oldest to newest
			finalAttrs = append(slices.Clip(g.attrs), finalAttrs...)
		}
	}

	// Add our 'prepended' context attributes to the start
	if v, ok := ctx.Value(attrsKey{}).(attrsValue); ok {
		finalAttrs = append(slices.Clip(v.prepended), finalAttrs...)
	}

	// Add all attributes to new record (because old record has all the old attributes as private members)
	newR := &slog.Record{
		Time:    r.Time,
		Level:   r.Level,
		Message: r.Message,
		PC:      r.PC,
	}

	// Add deduplicated attributes back in
	newR.AddAttrs(finalAttrs...)
	return h.next.Handle(ctx, *newR)
}

// WithGroup returns a new AppendHandler that still has h's attributes,
// but any future attributes added will be namespaced.
func (h *Handler) WithGroup(name string) slog.Handler {
	h2 := *h
	h2.goa = h2.goa.WithGroup(name)
	return &h2
}

// WithAttrs returns a new AppendHandler whose attributes consists of h's attributes followed by attrs.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := *h
	h2.goa = h2.goa.WithAttrs(attrs)
	return &h2
}

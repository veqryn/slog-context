package slogctx

import (
	"context"
	"log/slog"
	"slices"
	"time"
)

// AttrExtractor is a function that retrieves or creates slog.Attr's based
// information/values found in the context.Context and the slog.Record's basic
// attributes.
type AttrExtractor func(ctx context.Context, recordT time.Time, recordLvl slog.Level, recordMsg string) []slog.Attr

// HandlerOptions are options for a Handler
type HandlerOptions struct {
	// A list of functions to be called, each of which will return attributes
	// that should be prepended to the start of every log line with this context.
	// If left nil, the default ExtractPrepended function will be used only.
	Prependers []AttrExtractor

	// A list of functions to be called, each of which will return attributes
	// that should be appended to the end of every log line with this context.
	// If left nil, the default ExtractAppended function will be used only.
	Appenders []AttrExtractor
}

// Handler is a slog.Handler middleware that will Prepend and
// Append attributes to log lines. The attributes are extracted out of the log
// record's context by the provided AttrExtractor methods.
// It passes the final record and attributes off to the next handler when finished.
type Handler struct {
	next       slog.Handler
	goa        *groupOrAttrs
	prependers []AttrExtractor
	appenders  []AttrExtractor
}

var _ slog.Handler = &Handler{} // Assert conformance with interface

// NewMiddleware creates a slogctx.Handler slog.Handler middleware
// that conforms to [github.com/samber/slog-multi.Middleware] interface.
// It can be used with slogmulti methods such as Pipe to easily setup a pipeline of slog handlers:
//
//	slog.SetDefault(slog.New(slogmulti.
//		Pipe(slogctx.NewMiddleware(&slogctx.HandlerOptions{})).
//		Pipe(slogdedup.NewOverwriteMiddleware(&slogdedup.OverwriteHandlerOptions{})).
//		Handler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})),
//	))
func NewMiddleware(options *HandlerOptions) func(slog.Handler) slog.Handler {
	return func(next slog.Handler) slog.Handler {
		return NewHandler(
			next,
			options,
		)
	}
}

// NewHandler creates a Handler slog.Handler middleware that will Prepend and
// Append attributes to log lines. The attributes are extracted out of the log
// record's context by the provided AttrExtractor methods.
// It passes the final record and attributes off to the next handler when finished.
// If opts is nil, the default options are used.
func NewHandler(next slog.Handler, opts *HandlerOptions) *Handler {
	if opts == nil {
		opts = &HandlerOptions{}
	}
	if opts.Prependers == nil {
		opts.Prependers = []AttrExtractor{ExtractPrepended}
	}
	if opts.Appenders == nil {
		opts.Appenders = []AttrExtractor{ExtractAppended}
	}

	return &Handler{
		next:       next,
		prependers: slices.Clone(opts.Prependers),
		appenders:  slices.Clone(opts.Appenders),
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
	for _, f := range h.appenders {
		finalAttrs = append(finalAttrs, f(ctx, r.Time, r.Level, r.Message)...)
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

	// Add our 'prepended' context attributes to the start.
	// Go in reverse order, since each is prepending to the front.
	for i := len(h.prependers) - 1; i >= 0; i-- {
		finalAttrs = append(slices.Clip(h.prependers[i](ctx, r.Time, r.Level, r.Message)), finalAttrs...)
	}

	// Add all attributes to new record (because old record has all the old attributes as private members)
	newR := &slog.Record{
		Time:    r.Time,
		Level:   r.Level,
		Message: r.Message,
		PC:      r.PC,
	}

	// Add attributes back in
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

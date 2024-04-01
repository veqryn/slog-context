// Package test provides useful test helpers that can be used in multiple packages.
package test

import (
	"bytes"
	"context"
	"log/slog"
	"time"
)

// DefaultTime is a single point in time to use for log lines
var DefaultTime = time.Date(2023, 9, 29, 13, 0, 59, 0, time.UTC)

// Handler is a slog.Handler that records the records that come its way
type Handler struct {
	Records []slog.Record
	Ctxs    []context.Context
	Source  bool
}

// Enabled returns true
func (h *Handler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= slog.LevelDebug
}

// Handle records a log record
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	r.Time = DefaultTime
	h.Records = append(h.Records, r)
	h.Ctxs = append(h.Ctxs, ctx)
	return nil
}

// WithGroup panics
func (h *Handler) WithGroup(string) slog.Handler {
	panic("shouldn't be called")
}

// WithAttrs panics
func (h *Handler) WithAttrs([]slog.Attr) slog.Handler {
	panic("shouldn't be called")
}

// String formats all log records with slog.TextHandler
func (h *Handler) String() string {
	buf := &bytes.Buffer{}
	formatter := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: h.Source})
	for _, record := range h.Records {
		err := formatter.Handle(context.Background(), record)
		if err != nil {
			panic(err)
		}
	}
	return buf.String()
}

// MarshalJSON formats all log records with slog.JSONHandler
func (h *Handler) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	formatter := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: h.Source})
	for _, record := range h.Records {
		err := formatter.Handle(context.Background(), record)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

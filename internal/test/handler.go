package test

import (
	"bytes"
	"context"
	"log/slog"
	"time"
)

var DefaultTime = time.Date(2023, 9, 29, 13, 0, 59, 0, time.UTC)

type TestHandler struct {
	Ctx    context.Context
	Record slog.Record
	Source bool
}

func (h *TestHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *TestHandler) Handle(ctx context.Context, r slog.Record) error {
	h.Ctx = ctx
	h.Record = r
	h.Record.Time = DefaultTime
	return nil
}

func (h *TestHandler) WithGroup(string) slog.Handler {
	panic("shouldn't be called")
}

func (h *TestHandler) WithAttrs([]slog.Attr) slog.Handler {
	panic("shouldn't be called")
}

func (h *TestHandler) String() string {
	buf := &bytes.Buffer{}
	err := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: h.Source}).Handle(context.Background(), h.Record)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func (h *TestHandler) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: h.Source}).Handle(context.Background(), h.Record)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

package slogotel

import (
	"bytes"
	"context"
	"log/slog"
	"time"
)

var defaultTime = time.Date(2023, 9, 29, 13, 0, 59, 0, time.UTC)

type testHandler struct {
	Ctx    context.Context
	Record slog.Record
}

func (h *testHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *testHandler) Handle(ctx context.Context, r slog.Record) error {
	h.Ctx = ctx
	h.Record = r
	h.Record.Time = defaultTime
	return nil
}

func (h *testHandler) WithGroup(string) slog.Handler {
	panic("shouldn't be called")
}

func (h *testHandler) WithAttrs([]slog.Attr) slog.Handler {
	panic("shouldn't be called")
}

func (h *testHandler) String() string {
	buf := &bytes.Buffer{}
	err := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}).Handle(context.Background(), h.Record)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func (h *testHandler) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}).Handle(context.Background(), h.Record)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

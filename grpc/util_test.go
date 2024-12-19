package sloggrpc

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestReplaceAttrJsonPB(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	l := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		ReplaceAttr: ReplaceAttrJsonPB,
	}))

	p := timestamppb.New(time.Date(2024, 12, 18, 22, 55, 11, 123456789, time.UTC))
	l.Info("hello", "t", p, "z", "foobar")

	if !strings.HasSuffix(strings.TrimSpace(buf.String()), `"level":"INFO","msg":"hello","t":"2024-12-18T22:55:11.123456789Z","z":"foobar"}`) {
		t.Error(buf.String())
	}
}

func TestJsonPB(t *testing.T) {
	t.Parallel()

	p := timestamppb.New(time.Date(2024, 12, 18, 22, 55, 11, 123456789, time.UTC))
	a := JsonPB(p)

	j, ok := a.(json.RawMessage)
	if !ok {
		t.Fatal("Should implement json.RawMessage")
	}

	if string(j) != `"2024-12-18T22:55:11.123456789Z"` {
		t.Error(string(j))
	}
}

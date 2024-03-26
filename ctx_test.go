package slogctx

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
)

func TestCtx(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h := slog.NewJSONHandler(buf, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// fmt.Printf("ReplaceAttr: key:%s valueKind:%s value:%s nilGroups:%t groups:%#+v\n", a.Key, a.Value.Kind().String(), a.Value.String(), groups == nil, groups)
			if groups == nil && a.Key == slog.TimeKey {
				return slog.Time(slog.TimeKey, defaultTime)
			}
			return a
		},
	})

	// Confirm FromCtx retrieves default if nothing stored in ctx yet
	l := slog.New(h)
	slog.SetDefault(l)
	if FromCtx(nil) != l {
		t.Error("Expecting default logger retrieved")
	}
	if FromCtx(context.Background()) != l {
		t.Error("Expecting default logger retrieved")
	}

	ctx := NewCtx(nil, slog.Default())

	ctx = With(ctx, "with1", "arg1", "with1", "arg2")
	ctx = With(ctx, "with2", "arg1", "with2", "arg2")
	With(ctx, "with3", "arg1", "with3", "arg2") // Ensure we aren't overwriting the parent context

	WithGroup(ctx, "group0") // Ensure we aren't overwriting the parent context
	ctx = WithGroup(ctx, "group1")
	WithGroup(ctx, "group2") // Ensure we aren't overwriting the parent context

	ctx = With(ctx, "with4", "arg1", "with4", "arg2")
	ctx = With(ctx, "with5", "arg1", "with5", "arg2")
	With(ctx, "with6", "arg1", "with6", "arg2") // Ensure we aren't overwriting the parent context

	// Test with getting logger back out
	l = FromCtx(ctx)
	l.InfoContext(ctx, "main message", "main1", "arg1", "main1", "arg2")
	expectedInfo := `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"main message","with1":"arg1","with1":"arg2","with2":"arg1","with2":"arg2","group1":{"with4":"arg1","with4":"arg2","with5":"arg1","with5":"arg2","main1":"arg1","main1":"arg2"}}
`
	if buf.String() != expectedInfo {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedInfo, buf.String())
	}

	// Test with wrappers
	buf.Reset()
	Debug(ctx, "main message", "main1", "arg1", "main1", "arg2")
	expectedDebug := `{"time":"2023-09-29T13:00:59Z","level":"DEBUG","msg":"main message","with1":"arg1","with1":"arg2","with2":"arg1","with2":"arg2","group1":{"with4":"arg1","with4":"arg2","with5":"arg1","with5":"arg2","main1":"arg1","main1":"arg2"}}
`
	if buf.String() != expectedDebug {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedDebug, buf.String())
	}

	buf.Reset()
	Info(ctx, "main message", "main1", "arg1", "main1", "arg2")
	if buf.String() != expectedInfo {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedInfo, buf.String())
	}

	buf.Reset()
	Warn(ctx, "main message", "main1", "arg1", "main1", "arg2")
	expectedWarn := `{"time":"2023-09-29T13:00:59Z","level":"WARN","msg":"main message","with1":"arg1","with1":"arg2","with2":"arg1","with2":"arg2","group1":{"with4":"arg1","with4":"arg2","with5":"arg1","with5":"arg2","main1":"arg1","main1":"arg2"}}
`
	if buf.String() != expectedWarn {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedWarn, buf.String())
	}

	buf.Reset()
	Error(ctx, "main message", "main1", "arg1", "main1", "arg2", Err(errors.New("an error")))
	expectedError := `{"time":"2023-09-29T13:00:59Z","level":"ERROR","msg":"main message","with1":"arg1","with1":"arg2","with2":"arg1","with2":"arg2","group1":{"with4":"arg1","with4":"arg2","with5":"arg1","with5":"arg2","main1":"arg1","main1":"arg2","err":"an error"}}
`
	if buf.String() != expectedError {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedError, buf.String())
	}

	buf.Reset()
	Log(ctx, slog.LevelWarn, "main message", "main1", "arg1", "main1", "arg2")
	if buf.String() != expectedWarn {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedWarn, buf.String())
	}

	buf.Reset()
	LogAttrs(ctx, slog.LevelInfo, "main message", slog.String("main1", "arg1"), slog.String("main1", "arg2"))
	if buf.String() != expectedInfo {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedInfo, buf.String())
	}

	// Test with new context/nil
	buf.Reset()
	Log(nil, slog.LevelWarn, "main message", "main1", "arg1", "main1", "arg2")
	expectedWarnNil := `{"time":"2023-09-29T13:00:59Z","level":"WARN","msg":"main message","main1":"arg1","main1":"arg2"}
`
	if buf.String() != expectedWarnNil {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedWarnNil, buf.String())
	}

	buf.Reset()
	LogAttrs(nil, slog.LevelInfo, "main message", slog.String("main1", "arg1"), slog.String("main1", "arg2"))
	expectedInfoNil := `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"main message","main1":"arg1","main1":"arg2"}
`
	if buf.String() != expectedInfoNil {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedInfoNil, buf.String())
	}
}

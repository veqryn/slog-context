package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	slogctx "github.com/veqryn/slog-context"
)

type ctxKey struct{}

func customExtractor(ctx context.Context, _ time.Time, _ slog.Level, _ string) []slog.Attr {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return []slog.Attr{slog.String("my-key", v)}
	}
	return nil
}

// This workflow lets us use slog as normal, while letting us extract any
// custom values we want from any context, and having them added to the start
// or end of the log record.
func main() {
	// Create the *slogctx.Handler middleware
	h := slogctx.NewHandler(
		slog.NewJSONHandler(os.Stdout, nil), // The next handler in the chain
		&slogctx.HandlerOptions{
			// Prependers stays as default (leaving as nil would accomplish the same)
			Prependers: []slogctx.AttrExtractor{
				slogctx.ExtractPrepended,
			},
			// Appenders first appends anything added with slogctx.Append,
			// then appends our custom ctx value
			Appenders: []slogctx.AttrExtractor{
				slogctx.ExtractAppended,
				customExtractor,
			},
		},
	)
	slog.SetDefault(slog.New(h))

	// Add a value to the context
	ctx := context.WithValue(context.Background(), ctxKey{}, "my-value")

	// Use the logger like normal:
	slog.WarnContext(ctx, "main message", "mainKey", "mainValue")
	/*
		{
			"time": "2023-11-17T04:35:30.333732-07:00",
			"level": "WARN",
			"msg": "main message",
			"mainKey": "mainValue",
			"my-key": "my-value"
		}
	*/
}

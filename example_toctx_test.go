package slogcontext_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	slogcontext "github.com/veqryn/slog-context"
)

func TestExampleToCtx(t *testing.T) {
	t.Parallel()

	h := slogcontext.NewHandler(slog.NewJSONHandler(os.Stdout, nil), nil)
	slog.SetDefault(slog.New(h))

	// Store the logger inside the context:
	ctx := slogcontext.ToCtx(context.Background(), slog.Default())

	// Get the logger back out again at any time:
	log := slogcontext.Logger(ctx)
	log.Warn("warning")

	// Add attributes directly to the logger in the context:
	ctx = slogcontext.WithCtx(ctx, "rootKey", "rootValue")

	// Create a group directly on the logger in the context:
	ctx = slogcontext.WithGroup(ctx, "someGroup")

	// Or get the logger from the context, add attributes to it, and return it:
	log = slogcontext.With(ctx, "subKey", "subValue")

	// Store back in the context again
	ctx = slogcontext.ToCtx(ctx, log)

	// Access the logger in the context directly with handy wrappers for Debug/Info/Warn/Error/Log/LogAttrs:
	slogcontext.Info(ctx, "main message", "mainKey", "mainValue")
	/*
		{
			"time":"2023-11-14T00:53:46.363072-07:00",
			"level":"INFO",
			"msg":"main message",
			"rootKey":"rootValue",
			"someGroup":{
				"subKey":"subValue",
				"mainKey":"mainValue"
			}
		}
	*/
}

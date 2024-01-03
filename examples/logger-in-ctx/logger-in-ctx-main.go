package main

import (
	"context"
	"log/slog"
	"os"

	slogctx "github.com/veqryn/slog-context"
)

// This workflow has us pass the *slog.Logger around inside a context.Context.
// This lets us add attributes and groups to the logger, while naturally
// keeping the logger scoped just like the context itself is scoped.
//
// This eliminates the need to use the default package-level slog, and also
// eliminates the need to add a *slog.Logger as yet another argument to all
// functions.
//
// You can still get the Logger out of the context at any time, and pass it
// around manually if needed, but since contexts are already passed to most
// functions, passing the logger explicitly is now optional.
//
// Attributes and key-value pairs like request-id, trace-id, user-id, etc, can
// be added to the logger in the context, and as the context propagates the
// logger and its attributes will propagate with it, adding these to any log
// lines using that context.
func main() {
	h := slog.NewJSONHandler(os.Stdout, nil)
	slog.SetDefault(slog.New(h))

	// Store the logger inside the context:
	ctx := slogctx.NewCtx(context.Background(), slog.Default())

	// Get the logger back out again at any time, for manual usage:
	log := slogctx.FromCtx(ctx)
	log.Warn("warning")
	/*
		{
			"time":"2023-11-14T00:53:46.361201-07:00",
			"level":"INFO",
			"msg":"warning"
		}
	*/

	// Add attributes directly to the logger in the context:
	ctx = slogctx.With(ctx, "rootKey", "rootValue")

	// Create a group directly on the logger in the context:
	ctx = slogctx.WithGroup(ctx, "someGroup")

	// With and wrapper methods have the same args signature as slog methods,
	// and can take a mix of slog.Attr and key-value pairs.
	ctx = slogctx.With(ctx, slog.String("subKey", "subValue"), slog.Bool("someBool", true))

	// Access the logger in the context directly with handy wrappers for Debug/Info/Warn/Error/Log/LogAttrs:
	slogctx.Info(ctx, "main message", "mainKey", "mainValue")
	/*
		{
			"time":"2023-11-14T00:53:46.363072-07:00",
			"level":"INFO",
			"msg":"main message",
			"rootKey":"rootValue",
			"someGroup":{
				"subKey":"subValue",
				"someBool":true,
				"mainKey":"mainValue"
			}
		}
	*/
}

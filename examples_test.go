package slogctx_test

import (
	"context"
	"log/slog"
	"os"

	slogctx "github.com/veqryn/slog-context"
)

func ExampleNewHandler() {
	// This workflow lets us use slog as normal, while adding the ability to put
	// slog attributes into the context which will then show up at the start or end
	// of log lines.
	//
	// This is useful when you are not passing a *slog.Logger around to different
	// functions (because you are making use of the default package-level slog),
	// but you are passing a context.Context around.
	//
	// This can also be used when a library or vendor code you don't control is
	// using the default log methods, default logger, or doesn't accept a slog
	// Logger to all functions you wish to add attributes to.
	//
	// Attributes and key-value pairs like request-id, trace-id, user-id, etc, can
	// be added to the context, and the *slogctx.Handler will make sure they
	// are prepended to the start, or appended to the end, of any log lines using
	// that context.

	// Create the *slogctx.Handler middleware
	h := slogctx.NewHandler(slog.NewJSONHandler(os.Stdout, nil), nil)
	slog.SetDefault(slog.New(h))

	ctx := context.Background()

	// Prepend some slog attributes to the start of future log lines:
	ctx = slogctx.Prepend(ctx, "prependKey", "prependValue")

	// Append some slog attributes to the end of future log lines:
	// Prepend and Append have the same args signature as slog methods,
	// and can take a mix of slog.Attr and key-value pairs.
	ctx = slogctx.Append(ctx, slog.String("appendKey", "appendValue"))

	// Use the logger like normal:
	slog.WarnContext(ctx, "main message", "mainKey", "mainValue")
	/*
		{
			"time": "2023-11-15T18:43:23.290798-07:00",
			"level": "WARN",
			"msg": "main message",
			"prependKey": "prependValue",
			"mainKey": "mainValue",
			"appendKey": "appendValue"
		}
	*/

	// Use the logger like normal; add attributes, create groups, pass it around:
	log := slog.With("rootKey", "rootValue")
	log = log.WithGroup("someGroup")
	log = log.With("subKey", "subValue8")

	// The prepended/appended attributes end up in all log lines that use that context
	log.InfoContext(ctx, "main message", "mainKey", "mainValue")
	/*
		{
			"time": "2023-11-14T00:37:03.805196-07:00",
			"level": "INFO",
			"msg": "main message",
			"prependKey": "prependValue",
			"rootKey": "rootValue",
			"someGroup": {
				"subKey": "subValue",
				"mainKey": "mainValue",
				"appendKey": "appendValue"
			}
		}
	*/
}

func ExampleNewCtx() {
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

	h := slog.NewJSONHandler(os.Stdout, nil)
	slog.SetDefault(slog.New(h))

	// Store the logger inside the context:
	ctx := slogctx.NewCtx(context.Background(), slog.Default())

	// Get the logger back out again at any time, for manual usage:
	log := slogctx.FromCtx(ctx)
	log.Warn("warning")

	// Add attributes directly to the logger in the context:
	ctx = slogctx.With(ctx, "rootKey", "rootValue")

	// Create a group directly on the logger in the context:
	ctx = slogctx.WithGroup(ctx, "someGroup")

	// With and wrapper methods have the same args signature as slog methods,
	// and can take a mix of slog.Attr and key-value pairs.
	ctx = slogctx.With(ctx, slog.String("subKey", "subValue"))

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
				"mainKey":"mainValue"
			}
		}
	*/
}

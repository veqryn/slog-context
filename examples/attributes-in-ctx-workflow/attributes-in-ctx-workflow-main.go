package main

import (
	"context"
	"log/slog"
	"os"

	slogcontext "github.com/veqryn/slog-context"
)

// This workflow lets us use slog as normal, while adding the ability to put
// slog attributes into the context which will then show up at the start or end
// of log lines.
//
// This is useful when you are not passing a *slog.Logger around to different
// functions (because you are making use of the default package-level slog),
// but you are passing a context.Context around.
//
// Attributes and key-value pairs like request-id, trace-id, user-id, etc, can
// be added to the context, and the *slogcontext.Handler will make sure they
// are prepended to the start, or appended to the end, of any log lines using
// that context.
func main() {
	// Create the *slogcontext.Handler middleware
	h := slogcontext.NewHandler(slog.NewJSONHandler(os.Stdout, nil), nil)
	slog.SetDefault(slog.New(h))

	ctx := context.Background()

	// Prepend some slog attributes to the start of future log lines:
	ctx = slogcontext.Prepend(ctx, "prependKey", "prependValue")

	// Append some slog attributes to the end of future log lines:
	ctx = slogcontext.Append(ctx, "appendKey", "appendValue")

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
	log = log.With("subKey", "subValue")

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

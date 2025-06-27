package main

import (
	"context"
	"log/slog"
	"os"

	yasctx "github.com/pazams/yasctx"
)

// This workflow lets us use slog as normal, while adding the ability to put
// slog attributes into the context which will then show up at the root level
// of log lines (or group if AddToGroup is used).
//
// Attributes and key-value pairs like request-id, trace-id, user-id, etc, can
// be added to the context, and the *yasctx.Handler will make sure they are added
// to any log lines using that context.
func main() {
	// Create the *yasctx.Handler middleware
	h := yasctx.NewHandler(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(slog.New(h))

	ctx := context.Background()

	// Add some slog attributes to the start of future log lines:
	ctx = yasctx.Add(ctx, "key1", "value1")

	// Add some slog attributes to a specific group:
	ctx = yasctx.AddToGroup(ctx, "group1", slog.String("key2", "value2"))

	// Append some slog attributes to a group that won't be used will show up at the root level:
	ctx = yasctx.AddToGroup(ctx, "group2", slog.String("key3", "value3"))

	// Use the logger like normal:
	slog.Default().WithGroup("group1").WarnContext(ctx, "main message", "mainKey", "mainValue")
	/*
		{
		    "time": "2025-06-26T23:08:13.808079237-06:00",
		    "level": "WARN",
		    "msg": "main message",
		    "key1": "value1",
		    "key3": "value3",
		    "group1": {
		        "key2": "value2",
		        "mainKey": "mainValue"
		    }
		}
	*/
}

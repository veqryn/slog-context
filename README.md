# slog-context
[![tag](https://img.shields.io/github/tag/veqryn/slog-context.svg)](https://github.com/veqryn/slog-context/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-%23007d9c)
[![GoDoc](https://godoc.org/github.com/veqryn/slog-context?status.svg)](https://pkg.go.dev/github.com/veqryn/slog-context)
![Build Status](https://github.com/veqryn/slog-context/actions/workflows/build_and_test.yml/badge.svg)
[![Go report](https://goreportcard.com/badge/github.com/veqryn/slog-context)](https://goreportcard.com/report/github.com/veqryn/slog-context)
[![Coverage](https://img.shields.io/codecov/c/github/veqryn/slog-context)](https://codecov.io/gh/veqryn/slog-context)
[![Contributors](https://img.shields.io/github/contributors/veqryn/slog-context)](https://github.com/veqryn/slog-context/graphs/contributors)
[![License](https://img.shields.io/github/license/veqryn/slog-context)](./LICENSE)

Use golang structured logging (slog) with context.
Add attributes to context. Add and retrieve logger to and from context.

This library supports two different workflows for using slog and context.
These workflows can be used separately or together.

Using the `slogcontext.Handler` lets us `Prepend` and `Append` attributes to
log lines, even when a logger is not passed into a function or in code we don't
control. This is done without storing the logger in the context; instead the
attributes are stored in the context and the Handler picks them up later
whenever a new log line is written.

Using `ToCtx` and `Logger` lets us store the logger itself within a context,
and get it back out again. Wrapper methods `With`/`WithGroup`/`Debug`/`Info`/
`Warn`/`Error`/`Log`/`LogAttrs` let us work directly with a logger residing
with the context (or the default logger if no logger is stored in the context).

## Install
`go get github.com/veqryn/slog-context`

```go
import (
	slogcontext "github.com/veqryn/slog-context"
)
```

## Usage
### Add attributes to context workflow
```go
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
// This can also be used when a library or vendor code you don't control is
// using the default log methods, default logger, or doesn't accept a slog
// Logger to all functions you wish to add attributes to.
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
	// Prepend and Append have the same args signature as slog methods,
	// and can take a mix of slog.Attr and key-value pairs.
	ctx = slogcontext.Append(ctx, slog.String("appendKey", "appendValue"))

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
```

### Add logger to context workflow
```go
package main

import (
	"context"
	"log/slog"
	"os"

	slogcontext "github.com/veqryn/slog-context"
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
	ctx := slogcontext.ToCtx(context.Background(), slog.Default())

	// Get the logger back out again at any time, for manual usage:
	log := slogcontext.Logger(ctx)
	log.Warn("warning")

	// Add attributes directly to the logger in the context:
	ctx = slogcontext.With(ctx, "rootKey", "rootValue")

	// Create a group directly on the logger in the context:
	ctx = slogcontext.WithGroup(ctx, "someGroup")

	// With and wrapper methods have the same args signature as slog methods,
	// and can take a mix of slog.Attr and key-value pairs.
	ctx = slogcontext.With(ctx, slog.String("subKey", "subValue"))

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
```

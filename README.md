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
Add and retrieve logger to and from context.
Add attributes to context.
Automatically read any custom context values, such as OpenTelemetry TraceID.

This library supports two different workflows for using slog and context.
These workflows can be used individually or together at the same time.

#### Attributes Extracted from Context Workflow:

Using the `Handler` lets us `Prepend` and `Append` attributes to
log lines, even when a logger is not passed into a function or in code we don't
control. This is done without storing the logger in the context; instead the
attributes are stored in the context and the Handler picks them up later
whenever a new log line is written.

In that same workflow, the `HandlerOptions` and `AttrExtractor` types let us
extract any custom values from a context and have them automatically be
prepended or appended to all log lines using that context. For example, the
`slogotel.ExtractTraceSpanID` extractor will automatically extract the OTEL
(OpenTelemetry) TraceID and SpanID, and add them to the log record, while also
annotating the Span with an error code if the log is at error level.

#### Logger in Context Workflow:

Using `NewCtx` and `FromCtx` lets us store the logger itself within a context,
and get it back out again. Wrapper methods `With`/`WithGroup`/`Debug`/`Info`/
`Warn`/`Error`/`Log`/`LogAttrs` let us work directly with a logger residing
with the context (or the default logger if no logger is stored in the context).

#### Compatibility with both Slog and Logr
slog-context is compatible with both standard library [slog](https://pkg.go.dev/log/slog)
and with [logr](https://github.com/go-logr/logr), which is an alternative
logging api/interface/frontend.

If only slog is used, only `*slog.Logger`'s will be stored in the context.
If both slog and logr are used, `*slog.Logger` will be automatically converted
to a `logr.Logger` as needed, and vice versa. This allows full interoperability
down the stack and with any libraries that use either slog-context or logr.

### Other Great SLOG Utilities
- [slogctx](https://github.com/veqryn/slog-context): Add attributes to context and have them automatically added to all log lines. Work with a logger stored in context.
- [slogotel](https://github.com/veqryn/slog-context/tree/main/otel): Automatically extract and add [OpenTelemetry](https://opentelemetry.io/) TraceID's to all log lines.
- [slogdedup](https://github.com/veqryn/slog-dedup): Middleware that deduplicates and sorts attributes. Particularly useful for JSON logging.
- [slogbugsnag](https://github.com/veqryn/slog-bugsnag): Middleware that pipes Errors to [Bugsnag](https://www.bugsnag.com/).

## Install

```
go get github.com/veqryn/slog-context
```

```go
import (
	slogctx "github.com/veqryn/slog-context"
)
```

## Usage
### Logger in Context Workflow
```go
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
```


### Attributes Extracted from Context Workflow
#### Append and Prepend
```go
package main

import (
	"context"
	"log/slog"
	"os"

	slogctx "github.com/veqryn/slog-context"
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
// be added to the context, and the *slogctx.Handler will make sure they
// are prepended to the start, or appended to the end, of any log lines using
// that context.
func main() {
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

#### Custom Context Value Extractor
```go
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
```

#### OpenTelemetry TraceID SpanID Extractor
In order to avoid making all users of this repo require all the OTEL libraries,
the OTEL extractor is in a separate module in this repo:

`go get github.com/veqryn/slog-context/otel`

```go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	slogctx "github.com/veqryn/slog-context"
	slogotel "github.com/veqryn/slog-context/otel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	// Create the *slogctx.Handler middleware
	h := slogctx.NewHandler(
		slog.NewJSONHandler(os.Stdout, nil), // The next handler in the chain
		&slogctx.HandlerOptions{
			// Prependers will first add the OTEL Trace ID,
			// then anything else Prepended to the ctx
			Prependers: []slogctx.AttrExtractor{
				slogotel.ExtractTraceSpanID,
				slogctx.ExtractPrepended,
			},
			// Appenders stays as default (leaving as nil would accomplish the same)
			Appenders: []slogctx.AttrExtractor{
				slogctx.ExtractAppended,
			},
		},
	)
	slog.SetDefault(slog.New(h))

	setupOTEL()
}

func main() {
	// Handle OTEL shutdown properly so nothing leaks
	defer traceProvider.Shutdown(context.Background())

	slog.Info("Starting server. Please run: curl localhost:8080/hello")

	// Demonstrate the slogotel.ExtractTraceSpanID with a http server
	http.HandleFunc("/hello", helloHandler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

// helloHandler starts an OTEL Span, then begins a long-running calculation.
// The calculation will fail, and the logging at Error level will mark the span
// as codes.Error.
func helloHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "helloHandler")
	defer span.End()

	slogctx.Info(ctx, "starting long calculation...")
	/*
		{
			"time": "2023-11-17T03:11:20.584592-07:00",
			"level": "INFO",
			"msg": "starting long calculation...",
			"TraceID": "15715df45965b4a2db6dc103a76e52ae",
			"SpanID": "76d364cdd598c895"
		}
	*/

	time.Sleep(5 * time.Second)
	slogctx.Error(ctx, "something failed...")
	/*
		{
			"time": "2023-11-17T03:11:25.586464-07:00",
			"level": "ERROR",
			"msg": "something failed...",
			"TraceID": "15715df45965b4a2db6dc103a76e52ae",
			"SpanID": "76d364cdd598c895"
		}
	*/

	w.WriteHeader(http.StatusInternalServerError)

	// The OTEL exporter will soon after output the trace, which will include this and much more:
	/*
		{
			"Name": "helloHandler",
			"SpanContext": {
				"TraceID": "15715df45965b4a2db6dc103a76e52ae",
				"SpanID": "76d364cdd598c895"
			},
			"Status": {
				"Code": "Error",
				"Description": "something failed..."
			}
		}
	*/
}

var (
	tracer        trace.Tracer
	traceProvider *sdktrace.TracerProvider
)

// OTEL setup
func setupOTEL() {
	exp, err := stdouttrace.New()
	if err != nil {
		panic(err)
	}

	// Create a new tracer provider with a batch span processor and the given exporter.
	traceProvider = newTraceProvider(exp)

	// Set as global trace provider
	otel.SetTracerProvider(traceProvider)

	// Finally, set the tracer that can be used for this package.
	tracer = traceProvider.Tracer("ExampleService")
}

// OTEL tracer provider setup
func newTraceProvider(exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("ExampleService"),
		),
	)
	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
}
```

### slog-multi Middleware
This library has a convenience method that allow it to interoperate with [github.com/samber/slog-multi](https://github.com/samber/slog-multi),
in order to easily setup slog workflows such as pipelines, fanout, routing, failover, etc.
```go
slog.SetDefault(slog.New(slogmulti.
	Pipe(slogctx.NewMiddleware(&slogctx.HandlerOptions{})).
	Pipe(slogdedup.NewOverwriteMiddleware(&slogdedup.OverwriteHandlerOptions{})).
	Handler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})),
))
```

## Breaking Changes
### O.4.0 -> 0.5.0
Package function `ToCtx` renamed to `NewCtx`.
Package function `Logger` renamed to `FromCtx`.

Package renamed from `slogcontext` to `slogctx`.
To fix, change this:
```go
import "github.com/veqryn/slog-context"
var h = slogcontext.NewHandler(slog.NewJSONHandler(os.Stdout, nil), nil)
```
To this:
```go
import "github.com/veqryn/slog-context"
var h = slogctx.NewHandler(slog.NewJSONHandler(os.Stdout, nil), nil)
```
Named imports are unaffected.

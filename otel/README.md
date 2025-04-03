# slog-context
[![tag](https://img.shields.io/github/tag/veqryn/slog-context.svg)](https://github.com/veqryn/slog-context/tags)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-%23007d9c)
[![GoDoc](https://godoc.org/github.com/veqryn/slog-context?status.svg)](https://pkg.go.dev/github.com/veqryn/slog-context/otel)
![Build Status](https://github.com/veqryn/slog-context/actions/workflows/build_and_test.yml/badge.svg)
[![Go report](https://goreportcard.com/badge/github.com/veqryn/slog-context/otel)](https://goreportcard.com/report/github.com/veqryn/slog-context/otel)
[![Coverage](https://img.shields.io/codecov/c/github/veqryn/slog-context)](https://codecov.io/gh/veqryn/slog-context)
[![Contributors](https://img.shields.io/github/contributors/veqryn/slog-context)](https://github.com/veqryn/slog-context/graphs/contributors)
[![License](https://img.shields.io/github/license/veqryn/slog-context)](../LICENSE)

Golang SLOG middleware that automatically extracts the OpenTelemetry (OTEL)
Trace ID and Span ID from the context and add it to all log lines.

The `slogotel.ExtractTraceSpanID` extractor, when used with the [slogctx](github.com/veqryn/slog-context)
Handler, will automatically read and extract the Trace ID and Span ID attributes
from a `context.Context`, and add them to the log record at log time. This is
done without storing the logger in the context; instead the Handler picks them
up later whenever a new log line is written, even if it is written in a library
or code you don't control. `ExtractTraceSpanID` will also annotate the Span with
an error code if the log is at error level.

### Other Great SLOG Utilities
- [slogctx](https://github.com/veqryn/slog-context): Add attributes to context and have them automatically added to all log lines. Work with a logger stored in context.
- [slogotel](https://github.com/veqryn/slog-context/tree/main/otel): Automatically extract and add [OpenTelemetry](https://opentelemetry.io/) TraceID's to all log lines.
- [sloggrpc](https://github.com/veqryn/slog-context/tree/main/grpc): Instrument [GRPC](https://grpc.io/) with automatic logging of all requests and responses.
- [slogdedup](https://github.com/veqryn/slog-dedup): Middleware that deduplicates and sorts attributes. Particularly useful for JSON logging.
- [slogbugsnag](https://github.com/veqryn/slog-bugsnag): Middleware that pipes Errors to [Bugsnag](https://www.bugsnag.com/).

## Install
```
go get github.com/veqryn/slog-context/otel
```

```go
import (
	slogotel "github.com/veqryn/slog-context/otel"
)
```

## Usage
### Attributes Extracted from Context Workflow
#### OpenTelemetry TraceID SpanID Extractor
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
	Pipe(slogctx.NewMiddleware(&slogctx.HandlerOptions{
		Prependers: []slogctx.AttrExtractor{
			slogotel.ExtractTraceSpanID,
			slogctx.ExtractPrepended,
		},
	})).
	Pipe(slogdedup.NewOverwriteMiddleware(&slogdedup.OverwriteHandlerOptions{})).
	Handler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})),
))
```

## Other Notes
This module is separate from the main [slogctx](github.com/veqryn/slog-context)
module in order to prevent `slogctx` from requiring OTEL and all its many
dependencies.

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
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
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

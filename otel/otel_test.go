package slogotel

import (
	"context"
	"log/slog"
	"testing"

	slogctx "github.com/veqryn/slog-context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

func init() {
	DefaultSpanAddEventMinLevel = slog.LevelWarn
}

func TestExtractTraceSpanID(t *testing.T) {
	tester := &testHandler{}
	h := slogctx.NewHandler(
		tester,
		&slogctx.HandlerOptions{
			Prependers: []slogctx.AttrExtractor{
				ExtractTraceSpanID,
				slogctx.ExtractPrepended,
			},
		})
	ctx := slogctx.NewCtx(context.Background(), slog.New(h))

	// Manually create the trace id and span id so the test is repeatable
	traceID, err := trace.TraceIDFromHex(`0123456789abcdef0123456789abcdef`)
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex(`0123456789abcdef`)
	if err != nil {
		t.Fatal(err)
	}

	// Manually set the id's
	span := &recorderSpan{
		sc: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: traceID,
			SpanID:  spanID,
		}),
	}

	ctx = trace.ContextWithSpan(ctx, span)

	ctx = slogctx.Prepend(ctx, "prepend1", "arg1", "prepend1", "arg2")
	ctx = slogctx.Append(ctx, "append1", "arg1", "append1", "arg2")

	ctx = slogctx.With(ctx, "with1", "arg1", "with1", "arg2")
	ctx = slogctx.WithGroup(ctx, "group1")

	slogctx.Error(ctx, "main message", "main1", "arg1", "main1", "arg2")

	expectedText := `time=2023-09-29T13:00:59.000Z level=ERROR msg="main message" TraceID=0123456789abcdef0123456789abcdef SpanID=0123456789abcdef prepend1=arg1 prepend1=arg2 with1=arg1 with1=arg2 group1.main1=arg1 group1.main1=arg2 group1.append1=arg1 group1.append1=arg2
`
	if s := tester.String(); s != expectedText {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedText, s)
	}

	b, err := tester.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	expectedJSON := `{"time":"2023-09-29T13:00:59Z","level":"ERROR","msg":"main message","TraceID":"0123456789abcdef0123456789abcdef","SpanID":"0123456789abcdef","prepend1":"arg1","prepend1":"arg2","with1":"arg1","with1":"arg2","group1":{"main1":"arg1","main1":"arg2","append1":"arg1","append1":"arg2"}}
`
	if string(b) != expectedJSON {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedText, string(b))
	}

	if *span.status != codes.Error {
		t.Errorf("Expected: %v; Got: %v", codes.Error, *span.status)
	}
	if *span.description != "main message" {
		t.Errorf("Expected: %v; Got: %v", "main message", *span.description)
	}
	if (*span.err).Error() != "main message" {
		t.Errorf("Expected: %v; Got: %v", "main message", (*span.err).Error())
	}
}

func TestExtractTraceSpanIDAddEvent(t *testing.T) {
	tester := &testHandler{}
	h := slogctx.NewHandler(
		tester,
		&slogctx.HandlerOptions{
			Prependers: []slogctx.AttrExtractor{
				ExtractTraceSpanID,
				slogctx.ExtractPrepended,
			},
		})
	ctx := slogctx.NewCtx(context.Background(), slog.New(h))

	// Manually create the trace id and span id so the test is repeatable
	traceID, err := trace.TraceIDFromHex(`123456789abcdef0123456789abcdef0`)
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex(`123456789abcdef0`)
	if err != nil {
		t.Fatal(err)
	}

	// Manually set the id's
	span := &recorderSpan{
		sc: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: traceID,
			SpanID:  spanID,
		}),
	}

	ctx = trace.ContextWithSpan(ctx, span)

	slogctx.Warn(ctx, "some warn message")

	if *span.event != "some warn message" {
		t.Errorf("Expected: %v; Got: %v", "some warn message", *span.description)
	}
}

// recorderSpan is an implementation of Span that performs no operations.
type recorderSpan struct {
	embedded.Span
	sc          trace.SpanContext
	status      *codes.Code
	description *string
	err         *error
	event       *string
}

var _ trace.Span = &recorderSpan{}

// SpanContext returns an empty span context.
func (r *recorderSpan) SpanContext() trace.SpanContext { return r.sc }

// IsRecording always returns true.
func (*recorderSpan) IsRecording() bool { return true }

// SetStatus records the code and description.
func (r *recorderSpan) SetStatus(c codes.Code, d string) {
	r.status = &c
	r.description = &d
}

// SetError does nothing.
func (*recorderSpan) SetError(bool) {}

// SetAttributes does nothing.
func (*recorderSpan) SetAttributes(...attribute.KeyValue) {}

// End does nothing.
func (*recorderSpan) End(...trace.SpanEndOption) {}

// RecordError records the error
func (r *recorderSpan) RecordError(err error, opts ...trace.EventOption) {
	r.err = &err
}

// AddEvent records the event
func (r *recorderSpan) AddEvent(event string, opts ...trace.EventOption) {
	r.event = &event
}

// SetName does nothing.
func (*recorderSpan) SetName(string) {}

// TracerProvider returns nil.
func (*recorderSpan) TracerProvider() trace.TracerProvider { return nil }

func (r *recorderSpan) AddLink(link trace.Link) {}

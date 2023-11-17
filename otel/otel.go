package slogotel

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// DefaultKeyTraceID is the default attribute key sent to slog for trace id's.
var DefaultKeyTraceID = "TraceID" // Copied from otel stdouttrace

// DefaultKeySpanID is the default attribute key sent to slog for span id's.
var DefaultKeySpanID = "SpanID" // Copied from otel stdouttrace

// ExtractTraceSpanID is an AttrExtractor that returns any valid TraceID and
// SpanID in any recording span.
// In addition, if there is an error log being created inside a span, the span
// is coded as an error, with the log message as the description.
// The returned slice should not be appended to or modified in any way.
// Doing so will cause a race condition.
func ExtractTraceSpanID(ctx context.Context, _ time.Time, recordLvl slog.Level, recordMsg string) []slog.Attr {
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		if recordLvl >= slog.LevelError {
			span.SetStatus(codes.Error, recordMsg)
		}

		var attrs []slog.Attr
		spanCtx := span.SpanContext()
		if spanCtx.HasTraceID() {
			attrs = append(attrs, slog.String(DefaultKeyTraceID, spanCtx.TraceID().String()))
		}
		if spanCtx.HasSpanID() {
			attrs = append(attrs, slog.String(DefaultKeySpanID, spanCtx.SpanID().String()))
		}
		return attrs
	}
	return nil
}

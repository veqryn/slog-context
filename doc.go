/*
Package slogctx lets you use golang structured logging (slog) with context.
Add and retrieve logger to and from context. Add attributes to context.
Automatically read any custom context values, such as OpenTelemetry TraceID.

This library supports two different workflows for using slog and context.
These workflows can be used individually or together at the same time.

Attributes Extracted from Context Workflow:

Using the slogctx.NewHandler lets us Prepend and Append attributes to
log lines, even when a logger is not passed into a function or in code we don't
control. This is done without storing the logger in the context; instead the
attributes are stored in the context and the Handler picks them up later
whenever a new log line is written.

In that same workflow, the HandlerOptions and AttrExtractor types let us
extract any custom values from a context and have them automatically be
prepended or appended to all log lines using that context. For example, the
slogotel.ExtractTraceSpanID extractor will automatically extract the OTEL
(OpenTelemetry) TraceID and SpanID, and add them to the log record, while also
annotating the Span with an error code if the log is at error level.

Logger in Context Workflow:

Using NewCtx and FromCtx lets us store the logger itself within a context,
and get it back out again. Wrapper methods With / WithGroup / Debug / Info /
Warn / Error / Log / LogAttrs let us work directly with a logger residing
with the context (or the default logger if no logger is stored in the context).
*/
package slogctx

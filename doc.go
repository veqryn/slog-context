/*
Package yasctx lets you use golang structured logging (slog) with context.

Using the yasctx.NewHandler lets us Add attributes to
log lines, even when a logger is not passed into a function or in code we don't
control. This is done without storing the logger in the context; instead the
attributes are stored in the context and the Handler picks them up later
whenever a new log line is written.
*/
package yasctx

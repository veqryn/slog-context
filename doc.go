/*
Package slogcontext lets you use golang structured logging (slog) with context.
Add attributes to context. Add and retrieve logger to and from context.

This library supports two different workflows for using slog and context.
These workflows can be used separately or together.

Using the Handler lets us Prepend and Append attributes to
log lines, even when a logger is not passed into a function or in code we don't
control. This is done without storing the logger in the context; instead the
attributes are stored in the context and the Handler picks them up later
whenever a new log line is written.

Using ToCtx and Logger lets us store the logger itself within a context,
and get it back out again. Wrapper methods With / WithGroup / Debug / Info /
Warn / Error / Log / LogAttrs let us work directly with a logger residing
with the context (or the default logger if no logger is stored in the context).
*/
package slogcontext

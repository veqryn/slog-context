package slogctx

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

// ErrKey is the key used by handlers for an error
// when the log method is called. The associated Value is an error.
const ErrKey = "err" // TODO: consider making a var to allow changes before.

// Err is a convenience method that creates a [slog.Attr] out of an error.
// It uses a consistent key: [ErrKey]
func Err(err error) slog.Attr {
	return slog.Any(ErrKey, err)
}

// With calls With on the logger stored in the context,
// or if there isn't any, on the default logger.
// This new logger is stored in a child context and the new context is returned.
// [slog.Logger.With] returns a Logger that includes the given attributes in each output
// operation. Arguments are converted to attributes as if by [slog.Logger.Log].
func With(ctx context.Context, args ...any) context.Context {
	return NewCtx(ctx, FromCtx(ctx).With(args...))
}

// WithGroup calls WithGroup on the logger stored in the context,
// or if there isn't any, on the default logger.
// This new logger is stored in a child context and the new context is returned.
// [slog.Logger.WithGroup] returns a Logger that starts a group, if name is non-empty.
// The keys of all attributes added to the Logger will be qualified by the given
// name. (How that qualification happens depends on the [Handler.WithGroup]
// method of the Logger's Handler.)
//
// If name is empty, WithGroup returns the receiver.
func WithGroup(ctx context.Context, name string) context.Context {
	return NewCtx(ctx, FromCtx(ctx).WithGroup(name))
}

// Debug calls DebugContext on the logger stored in the context,
// or if there isn't any, on the default logger.
// [slog.Logger.DebugContext] logs at LevelDebug with the given context.
func Debug(ctx context.Context, msg string, args ...any) {
	log(ctx, FromCtx(ctx), slog.LevelDebug, msg, args...)
}

// Info calls InfoContext on the logger stored in the context,
// or if there isn't any, on the default logger.
// [slog.Logger.InfoContext] logs at LevelInfo with the given context.
func Info(ctx context.Context, msg string, args ...any) {
	log(ctx, FromCtx(ctx), slog.LevelInfo, msg, args...)
}

// Warn calls WarnContext on the logger stored in the context,
// or if there isn't any, on the default logger.
// [slog.Logger.WarnContext] logs at LevelWarn with the given context.
func Warn(ctx context.Context, msg string, args ...any) {
	log(ctx, FromCtx(ctx), slog.LevelWarn, msg, args...)
}

// Error calls ErrorContext on the logger stored in the context,
// or if there isn't any, on the default logger.
// [slog.Logger.ErrorContext] logs at LevelError with the given context.
func Error(ctx context.Context, msg string, args ...any) {
	log(ctx, FromCtx(ctx), slog.LevelError, msg, args...)
}

// Log calls Log on the logger stored in the context,
// or if there isn't any, on the default logger.
// [slog.Logger.Log] emits a log record with the current time and the given level and message.
// The Record's Attrs consist of the Logger's attributes followed by
// the Attrs specified by args.
//
// The attribute arguments are processed as follows:
//   - If an argument is an Attr, it is used as is.
//   - If an argument is a string and this is not the last argument,
//     the following argument is treated as the value and the two are combined
//     into an Attr.
//   - Otherwise, the argument is treated as a value with key "!BADKEY".
func Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	log(ctx, FromCtx(ctx), level, msg, args...)
}

// LogAttrs calls LogAttrs on the logger stored in the context,
// or if there isn't any, on the default logger.
// [slog.Logger.LogAttrs] is a more efficient version of [slog.Logger.Log] that accepts only Attrs.
func LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	logAttrs(ctx, FromCtx(ctx), level, msg, attrs...)
}

// log is the low-level logging method for methods that take ...any.
// It must always be called directly by an exported logging method
// or function, because it uses a fixed call depth to obtain the pc.
// This is copied from golang sdk.
func log(ctx context.Context, l *slog.Logger, level slog.Level, msg string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}

	if !l.Enabled(ctx, level) {
		return
	}

	var pc uintptr
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3, pcs[:])
	pc = pcs[0]

	r := slog.NewRecord(time.Now(), level, msg, pc)
	r.Add(args...)
	_ = l.Handler().Handle(ctx, r)
}

// logAttrs is like log, but for methods that take ...Attr.
// This is copied from golang sdk.
func logAttrs(ctx context.Context, l *slog.Logger, level slog.Level, msg string, attrs ...slog.Attr) {
	if ctx == nil {
		ctx = context.Background()
	}

	if !l.Enabled(ctx, level) {
		return
	}

	var pc uintptr
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3, pcs[:])
	pc = pcs[0]

	r := slog.NewRecord(time.Now(), level, msg, pc)
	r.AddAttrs(attrs...)
	_ = l.Handler().Handle(ctx, r)
}

package sloghttp

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	propagation "github.com/veqryn/slog-context/propagation"
)

// AttrCollection is an http middleware that collects slog.Attr from
// any and all later middlewares and the final http request handler, and makes
// them available to all middlewares and the request handler.
// Essentially, it lets you collect slog attributes that are discovered later in
// the stack (such as authentication and user ID's, derived values, attributes
// only discovered halfway-through the final request handler after several db
// queries, etc), and be able to have them be included in the log lines of other
// middlewares (such as a middleware that logs all requests that come in).
//
// Requires the use of slogctx.Handler, as a wrapper or middleware around your
// slog formatter/sink.
//
// Attributes are added by calls to sloghttp.With, and the attributes are then
// stored inside the context. All calls log that include the context will
// automatically have all the attributes included (ex: slogctx.Info, or
// slog.InfoContext).
func AttrCollection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use the propagation initializer from the propagation package.
		// It is a no-op if propagation was already initialized on the context.
		r = r.WithContext(propagation.Init(r.Context()))
		next.ServeHTTP(w, r)
	})
}

// With adds the provided slog.Attr's to the context. If used with
// sloghttp.AttrCollection it will add them to the context in a way
// that is visible to all intermediate middlewares and functions between the
// collector middleware and the call to With.
func With(ctx context.Context, args ...any) context.Context {
	// Delegate to the propagation-aware add function. It will fall back to
	// the attached-logger flow when propagation isn't initialized.
	return propagation.Add(ctx, args...)
}

// ExtractAttrCollection is a slogctx Extractor that must be used with a
// slogctx.Handler (via slogctx.HandlerOptions) as Prependers or Appenders.
// It will cause the Handler to add the Attributes added by sloghttp.With to all
// log lines using that same context.
func ExtractAttrCollection(ctx context.Context, t time.Time, lvl slog.Level, msg string) []slog.Attr {
	return propagation.ExtractAttrs(ctx, t, lvl, msg)
}

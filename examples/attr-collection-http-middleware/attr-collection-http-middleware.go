package main

import (
	"log/slog"
	"net/http"
	"os"

	slogctx "github.com/veqryn/slog-context"
	sloghttp "github.com/veqryn/slog-context/http"
)

func init() {
	// Create the *slogctx.Handler middleware
	h := slogctx.NewHandler(
		slog.NewJSONHandler(os.Stdout, nil), // The next or final handler in the chain
		&slogctx.HandlerOptions{
			// Prependers will first add any sloghttp.With attributes,
			// then anything else Prepended to the ctx
			Prependers: []slogctx.AttrExtractor{
				sloghttp.ExtractAttrCollection, // our sloghttp middleware extractor
				slogctx.ExtractPrepended,       // for all other prepended attributes
			},
		},
	)
	slog.SetDefault(slog.New(h))
}

func main() {
	slog.Info("Starting server. Please run: curl localhost:8080/hello?id=24680")

	// Wrap our final handler inside our middlewares.
	// AttrCollector -> Request Logging -> Final Endpoint Handler (helloUser)
	handler := sloghttp.AttrCollection(
		httpLoggingMiddleware(
			http.HandlerFunc(helloUser),
		),
	)

	// Demonstrate the sloghttp middleware with a http server
	http.Handle("/hello", handler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

// This is a stand-in for a middleware that might be capturing and logging out
// things like the response code, request body, response body, url, method, etc.
// It doesn't have access to any of the new context objects's created within the
// next handler. But it should still log with any of the attributes added to our
// sloghttp.Middleware, via sloghttp.With.
func httpLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add some logging context/baggage before the handler
		r = r.WithContext(sloghttp.With(r.Context(), "path", r.URL.Path))

		// Call the next handler
		next.ServeHTTP(w, r)

		// Log out that we had a response. This would be where we could add
		// things such as the response status code, body, etc.

		// Should also have both "path" and "id", but not "foo".
		// Having "id" included in the log is the whole point of this package!
		slogctx.Info(r.Context(), "Response", "method", r.Method)
		/*
			{
				"time": "2024-04-01T00:06:11Z",
				"level": "INFO",
				"msg": "Response",
				"path": "/hello",
				"id": "24680",
				"method": "GET"
			}
		*/
	})
}

// This is our final api endpoint handler
func helloUser(w http.ResponseWriter, r *http.Request) {
	// Stand-in for a User ID.
	// Add it to our middleware's context
	id := r.URL.Query().Get("id")

	// sloghttp.With will add the "id" to the middleware, because it is a
	// synchronized map. It will show up in all log calls up and down the stack,
	// until the request sloghttp middleware exits.
	ctx := sloghttp.With(r.Context(), "id", id)

	// The regular slogctx.With will add "foo" only to the Returned context,
	// which will limits its scope to the rest of this function (helloUser) and
	// any functions called by helloUser and passed this context.
	// The original caller of helloUser and all the middlewares will NOT see
	// "foo", because it is only part of the newly returned ctx.
	ctx = slogctx.With(ctx, "foo", "bar")

	// Log some things.
	// Should also have both "path", "id", and "foo"
	slogctx.Info(ctx, "saying hello...")
	/*
		{
			"time": "2024-04-01T00:06:11Z",
			"level": "INFO",
			"msg": "saying hello...",
			"path": "/hello",
			"id": "24680",
			"foo": "bar"
		}
	*/

	// Response
	_, _ = w.Write([]byte("Hello User #" + id))
}

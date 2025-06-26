package slogctx_test

import (
	"log/slog"
	"net/http"
	"os"

	slogctx "github.com/veqryn/slog-context"
)

func ExampleExtractAttrCollection() {
	// Create the *slogctx.Handler middleware
	h := slogctx.NewHandler(
		slog.NewJSONHandler(os.Stdout, nil), // The next or final handler in the chain
	)
	slog.SetDefault(slog.New(h))
}

func ExampleAttrCollection() {
	// This is our final api endpoint handler
	helloUserHandler := func(w http.ResponseWriter, r *http.Request) {
		// Stand-in for a User ID.
		// Add it to our middleware's context
		id := r.URL.Query().Get("id")

		// sloghttp.With will add the "id" to to the middleware, because it is a
		// synchronized map. It will show up in all log calls up and down the stack,
		// until the request sloghttp middleware exits.
		ctx := slogctx.AddWithPropagation(r.Context(), "id", id)

		// Log some things. Should also have both "path", "id"
		slog.InfoContext(ctx, "saying hello...")
		_, _ = w.Write([]byte("Hello User #" + id))
	}

	// This is a stand-in for a middleware that might be capturing and logging out
	// things like the response code, request body, response body, url, method, etc.
	// It doesn't have access to any of the new context objects's created within the
	// next handler. But it should still log with any of the attributes added to our
	// sloghttp.Middleware, via sloghttp.With.
	httpLoggingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add some logging context/baggage before the handler
			r = r.WithContext(slogctx.AddWithPropagation(r.Context(), "path", r.URL.Path))

			// Call the next handler
			next.ServeHTTP(w, r)

			// Log out that we had a response. This would be where we could add
			// things such as the response status code, body, etc.
			// Should also have both "path" and "id", but not "foo".
			// Having "id" included in the log is the whole point of this package!
			slog.InfoContext(r.Context(), "Response", "method", r.Method)
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

	// Wrap our final handler inside our middlewares.
	// AttrCollector -> Request Logging -> Final Endpoint Handler (helloUser)
	handler := middlewareWithInitGlobal(
		httpLoggingMiddleware(
			http.HandlerFunc(helloUserHandler),
		),
	)

	// Demonstrate the sloghttp middleware with a http server
	http.Handle("/hello", handler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

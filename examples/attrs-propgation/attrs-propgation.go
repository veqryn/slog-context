package main

import (
	"log/slog"
	"net/http"
	"os"

	yasctx "github.com/pazams/yasctx"
)

func init() {
	// Create the *yasctx.Handler middleware
	h := yasctx.NewHandler(
		slog.NewJSONHandler(os.Stdout, nil), // The next or final handler in the chain
	)
	slog.SetDefault(slog.New(h))
}

func main() {
	slog.Info("Starting server. Please run: curl localhost:8080/hello?id=24680")

	// Wrap our final handler inside our middlewares.
	handler := middlewareWithInitGlobal(
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
		r = r.WithContext(yasctx.AddWithPropagation(r.Context(), "path", r.URL.Path))

		// Call the next handler
		next.ServeHTTP(w, r)

		// Log out that we had a response. This would be where we could add
		// things such as the response status code, body, etc.
		// Should also have both "path" and "id", but not "foo".
		// Having "id" included in the log is the whole point of this package!
		slog.InfoContext(r.Context(), "Response", "method", r.Method)
		/*
			{
			    "time": "2025-06-26T23:29:27.034817656-06:00",
			    "level": "INFO",
			    "msg": "Response",
			    "path": "/hello",
			    "id": "24680",
			    "method": "GET"
			}
		*/
	})
}

func middlewareWithInitGlobal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(yasctx.InitPropagation(r.Context()))
		next.ServeHTTP(w, r)
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
	ctx := yasctx.AddWithPropagation(r.Context(), "id", id)

	// The regular yasctx.Add  will add "foo" only to the Returned context,
	// which will limits its scope to the rest of this function (helloUser) and
	// any functions called by helloUser and passed this context.
	// The original caller of helloUser and all the middlewares will NOT see
	// "foo", because it is only part of the newly returned ctx.
	ctx = yasctx.Add(ctx, "foo", "bar")

	// Log some things.
	// Should also have both "path", "id", and "foo"
	slog.InfoContext(ctx, "saying hello...")
	/*
		{
		    "time": "2025-06-26T23:29:27.034778494-06:00",
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

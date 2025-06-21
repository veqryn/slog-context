package sloghttp

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	slogctx "github.com/veqryn/slog-context"
	"github.com/veqryn/slog-context/internal/test"
)

func TestAttrCollection(t *testing.T) {
	// Create the *slogctx.Handler middleware
	tester := &test.Handler{}
	h := slogctx.NewHandler(
		tester,
		&slogctx.HandlerOptions{
			Prependers: []slogctx.AttrExtractor{
				ExtractAttrCollection,    // our sloghttp middleware extractor
				slogctx.ExtractPrepended, // for all other prepended attributes
			},
		},
	)
	// Using slog.SetDefault in tests can be problematic,
	// as the steps run in parallel and step on each other
	l := slog.New(h)
	ctx := context.Background()

	// Setup with our sloghttp middleware, a logging middleware, then our endpoint
	httpHandler := AttrCollection(
		httpLoggingMiddleware(l)(
			http.HandlerFunc(helloUser(l)),
		),
	)

	srv := httptest.NewUnstartedServer(httpHandler)
	srv.Config.BaseContext = func(l net.Listener) context.Context {
		return ctx
	}
	srv.Start()
	defer srv.Close()

	//t.Log(srv.URL)
	resp, err := http.Get(srv.URL + "/?id=24680")
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Expected status code of 200; Got: ", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(respBody) != "Hello User #24680" {
		t.Fatal("Response body incorrect: ", string(respBody))
	}

	jsn, err := tester.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	expected := `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"saying hello...","path":"/","id":"24680","foo":"bar"}
{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"Response","path":"/","id":"24680","method":"GET"}
`
	if string(jsn) != expected {
		t.Error("Incorrect logs received: ", string(jsn))
	}
}

// This is a stand-in for a middleware that might be capturing and logging out
// things like the response code, request body, response body, url, method, etc.
// It doesn't have access to any of the new context's created within the next handler.
// But it should still log with any of the attributes added to our sloghttp.Middleware.
func httpLoggingMiddleware(l *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add some logging context/baggage before the handler
			r = r.WithContext(With(r.Context(), "path", r.URL.Path))

			// Call the next handler
			next.ServeHTTP(w, r)

			// Log out that we had a response
			l.InfoContext(r.Context(), "Response", "method", r.Method) // should also have both "path" and "id", but not "foo"
		})
	}
}

// This is a stand-in for an api endpoint
func helloUser(l *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Stand-in for a User ID.
		// Add it to our middleware's map
		id := r.URL.Query().Get("id")

		// sloghttp.With will add "id" to to the middleware, because it is a synchronized map.
		// It will show up in all log calls up and down the stack, until the request sloghttp middleware exits.
		ctx := With(r.Context(), "id", id)

		// "foo" only to the Returned context, which will limits its scope
		// to the rest of this function and any sub-functions called.
		// The callers of helloUser and all the middlewares will not see "foo".
		ctx = slogctx.Append(ctx, slog.String("foo", "bar")) // also works

		// Log some things
		l.InfoContext(ctx, "saying hello...") // should also have both "path", "id", and "foo"

		// Respond
		_, _ = w.Write([]byte("Hello User #" + id))
	}
}

func TestOutsideRequest(t *testing.T) {
	// Create the *slogctx.Handler middleware
	tester := &test.Handler{}
	h := slogctx.NewHandler(
		tester,
		&slogctx.HandlerOptions{
			Prependers: []slogctx.AttrExtractor{
				ExtractAttrCollection,    // our sloghttp middleware extractor
				slogctx.ExtractPrepended, // for all other prepended attributes
			},
		},
	)
	ctx := context.Background()
	l := slog.New(h)

	ctx = With(ctx, "id", "13579")
	ctx = With(ctx) // Should be ignored

	l.InfoContext(ctx, "utility method") // should also have "id"

	jsn, err := tester.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	expected := `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"utility method","id":"13579"}
`
	if string(jsn) != expected {
		t.Error("Incorrect logs received: ", string(jsn))
	}
}

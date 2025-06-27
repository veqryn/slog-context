# yasctx
Yet Anoter Slog Context libary
[![tag](https://img.shields.io/github/tag/pazams/yasctx.svg)](https://github.com/pazams/yasctx/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-%23007d9c)
[![GoDoc](https://godoc.org/github.com/pazams/yasctx?status.svg)](https://pkg.go.dev/github.com/pazams/yasctx)
![Build Status](https://github.com/pazams/yasctx/actions/workflows/build_and_test.yml/badge.svg)
[![Go report](https://goreportcard.com/badge/github.com/pazams/yasctx)](https://goreportcard.com/report/github.com/pazams/yasctx)
[![Coverage](https://img.shields.io/codecov/c/github/pazams/yasctx)](https://codecov.io/gh/pazams/yasctx)
[![Contributors](https://img.shields.io/github/contributors/pazams/yasctx)](https://github.com/pazams/yasctx/graphs/contributors)
[![License](https://img.shields.io/github/license/pazams/yasctx)](./LICENSE)


Package yasctx lets you use golang structured logging (slog) with context.

Using the yasctx.NewHandler lets us add attributes to
log lines, even when a logger is not passed into a function or in code we don't
control. This is done without storing the logger in the context; instead the
attributes are stored in the context and the Handler picks them up later
whenever a new log line is written.

This library was forked and pivoted from https://github.com/veqryn/slog-context
## Install

```
go get github.com/pazams/yasctx
```

```go
import (
	yasctx "github.com/pazams/yasctx"
)
```

## Usage
[Examples in repo](examples/)
### Attributes Extracted from Context Workflow
```go
package main

import (
	"context"
	"log/slog"
	"os"

	yasctx "github.com/pazams/yasctx"
)

// This workflow lets us use slog as normal, while adding the ability to put
// slog attributes into the context which will then show up at the root level
// of log lines (or group if AddToGroup is used).
//
// Attributes and key-value pairs like request-id, trace-id, user-id, etc, can
// be added to the context, and the *yasctx.Handler will make sure they are added
// to any log lines using that context.
func main() {
	// Create the *yasctx.Handler middleware
	h := yasctx.NewHandler(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(slog.New(h))

	ctx := context.Background()

	// Add some slog attributes to the start of future log lines:
	ctx = yasctx.Add(ctx, "key1", "value1")

	// Add some slog attributes to a specific group:
	ctx = yasctx.AddToGroup(ctx, "group1", slog.String("key2", "value2"))

	// Append some slog attributes to a group that won't be used will show up at the root level:
	ctx = yasctx.AddToGroup(ctx, "group2", slog.String("key3", "value3"))

	// Use the logger like normal:
	slog.Default().WithGroup("group1").WarnContext(ctx, "main message", "mainKey", "mainValue")
	/*
		{
		    "time": "2025-06-26T23:08:13.808079237-06:00",
		    "level": "WARN",
		    "msg": "main message",
		    "key1": "value1",
		    "key3": "value3",
		    "group1": {
		        "key2": "value2",
		        "mainKey": "mainValue"
		    }
		}
	*/
}

```

### Attributes propagated to parent contexes 
```go
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
```

### slog-multi Middleware
This library has a convenience method that allow it to interoperate with [github.com/samber/slog-multi](https://github.com/samber/slog-multi),
in order to easily setup slog workflows such as pipelines, fanout, routing, failover, etc.
```go
slog.SetDefault(slog.New(slogmulti.
	Pipe(yasctx.NewMiddleware()).
	Pipe(slogdedup.NewOverwriteMiddleware(&slogdedup.OverwriteHandlerOptions{})).
	Handler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})),
))
```
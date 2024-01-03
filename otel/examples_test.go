package slogotel_test

import (
	"log/slog"
	"os"

	slogctx "github.com/veqryn/slog-context"
	slogotel "github.com/veqryn/slog-context/otel"
)

func ExampleExtractTraceSpanID() {
	// Create the *slogctx.Handler middleware
	h := slogctx.NewHandler(
		slog.NewJSONHandler(os.Stdout, nil), // The next handler in the chain
		&slogctx.HandlerOptions{
			// Prependers will first add the OTEL Trace ID, then anything else Prepended to the ctx
			Prependers: []slogctx.AttrExtractor{
				slogotel.ExtractTraceSpanID,
				slogctx.ExtractPrepended,
			},
			// Appenders stays as default (leaving as nil would accomplish the same)
			Appenders: []slogctx.AttrExtractor{
				slogctx.ExtractAppended,
			},
		},
	)
	slog.SetDefault(slog.New(h))
}

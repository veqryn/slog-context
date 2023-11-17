package slogotel_test

import (
	"log/slog"
	"os"

	slogcontext "github.com/veqryn/slog-context"
	slogotel "github.com/veqryn/slog-context/otel"
)

func ExampleExtractTraceSpanID() {
	// Create the *slogcontext.Handler middleware
	h := slogcontext.NewHandler(
		slog.NewJSONHandler(os.Stdout, nil), // The next handler in the chain
		&slogcontext.HandlerOptions{
			// Prependers will first add the OTEL Trace ID, then anything else Prepended to the ctx
			Prependers: []slogcontext.AttrExtractor{
				slogotel.ExtractTraceSpanID,
				slogcontext.ExtractPrepended,
			},
			// Appenders stays as default (leaving as nil would accomplish the same)
			Appenders: []slogcontext.AttrExtractor{
				slogcontext.ExtractAppended,
			},
		},
	)
	slog.SetDefault(slog.New(h))
}

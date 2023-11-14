package slogcontext_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	slogcontext "github.com/veqryn/slog-context"
)

func TestExampleHandler(t *testing.T) {
	t.Parallel()

	h := slogcontext.NewHandler(slog.NewJSONHandler(os.Stdout, nil), nil)
	slog.SetDefault(slog.New(h))

	ctx := context.Background()

	// Prepend some slog attributes to the start of future log lines:
	ctx = slogcontext.Prepend(ctx, "prependKey", "prependValue")

	// Append some slog attributes to the end of future log lines:
	ctx = slogcontext.Append(ctx, "appendKey", "appendValue")

	log := slog.With("rootKey", "rootValue")
	log = log.WithGroup("someGroup")
	log = log.With("subKey", "subValue")

	log.InfoContext(ctx, "main message", "mainKey", "mainValue")
	/*
		{
			"time": "2023-11-14T00:37:03.805196-07:00",
			"level": "INFO",
			"msg": "main message",
			"prependKey": "prependValue",
			"rootKey": "rootValue",
			"someGroup": {
				"subKey": "subValue",
				"mainKey": "mainValue",
				"appendKey": "appendValue"
			}
		}
	*/
}

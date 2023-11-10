package slogcontext

import (
	"context"
	"log/slog"
	"testing"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	tester := &testHandler{}
	h := NewHandler(tester, nil)

	ctx := context.Background()
	ctx = Prepend(ctx, "prepend1", "arg1", "prepend1", "arg2")
	ctx = Prepend(ctx, "prepend2", "arg1", "prepend2", "arg2")
	ctx = Append(ctx, "append1", "arg1", "append1", "arg2")
	ctx = Append(ctx, "append2", "arg1", "append2", "arg2")

	log := slog.New(h)

	log = log.With("with1", "arg1", "with1", "arg2")
	log = log.WithGroup("group1")
	log = log.With("with2", "arg1", "with2", "arg2")

	log.InfoContext(ctx, "main message", "main1", "arg1", "main1", "arg2")

	t.Log(tester.String())
	// time=2023-09-29T13:00:59.000Z level=INFO msg="main message" prepend1=arg1 prepend1=arg2 prepend2=arg1 prepend2=arg2 with1=arg1 with1=arg2 group1.with2=arg1 group1.with2=arg2 group1.main1=arg1 group1.main1=arg2 group1.append1=arg1 group1.append1=arg2 group1.append2=arg1 group1.append2=arg2

	b, err := tester.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))
	// {"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"main message","prepend1":"arg1","prepend1":"arg2","prepend2":"arg1","prepend2":"arg2","with1":"arg1","with1":"arg2","group1":{"with2":"arg1","with2":"arg2","main1":"arg1","main1":"arg2","append1":"arg1","append1":"arg2","append2":"arg1","append2":"arg2"}}
}

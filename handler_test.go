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
	Prepend(ctx, "prepend3", "arg1", "prepend3", "arg2") // Ensure we aren't overwriting the parent context
	ctx = Append(ctx, "append1", "arg1", "append1", "arg2")
	ctx = Append(ctx, "append2", "arg1", "append2", "arg2")
	Append(ctx, "append3", "arg1", "append3", "arg2") // Ensure we aren't overwriting the parent context

	l := slog.New(h)

	l = l.With("with1", "arg1", "with1", "arg2")
	l = l.WithGroup("group1")
	l = l.With("with2", "arg1", "with2", "arg2")

	l.InfoContext(ctx, "main message", "main1", "arg1", "main1", "arg2")

	expectedText := `time=2023-09-29T13:00:59.000Z level=INFO msg="main message" prepend1=arg1 prepend1=arg2 prepend2=arg1 prepend2=arg2 with1=arg1 with1=arg2 group1.with2=arg1 group1.with2=arg2 group1.main1=arg1 group1.main1=arg2 group1.append1=arg1 group1.append1=arg2 group1.append2=arg1 group1.append2=arg2
`
	if s := tester.String(); s != expectedText {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedText, s)
	}

	b, err := tester.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	expectedJSON := `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"main message","prepend1":"arg1","prepend1":"arg2","prepend2":"arg1","prepend2":"arg2","with1":"arg1","with1":"arg2","group1":{"with2":"arg1","with2":"arg2","main1":"arg1","main1":"arg2","append1":"arg1","append1":"arg2","append2":"arg1","append2":"arg2"}}
`
	if string(b) != expectedJSON {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedText, string(b))
	}
}

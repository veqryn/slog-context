package yasctx

import (
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/pazams/yasctx/internal/test"
)

type logLine struct {
	Source struct {
		Function string `json:"function"`
		File     string `json:"file"`
		Line     int    `json:"line"`
	} `json:"source"`
}

func TestHandler(t *testing.T) {
	t.Parallel()

	tester := &test.Handler{}
	h := NewHandler(tester)

	ctx := Add(nil, "prepend1", "arg1", slog.String("prepend1", "arg2"))
	ctx = Add(ctx, "prepend2", "arg1", "prepend2", "arg2")
	Add(ctx, "prepend3", "arg1", "prepend3", "arg2") // Ensure we aren't overwriting the parent context
	ctx = AddToGroup(ctx, "group2", "prependGroupFound", "arg1", "prependGroupFound", "arg2")
	ctx = AddToGroup(ctx, "group3", "prependGroupNotFound", "arg1", "prependGroupNotFound", "arg2") // Ensure attrs on missing group get added to root level

	l := slog.New(h)

	l = l.With("with1", "arg1", "with1", "arg2").With()
	l = l.WithGroup("group1").WithGroup("")
	l = l.With("with2", "arg1", "with2", "arg2")
	l = l.WithGroup("group2").WithGroup("group2")
	l = l.With("with3", "arg1", "with3", "arg2")

	l.InfoContext(ctx, "main message", "main1", "arg1", "main1", "arg2")

	expectedText := `time=2023-09-29T13:00:59.000Z level=INFO msg="main message" prepend1=arg1 prepend1=arg2 prepend2=arg1 prepend2=arg2 prependGroupNotFound=arg1 prependGroupNotFound=arg2 with1=arg1 with1=arg2 group1.with2=arg1 group1.with2=arg2 group1.group2.group2.prependGroupFound=arg1 group1.group2.group2.prependGroupFound=arg2 group1.group2.group2.with3=arg1 group1.group2.group2.with3=arg2 group1.group2.group2.main1=arg1 group1.group2.group2.main1=arg2
`
	if s := tester.String(); s != expectedText {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedText, s)
	}

	b, err := tester.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	expectedJSON := `{"time":"2023-09-29T13:00:59Z","level":"INFO","msg":"main message","prepend1":"arg1","prepend1":"arg2","prepend2":"arg1","prepend2":"arg2","prependGroupNotFound":"arg1","prependGroupNotFound":"arg2","with1":"arg1","with1":"arg2","group1":{"with2":"arg1","with2":"arg2","group2":{"group2":{"prependGroupFound":"arg1","prependGroupFound":"arg2","with3":"arg1","with3":"arg2","main1":"arg1","main1":"arg2"}}}}
`
	if string(b) != expectedJSON {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedText, string(b))
	}

	// Check the source location fields
	tester.Source = true
	b, err = tester.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	var unmarshalled logLine
	err = json.Unmarshal(b, &unmarshalled)
	if err != nil {
		t.Fatal(err)
	}

	if unmarshalled.Source.Function != "github.com/pazams/yasctx.TestHandler" ||
		!strings.HasSuffix(unmarshalled.Source.File, "slog-context/handler_test.go") ||
		unmarshalled.Source.Line != 40 {
		t.Errorf("Expected source fields are incorrect: %#+v\n", unmarshalled)
	}
}

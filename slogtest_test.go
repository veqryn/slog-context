package slogctx_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"testing/slogtest"

	slogctx "github.com/veqryn/slog-context"
)

func TestSlogtest(t *testing.T) {
	var buf bytes.Buffer
	h := slogctx.NewHandler(slog.NewJSONHandler(&buf, nil), nil)

	results := func() []map[string]any {
		ms, err := parseLines(buf.Bytes(), parseJSON)
		if err != nil {
			t.Fatal(err)
		}
		return ms
	}
	if err := slogtest.TestHandler(h, results); err != nil {
		t.Fatal(err)
	}
}

func parseLines(src []byte, parse func([]byte) (map[string]any, error)) ([]map[string]any, error) {
	fmt.Println(string(src))
	var records []map[string]any
	for _, line := range bytes.Split(src, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		m, err := parse(line)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", string(line), err)
		}
		records = append(records, m)
	}
	return records, nil
}

func parseJSON(bs []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(bs, &m); err != nil {
		return nil, err
	}
	return m, nil
}

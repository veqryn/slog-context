package slogcontext

import (
	"log/slog"
	"testing"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	tester := &testHandler{}
	h := NewHandler(tester, nil)

	log := slog.New(h)

	log = log.With("with1", "arg0")
	log = log.WithGroup("group1")
	log = log.With("with2", "arg0")

	log.Info("main message", "main1", "arg0")

	t.Log(tester.String())
}

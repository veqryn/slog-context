package slogctx

import (
	"context"
	"errors"
	"testing"
)

func TestPanic1(t *testing.T) {
	t.Parallel()

	defer func() {
		recovered := recover()
		if recovered != "hello world" {
			t.Error("Received:", recovered)
		}
	}()

	Panic(context.Background(), "hello world")
}

func TestPanic2(t *testing.T) {
	t.Parallel()

	defer func() {
		recovered := recover()
		if recovered.(error).Error() != "oh no!" {
			t.Error("Received:", recovered)
		}
	}()

	Panic(context.Background(), "hello world", Err(errors.New("oh no!")))
}

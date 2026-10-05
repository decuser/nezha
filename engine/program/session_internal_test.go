package program

import (
	"errors"
	"testing"
	"time"

	"nezha/engine/runtime/hal/substrate"
)

func TestSessionInterruptUnwindsFunctionStack(t *testing.T) {
	s, err := NewSession(substrate.NewGoRuntime())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Eval("defs.nez", "let spin = () {\n\twhile true {\n\t}\n}\n"); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := s.Eval("call.nez", "spin()\n")
		done <- err
	}()

	deadline := time.Now().Add(2 * time.Second)
	for !s.Interrupt() {
		if time.Now().After(deadline) {
			t.Fatal("session never entered running evaluation")
		}
		time.Sleep(time.Millisecond)
	}

	select {
	case err := <-done:
		if !errors.Is(err, ErrInterrupted) {
			t.Fatalf("Eval error = %v, want ErrInterrupted", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("interrupted function evaluation did not return")
	}

	if got := s.userEnv.StackDepth(); got != 0 {
		t.Fatalf("stack depth after interrupt = %d, want 0", got)
	}
}

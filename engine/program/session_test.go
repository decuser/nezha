package program_test

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"nezha/engine/program"
	"nezha/engine/runtime/hal/substrate"
)

func evalInspect(t *testing.T, s *program.Session, filename, source string) string {
	t.Helper()
	v, err := s.Eval(filename, source)
	if err != nil {
		t.Fatal(err)
	}
	return v.Inspect()
}

func TestSessionPersistsBindingsAndAssignment(t *testing.T) {
	s, err := program.NewSession(substrate.NewGoRuntime())
	if err != nil {
		t.Fatal(err)
	}

	evalInspect(t, s, "one.nez", "let x = 40\n")
	if got := evalInspect(t, s, "two.nez", "x + 2\n"); got != "42" {
		t.Fatalf("x + 2 = %s, want 42", got)
	}

	evalInspect(t, s, "three.nez", "x = x + 1\n")
	if got := evalInspect(t, s, "four.nez", "x\n"); got != "41" {
		t.Fatalf("x after assignment = %s, want 41", got)
	}
}

func TestSessionPersistsFunctionsAndClosures(t *testing.T) {
	s, err := program.NewSession(substrate.NewGoRuntime())
	if err != nil {
		t.Fatal(err)
	}

	evalInspect(t, s, "defs.nez", "let make_adder = (base) {\n\treturn (n) {\n\t\treturn base + n\n\t}\n}\nlet add_10 = make_adder(10)\n")
	if got := evalInspect(t, s, "use.nez", "add_10(5)\n"); got != "15" {
		t.Fatalf("closure result = %s, want 15", got)
	}
}

func TestSessionResetDropsUserStateAndRetainsPrelude(t *testing.T) {
	s, err := program.NewSession(substrate.NewGoRuntime())
	if err != nil {
		t.Fatal(err)
	}

	evalInspect(t, s, "before.nez", "let x = 7\n")
	if err := s.Reset(); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Eval("after.nez", "x\n"); err == nil || !strings.Contains(err.Error(), "undefined: x") {
		t.Fatalf("x after reset error = %v, want undefined x", err)
	}
	if got := evalInspect(t, s, "prelude.nez", "length([1, 2, 3])\n"); got != "3" {
		t.Fatalf("length after reset = %s, want 3", got)
	}
}

func TestSessionsDoNotShareUserBindings(t *testing.T) {
	rt := substrate.NewGoRuntime()
	left, err := program.NewSession(rt)
	if err != nil {
		t.Fatal(err)
	}
	right, err := program.NewSession(rt)
	if err != nil {
		t.Fatal(err)
	}

	evalInspect(t, left, "left.nez", "let private = 9\n")
	if _, err := right.Eval("right.nez", "private\n"); err == nil || !strings.Contains(err.Error(), "undefined: private") {
		t.Fatalf("second session error = %v, want undefined private", err)
	}
}

func TestSessionUserScopeHasNoRawPrimitiveAuthority(t *testing.T) {
	s, err := program.NewSession(substrate.NewGoRuntime())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.Eval("authority.nez", "_print(\"no\")\n"); err == nil || !strings.Contains(err.Error(), "undefined: _print") {
		t.Fatalf("raw primitive error = %v, want undefined _print", err)
	}
	if got := evalInspect(t, s, "ordinary.nez", "equal(1, 1)\n"); got != "true" {
		t.Fatalf("prelude equal = %s, want true", got)
	}
}

func TestSessionDiagnosticsFollowCurrentSubmission(t *testing.T) {
	s, err := program.NewSession(substrate.NewGoRuntime())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Eval("first.nez", "missing_one\n")
	if err == nil || !strings.Contains(err.Error(), "first.nez") || !strings.Contains(err.Error(), "missing_one") {
		t.Fatalf("first diagnostic = %v", err)
	}

	_, err = s.Eval("second.nez", "missing_two\n")
	if err == nil || !strings.Contains(err.Error(), "second.nez") || !strings.Contains(err.Error(), "missing_two") {
		t.Fatalf("second diagnostic = %v", err)
	}
	if strings.Contains(err.Error(), "first.nez") {
		t.Fatalf("second diagnostic retained prior filename: %v", err)
	}
}

func TestSessionInterruptStopsCurrentEvaluationAndSessionRemainsUsable(t *testing.T) {
	s, err := program.NewSession(substrate.NewGoRuntime())
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := s.Eval("loop.nez", "while true {\n}\n")
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
		if !errors.Is(err, program.ErrInterrupted) {
			t.Fatalf("Eval error = %v, want ErrInterrupted", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("interrupted evaluation did not return")
	}

	if got := evalInspect(t, s, "after.nez", "1 + 1\n"); got != "2" {
		t.Fatalf("post-interrupt result = %s, want 2", got)
	}
	if s.Interrupt() {
		t.Fatal("Interrupt reported an active evaluation while idle")
	}
}

type firstWriteSignal struct {
	once sync.Once
	ch   chan struct{}
}

func (w *firstWriteSignal) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.ch) })
	return len(p), nil
}

var _ io.Writer = (*firstWriteSignal)(nil)

func TestSessionInterruptDoesNotRollBackCompletedEffects(t *testing.T) {
	w := &firstWriteSignal{ch: make(chan struct{})}
	rt := substrate.NewGoRuntime()
	rt.SetOutput(w)
	s, err := program.NewSession(rt)
	if err != nil {
		t.Fatal(err)
	}

	evalInspect(t, s, "setup.nez", "let i = 0\n")
	done := make(chan error, 1)
	go func() {
		_, err := s.Eval("mutate.nez", "while true {\n\ti = i + 1\n\tprintln(i)\n}\n")
		done <- err
	}()

	select {
	case <-w.ch:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not produce its first completed output effect")
	}

	if !s.Interrupt() {
		t.Fatal("Interrupt reported no active evaluation")
	}
	select {
	case err := <-done:
		if !errors.Is(err, program.ErrInterrupted) {
			t.Fatalf("Eval error = %v, want ErrInterrupted", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("interrupted mutating evaluation did not return")
	}

	got := evalInspect(t, s, "inspect.nez", "i\n")
	n, err := strconv.Atoi(got)
	if err != nil {
		t.Fatalf("i = %q, want integer", got)
	}
	if n < 1 {
		t.Fatalf("i = %d after observed loop effect, want >= 1", n)
	}
}

func TestIdleInterruptDoesNotPoisonNextEvaluation(t *testing.T) {
	s, err := program.NewSession(substrate.NewGoRuntime())
	if err != nil {
		t.Fatal(err)
	}
	if s.Interrupt() {
		t.Fatal("idle Interrupt returned true")
	}
	if got := evalInspect(t, s, "next.nez", "6 * 7\n"); got != "42" {
		t.Fatalf("next evaluation = %s, want 42", got)
	}
}

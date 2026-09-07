package program_test

import (
	"bytes"
	"testing"

	"nezha/engine/program"
	"nezha/engine/runtime/hal/substrate"
)

func TestGoRuntimeSatisfiesProgramRuntime(t *testing.T) {
	var _ program.Runtime = substrate.NewGoRuntime()
}

func TestEvalExprUsesCanonicalPrelude(t *testing.T) {
	rt := substrate.NewGoRuntime()
	got, err := program.EvalExpr(rt, "length([1, 2, 3])")
	if err != nil {
		t.Fatal(err)
	}
	if got != "3" {
		t.Fatalf("got %q, want %q", got, "3")
	}
}

func TestRunSourceUsesRuntimeOutput(t *testing.T) {
	var out bytes.Buffer
	rt := substrate.NewGoRuntime()
	rt.SetOutput(&out)

	if err := program.RunSource(rt, "test.nez", "println(\"hello\")\n"); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "hello\n" {
		t.Fatalf("got %q, want %q", got, "hello\\n")
	}
}

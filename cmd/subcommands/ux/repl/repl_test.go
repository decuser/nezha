package repl

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"nezha/engine/program"
	"nezha/engine/runtime/hal/substrate"
)

type scriptItem struct {
	line string
	err  error
}

type scriptedReader struct {
	items []scriptItem
	next  int
}

func newScriptedReader(items ...scriptItem) *scriptedReader {
	return &scriptedReader{items: items}
}

func (r *scriptedReader) Prompt() (string, error) {
	if r.next >= len(r.items) {
		return "", io.EOF
	}
	item := r.items[r.next]
	r.next++
	return item.line, item.err
}

func newTestSession(t *testing.T, out *trackingWriter) *program.Session {
	t.Helper()
	rt := substrate.NewGoRuntime()
	rt.SetOutput(out)
	s, err := program.NewSession(rt)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestREPLPersistsStateAcrossSubmissions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	programOut := newTrackingWriter(&stdout)
	s := newTestSession(t, programOut)
	reader := newScriptedReader(
		scriptItem{line: "let x = 40"},
		scriptItem{line: "x + 2"},
	)

	if code := run(reader, &stdout, &stderr, programOut, s, nil); code != 0 {
		t.Fatalf("run code = %d, stderr = %q", code, stderr.String())
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
	if got := stdout.String(); !strings.Contains(got, "42\n") {
		t.Fatalf("stdout = %q, want expression result 42", got)
	}
}

func TestREPLAcceptsMultilineOrdinaryNezha(t *testing.T) {
	var stdout, stderr bytes.Buffer
	programOut := newTrackingWriter(&stdout)
	s := newTestSession(t, programOut)
	reader := newScriptedReader(
		scriptItem{line: "let twice = (x) {"},
		scriptItem{line: "return x + x"},
		scriptItem{line: "}"},
		scriptItem{line: "twice(21)"},
	)

	if code := run(reader, &stdout, &stderr, programOut, s, nil); code != 0 {
		t.Fatalf("run code = %d, stderr = %q", code, stderr.String())
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
	if got := stdout.String(); !strings.Contains(got, "42\n") {
		t.Fatalf("stdout = %q, want function result 42", got)
	}
	if got := stdout.String(); !strings.Contains(got, promptCont) {
		t.Fatalf("stdout = %q, want continuation prompt", got)
	}
}

type interruptOnFirstWrite struct {
	out        *bytes.Buffer
	interrupts chan os.Signal
	once       sync.Once
}

func (w *interruptOnFirstWrite) Write(p []byte) (int, error) {
	n, err := w.out.Write(p)
	w.once.Do(func() { w.interrupts <- os.Interrupt })
	return n, err
}

func TestEvaluateStreamsOutputAndInterruptsCurrentEvaluation(t *testing.T) {
	interrupts := make(chan os.Signal, 1)
	var output bytes.Buffer
	writer := &interruptOnFirstWrite{out: &output, interrupts: interrupts}
	rt := substrate.NewGoRuntime()
	rt.SetOutput(writer)
	s, err := program.NewSession(rt)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := evaluate(s, "while true {\n\tprintln(\"live\")\n}\n", interrupts)
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, program.ErrInterrupted) {
			t.Fatalf("evaluate error = %v, want ErrInterrupted", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("interrupted REPL evaluation did not return")
	}
	if got := output.String(); !strings.Contains(got, "live\n") {
		t.Fatalf("streamed output = %q, want live output before interrupt", got)
	}

	v, err := s.Eval("<repl>", "6 * 7\n")
	if err != nil {
		t.Fatalf("session unusable after interrupt: %v", err)
	}
	if got := v.Inspect(); got != "42" {
		t.Fatalf("post-interrupt evaluation = %s, want 42", got)
	}
}

func TestREPLPrintWithoutNewlineDoesNotConsumePromptLine(t *testing.T) {
	var stdout, stderr bytes.Buffer
	programOut := newTrackingWriter(&stdout)
	s := newTestSession(t, programOut)
	reader := newScriptedReader(scriptItem{line: `print("x")`})

	if code := run(reader, &stdout, &stderr, programOut, s, nil); code != 0 {
		t.Fatalf("run code = %d, stderr = %q", code, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "x\n> ") {
		t.Fatalf("stdout = %q, want prompt on line after print output", got)
	}
}

func TestCompleteness(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"1 + 1", true},
		{"if true {", false},
		{"if true {\nprintln(1)\n}", true},
		{`println("{")`, true},
		{"# {\n1", true},
		{"[1, 2", false},
		{"equal(1,", false},
		{"\"unterminated", false},
	}
	for _, tc := range cases {
		if got := isComplete(tc.input); got != tc.want {
			t.Errorf("isComplete(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestBlank(t *testing.T) {
	for _, input := range []string{"", "   ", "# comment", "\n # comment\n"} {
		if !isBlank(input) {
			t.Errorf("isBlank(%q) = false, want true", input)
		}
	}
	if isBlank("1 # comment") {
		t.Fatal("ordinary source classified as blank")
	}
}

func TestREPLAndProgramReadShareOneInputStream(t *testing.T) {
	input := newSharedInput(strings.NewReader("let name = read()\nMax\nprintln(name)\n"))
	var stdout, stderr bytes.Buffer
	programOut := newTrackingWriter(&stdout)
	rt := substrate.NewGoRuntime()
	rt.SetInput(input)
	rt.SetOutput(programOut)
	s, err := program.NewSession(rt)
	if err != nil {
		t.Fatal(err)
	}

	reader := &terminalReader{input: input}
	if code := run(reader, &stdout, &stderr, programOut, s, nil); code != 0 {
		t.Fatalf("run code = %d, stderr = %q", code, stderr.String())
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
	if got := stdout.String(); !strings.Contains(got, "Max\n") {
		t.Fatalf("stdout = %q, want program read/println to consume Max", got)
	}
}

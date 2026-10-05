package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func startAndWait(t *testing.T, app *App, source, input string) runResponse {
	t.Helper()

	done := make(chan runResponse, 1)
	app.completionSink = func(result runResponse) {
		done <- result
	}

	start := app.Start(source, input)
	if start.Error != "" {
		t.Fatalf("Start: %s", start.Error)
	}

	select {
	case result := <-done:
		return result
	case <-time.After(2 * time.Second):
		t.Fatal("run did not complete")
		return runResponse{}
	}
}

func captureRun(t *testing.T, source, input string) (string, runResponse) {
	t.Helper()

	var mu sync.Mutex
	var output strings.Builder
	app := &App{
		outputSink: func(chunk string) {
			mu.Lock()
			output.WriteString(chunk)
			mu.Unlock()
		},
	}

	result := startAndWait(t, app, source, input)

	mu.Lock()
	got := output.String()
	mu.Unlock()
	return got, result
}

func requireRunOK(t *testing.T, source, input, want string) {
	t.Helper()
	got, result := captureRun(t, source, input)
	if result.Error != "" {
		t.Fatal(result.Error)
	}
	if result.Stopped {
		t.Fatal("run unexpectedly stopped")
	}
	if got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunSourceLiteralPrints(t *testing.T) {
	requireRunOK(t, ""+
		"println(42)\n"+
		"println(3.14)\n"+
		"println(1/3)\n"+
		"println(\"hello\")\n"+
		"println('N')\n"+
		"println(:ready)\n"+
		"println(true)\n"+
		"println(false)\n", "",
		"42\n157/50\n1/3\nhello\n'N'\n:ready\ntrue\nfalse\n")
}

func TestRunSourceLetAndName(t *testing.T) {
	requireRunOK(t, "let answer = 42\nprintln(answer)\n", "", "42\n")
}

func TestRunSourcePrintAssignmentAndBinary(t *testing.T) {
	requireRunOK(t, ""+
		"let x = 40\n"+
		"x = x + 2\n"+
		"print(\"answer: \")\n"+
		"println(x)\n"+
		"println((1 + 2) * 3)\n"+
		"println(1 < 2 and true)\n", "",
		"answer: 42\n9\ntrue\n")
}

func TestRunSourceIf(t *testing.T) {
	requireRunOK(t, ""+
		"let x = 42\n"+
		"if x > 0 {\n"+
		"\tprintln(x)\n"+
		"}\n", "", "42\n")
}

func TestRunSourceEqualNotWhile(t *testing.T) {
	requireRunOK(t, ""+
		"let x = 0\n"+
		"while not equal(x, 2) {\n"+
		"\tprintln(x)\n"+
		"\tx = x + 1\n"+
		"}\n", "", "0\n1\n")
}

func TestRunSourceRead(t *testing.T) {
	requireRunOK(t, "let name = read()\nprintln(name)\n", "Ada\n", "Ada\n")
}

func TestStartReportsExecutionFault(t *testing.T) {
	_, got := captureRun(t, "x = 42\n", "")
	if got.Error == "" {
		t.Fatal("expected unbound assignment error")
	}
}

func TestDefaultInputOutputSample(t *testing.T) {
	requireRunOK(t, "let name = read()\nprint(\"hello, \")\nprintln(name)\n", "Max", "hello, Max\n")
}

func TestOutputStreamBatchesWrites(t *testing.T) {
	var mu sync.Mutex
	var chunks []string
	stream := newOutputStream(func(chunk string) {
		mu.Lock()
		chunks = append(chunks, chunk)
		mu.Unlock()
	})

	for i := 0; i < 1000; i++ {
		if _, err := stream.Write([]byte("x\n")); err != nil {
			t.Fatal(err)
		}
	}
	stream.Close()

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		count := len(chunks)
		total := strings.Join(chunks, "")
		mu.Unlock()

		if len(total) == 2000 {
			if count >= 1000 {
				t.Fatalf("output was not batched: %d chunks for 1000 writes", count)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("batched output incomplete: got %d bytes, want 2000", len(total))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestStartReturnsBeforeLongRunCompletes(t *testing.T) {
	firstOutput := make(chan struct{})
	var once sync.Once
	app := &App{outputSink: func(string) { once.Do(func() { close(firstOutput) }) }}
	app.completionSink = func(runResponse) {}

	started := time.Now()
	result := app.Start("while true {\n\tprintln(\"tick\")\n}\n", "")
	if result.Error != "" {
		t.Fatal(result.Error)
	}
	if time.Since(started) > 250*time.Millisecond {
		t.Fatal("Start blocked on evaluation")
	}

	select {
	case <-firstOutput:
	case <-time.After(2 * time.Second):
		t.Fatal("no streamed output")
	}

	if !app.Stop() {
		t.Fatal("Stop did not interrupt active evaluation")
	}
}

func TestStreamStopAndReuse(t *testing.T) {
	firstOutput := make(chan struct{})
	done := make(chan runResponse, 1)
	var once sync.Once

	app := &App{
		outputSink: func(string) {
			once.Do(func() { close(firstOutput) })
		},
		completionSink: func(result runResponse) {
			done <- result
		},
	}

	start := app.Start("while true {\n\tprintln(\"tick\")\n}\n", "")
	if start.Error != "" {
		t.Fatal(start.Error)
	}

	select {
	case <-firstOutput:
	case <-time.After(2 * time.Second):
		t.Fatal("no streamed output before completion")
	}

	if !app.Stop() {
		t.Fatal("Stop did not interrupt active evaluation")
	}

	select {
	case result := <-done:
		if result.Error != "" {
			t.Fatalf("interrupted run error = %q", result.Error)
		}
		if !result.Stopped {
			t.Fatal("interrupted run was not reported as stopped")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("interrupted run did not complete")
	}

	var mu sync.Mutex
	var output strings.Builder
	done = make(chan runResponse, 1)
	app.outputSink = func(chunk string) {
		mu.Lock()
		output.WriteString(chunk)
		mu.Unlock()
	}
	app.completionSink = func(result runResponse) {
		done <- result
	}

	start = app.Start("println(42)\n", "")
	if start.Error != "" {
		t.Fatal(start.Error)
	}

	select {
	case result := <-done:
		if result.Error != "" || result.Stopped {
			t.Fatalf("reuse after stop = %+v", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("reuse run did not complete")
	}

	mu.Lock()
	got := output.String()
	mu.Unlock()
	if got != "42\n" {
		t.Fatalf("reuse output = %q, want %q", got, "42\\n")
	}
}

func TestDirectAppStopInterruptsInfiniteRun(t *testing.T) {
	firstOutput := make(chan struct{})
	done := make(chan runResponse, 1)
	var once sync.Once

	app := &App{
		outputSink: func(string) {
			once.Do(func() { close(firstOutput) })
		},
		completionSink: func(result runResponse) {
			done <- result
		},
	}

	start := app.Start("while true {\n\tprintln(\"tick\")\n}\n", "")
	if start.Error != "" {
		t.Fatal(start.Error)
	}

	select {
	case <-firstOutput:
	case <-time.After(2 * time.Second):
		t.Fatal("run produced no output")
	}

	if !app.Stop() {
		t.Fatal("App.Stop reported no active session")
	}

	select {
	case result := <-done:
		if !result.Stopped {
			t.Fatalf("result = %+v, want stopped", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("App.Stop did not end evaluation")
	}
}

func TestStopWhenIdle(t *testing.T) {
	if (&App{}).Stop() {
		t.Fatal("idle Stop reported an interruption")
	}
}

func TestConcurrentStartRejected(t *testing.T) {
	firstOutput := make(chan struct{})
	done := make(chan runResponse, 1)
	var once sync.Once

	app := &App{
		outputSink:     func(string) { once.Do(func() { close(firstOutput) }) },
		completionSink: func(result runResponse) { done <- result },
	}

	first := app.Start("while true {\n\tprintln(\"tick\")\n}\n", "")
	if first.Error != "" {
		t.Fatal(first.Error)
	}

	select {
	case <-firstOutput:
	case <-time.After(2 * time.Second):
		t.Fatal("first run did not start")
	}

	second := app.Start("println(1)\n", "")
	if second.Error != "a program is already running" {
		t.Fatalf("second start error = %q", second.Error)
	}

	if !app.Stop() {
		t.Fatal("failed to stop first run")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("first run did not stop")
	}
}

func TestDocumentTextRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "program.nezha-blocks")
	want := "{\n  \"format\": \"nezha-blockly\"\n}\n"
	if err := writeTextFile(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := readTextFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("document = %q, want %q", got, want)
	}
}

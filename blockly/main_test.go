package main

import (
	"path/filepath"
	"testing"
)

func TestRunSourceLiteralPrints(t *testing.T) {
	got, err := runSource(""+
		"println(42)\n"+
		"println(3.14)\n"+
		"println(1/3)\n"+
		"println(\"hello\")\n"+
		"println('N')\n"+
		"println(:ready)\n"+
		"println(true)\n"+
		"println(false)\n", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "42\n157/50\n1/3\nhello\n'N'\n:ready\ntrue\nfalse\n"
	if got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunSourceLetAndName(t *testing.T) {
	got, err := runSource("let answer = 42\nprintln(answer)\n", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "42\n" {
		t.Fatalf("output = %q, want %q", got, "42\n")
	}
}

func TestRunSourcePrintAssignmentAndBinary(t *testing.T) {
	got, err := runSource(""+
		"let x = 40\n"+
		"x = x + 2\n"+
		"print(\"answer: \")\n"+
		"println(x)\n"+
		"println((1 + 2) * 3)\n"+
		"println(1 < 2 and true)\n", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "answer: 42\n9\ntrue\n"
	if got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunSourceIf(t *testing.T) {
	got, err := runSource(""+
		"let x = 42\n"+
		"if x > 0 {\n"+
		"\tprintln(x)\n"+
		"}\n", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "42\n" {
		t.Fatalf("output = %q, want %q", got, "42\\n")
	}
}

func TestRunSourceEqualNotWhile(t *testing.T) {
	got, err := runSource(""+
		"let x = 0\n"+
		"while not equal(x, 2) {\n"+
		"\tprintln(x)\n"+
		"\tx = x + 1\n"+
		"}\n", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0\n1\n" {
		t.Fatalf("output = %q, want %q", got, "0\\n1\\n")
	}
}

func TestRunSourceRead(t *testing.T) {
	got, err := runSource("let name = read()\nprintln(name)\n", "Ada\n")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Ada\n" {
		t.Fatalf("output = %q, want %q", got, "Ada\n")
	}
}

func TestAppRunReportsExecutionFault(t *testing.T) {
	got := (&App{}).Run("x = 42\n", "")
	if got.Error == "" {
		t.Fatal("expected unbound assignment error")
	}
}

func TestDefaultInputOutputSample(t *testing.T) {
	got, err := runSource("let name = read()\nprint(\"hello, \")\nprintln(name)\n", "Max")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello, Max\n" {
		t.Fatalf("output = %q, want %q", got, "hello, Max\n")
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

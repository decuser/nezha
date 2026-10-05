package main

import "testing"

func TestRunSourceLiteralPrints(t *testing.T) {
	got, err := runSource("" +
		"println(42)\n" +
		"println(3.14)\n" +
		"println(1/3)\n" +
		"println(\"hello\")\n" +
		"println('N')\n" +
		"println(:ready)\n" +
		"println(true)\n" +
		"println(false)\n")
	if err != nil {
		t.Fatal(err)
	}
	want := "42\n157/50\n1/3\nhello\n'N'\n:ready\ntrue\nfalse\n"
	if got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

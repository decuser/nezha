package substrate

import (
	"testing"

	"nezha/engine/semantics/value"
)

func mathNumber(t *testing.T, text string) *value.Number {
	t.Helper()
	n, err := value.NewNumberFromString(text)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func mathWantNumber(t *testing.T, got value.Value, want string) {
	t.Helper()
	n, ok := got.(*value.Number)
	if !ok {
		t.Fatalf("expected number %s, got %T (%s)", want, got, got.Inspect())
	}
	expected := mathNumber(t, want)
	if !n.Equal(expected) {
		t.Fatalf("want %s, got %s", expected.Inspect(), n.Inspect())
	}
}

func TestModuloExactRationalEuclideanSemantics(t *testing.T) {
	cases := []struct {
		left  string
		right string
		want  string
	}{
		{"7/3", "2/3", "1/3"},
		{"-7/3", "2/3", "1/3"},
		{"7/3", "-2/3", "1/3"},
		{"-7/3", "-2/3", "1/3"},
		{"1/2", "3/2", "1/2"},
		{"4/3", "1/2", "1/3"},
	}
	for _, tc := range cases {
		t.Run(tc.left+" mod "+tc.right, func(t *testing.T) {
			got := halModulo([]value.Value{mathNumber(t, tc.left), mathNumber(t, tc.right)}, nil)
			mathWantNumber(t, got, tc.want)
		})
	}
}

func TestModuloExactRationalArbitraryPrecision(t *testing.T) {
	left := mathNumber(t, "92233720368547758081234567890123456789/97")
	right := mathNumber(t, "18446744073709551617000000000000000003/101")
	got := halModulo([]value.Value{left, right}, nil)

	// Exact expected remainder from a - floor(a / b) * b. This witnesses the
	// arbitrary-precision rational path without depending on host floating point.
	mathWantNumber(t, got, "368934881474191031959691356902469134234/9797")
}

func TestModuloDivisionByZeroUsesLanguageFault(t *testing.T) {
	got := halModulo([]value.Value{mathNumber(t, "10"), mathNumber(t, "0")}, nil)
	fault, ok := got.(*value.Fault)
	if !ok {
		t.Fatalf("expected fault, got %T (%s)", got, got.Inspect())
	}
	if fault.Message != "division by zero" {
		t.Fatalf("want language division-by-zero fault, got %q", fault.Message)
	}
}

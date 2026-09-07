package test

import "testing"

func TestCoverageScopeIsRepositoryPathPolicy(t *testing.T) {
	cases := map[string]string{
		"test/behavior/math_smoke.ai":             "auxiliary",
		"samples/minimal.ai":                      "examples",
		"examples/hello.ai":                       "examples",
		"engineering/profiling/03-list-append.ai": "auxiliary",
	}
	for path, want := range cases {
		if got := coverageScope(path); got != want {
			t.Fatalf("coverageScope(%q)=%q want %q", path, got, want)
		}
	}
}

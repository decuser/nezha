package test

import "testing"

func TestCoverageScopeIsRepositoryPathPolicy(t *testing.T) {
	cases := map[string]string{
		"test/behavior/math_smoke.nez":             "auxiliary",
		"samples/minimal.nez":                      "examples",
		"examples/hello.nez":                       "examples",
		"engineering/profiling/03-list-append.nez": "auxiliary",
	}
	for path, want := range cases {
		if got := coverageScope(path); got != want {
			t.Fatalf("coverageScope(%q)=%q want %q", path, got, want)
		}
	}
}

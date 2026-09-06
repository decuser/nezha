package test

import "testing"

func TestCoverageScopeIsRepositoryPathPolicy(t *testing.T) {
	cases := map[string]string{
		"lib/string/string.ai":                    "supported",
		"samples/pipeline.ai":                     "examples",
		"examples/hello.ai":                       "examples",
		"engineering/profiling/03-list-append.ai": "auxiliary",
	}
	for path, want := range cases {
		if got := coverageScope(path); got != want {
			t.Fatalf("coverageScope(%q)=%q want %q", path, got, want)
		}
	}
}

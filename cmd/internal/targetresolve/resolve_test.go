package targetresolve

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func makeTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "src", "sub"))
	mustMkdirAll(t, filepath.Join(root, "experiments", "demo"))
	mustWrite(t, filepath.Join(root, "src", "a.nez"), "let a = 1\n")
	mustWrite(t, filepath.Join(root, "src", "sub", "b.nez"), "let b = 2\n")
	mustWrite(t, filepath.Join(root, "experiments", "demo", "e_test.nez"), "let e = 3\n")
	mustWrite(t, filepath.Join(root, ".nezhaignore"), "[test]\nexperiments/**\n")
	return root
}

func TestRecursiveTargetsAreOrderedAndDeduplicated(t *testing.T) {
	root := makeTree(t)
	got, err := ResolveFrom(root, "test", []string{"src/...", "src/a.nez"}, true)
	if err != nil {
		t.Fatalf("ResolveFrom returned error: %v", err)
	}
	want := []string{"src/a.nez", "src/sub/b.nez"}
	assertStrings(t, got, want)
}

func TestPolicyPrunesExperimentsFromBroadRecursion(t *testing.T) {
	root := makeTree(t)
	got, err := ResolveFrom(root, "test", []string{"./..."}, true)
	if err != nil {
		t.Fatalf("ResolveFrom returned error: %v", err)
	}
	want := []string{".nezhaignore", "src/a.nez", "src/sub/b.nez"}
	assertStrings(t, got, want)
}

func TestExplicitRecursiveTargetWinsOverRepositoryPolicy(t *testing.T) {
	root := makeTree(t)
	got, err := ResolveFrom(root, "test", []string{"experiments/..."}, true)
	if err != nil {
		t.Fatalf("ResolveFrom returned error: %v", err)
	}
	want := []string{"experiments/demo/e_test.nez"}
	assertStrings(t, got, want)
}

func TestPlainDirectoryBehaviorIsCallerControlled(t *testing.T) {
	root := makeTree(t)
	got, err := ResolveFrom(root, "lint", []string{"src"}, true)
	if err != nil {
		t.Fatalf("ResolveFrom returned error: %v", err)
	}
	want := []string{"src/a.nez", "src/sub/b.nez"}
	assertStrings(t, got, want)

	_, err = ResolveFrom(root, "lint", []string{"src"}, false)
	if err == nil || !strings.Contains(err.Error(), "directory target requires recursion: src") {
		t.Fatalf("ResolveFrom plain directory error = %v, want directory recursion error", err)
	}
}

func TestExhaustiveResolutionBypassesRepositoryPolicy(t *testing.T) {
	root := makeTree(t)
	got, err := ResolveAllFrom(root, "test", []string{"./..."}, true)
	if err != nil {
		t.Fatalf("ResolveAllFrom returned error: %v", err)
	}
	want := []string{".nezhaignore", "experiments/demo/e_test.nez", "src/a.nez", "src/sub/b.nez"}
	assertStrings(t, got, want)
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
}

func mustWrite(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

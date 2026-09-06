package fmt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withDistributionRoot(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("cannot find distribution root at %s: %v", root, err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

func TestRunFailsOnUndeclaredMalformedFile(t *testing.T) {
	withDistributionRoot(t)
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.ai")
	good := filepath.Join(dir, "good.ai")
	if err := os.WriteFile(bad, []byte("let x =\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(good, []byte("let y = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{dir + "/..."}); code != 1 {
		t.Fatalf("Run exit = %d, want 1", code)
	}
}

func TestFormatPathSkipsDeclaredParseNegative(t *testing.T) {
	withDistributionRoot(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "negative_smoke.ai")
	if err := os.WriteFile(path, []byte("# @negative parse\nlet x =\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := FormatPath(dir+"/...", Config{}); err != nil {
		t.Fatalf("declared parse-negative was not skipped: %v", err)
	}
}

func TestFormatPathRejectsNegativeMarkerOutsideSmokeFixture(t *testing.T) {
	withDistributionRoot(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "ordinary.ai")
	if err := os.WriteFile(path, []byte("# @negative parse\nlet x =\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := FormatPath(dir+"/...", Config{}); err == nil {
		t.Fatal("ordinary source used @negative parse as a formatting exemption")
	}
}

func TestFormatDirReportsAllUndeclaredMalformedFiles(t *testing.T) {
	withDistributionRoot(t)
	dir := t.TempDir()
	for _, name := range []string{"a.ai", "b.ai"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("let x =\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := FormatPath(dir+"/...", Config{})
	if err == nil {
		t.Fatal("expected malformed files to fail formatting")
	}
	msg := err.Error()
	for _, name := range []string{"a.ai", "b.ai"} {
		if !strings.Contains(msg, name) {
			t.Fatalf("error %q does not mention %s", msg, name)
		}
	}
}

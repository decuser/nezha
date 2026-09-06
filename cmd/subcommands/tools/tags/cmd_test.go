package tags

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

func TestTagsWritesTopLevelDefinitions(t *testing.T) {
	withDistributionRoot(t)
	d := t.TempDir()
	p := filepath.Join(d, "sample.ai")
	if err := os.WriteFile(p, []byte("let outer = 1\nlet f = (x) { let y = x; return y }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(d, "tags")
	if code := Run([]string{"-o", out, p}); code != 0 {
		t.Fatalf("code=%d", code)
	}
	b, _ := os.ReadFile(out)
	s := string(b)
	if !strings.Contains(s, "outer\t"+p+"\t1;\"") || !strings.Contains(s, "f\t"+p+"\t2;\"") || strings.Contains(s, "\ny\t") {
		t.Fatalf("tags=%q", s)
	}
}

func TestExpandSupportsArbitraryRecursiveSubtree(t *testing.T) {
	withDistributionRoot(t)
	d := t.TempDir()
	sub := filepath.Join(d, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(sub, "x.ai")
	if err := os.WriteFile(p, []byte("let x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := expand([]string{sub + "/..."})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != p {
		t.Fatalf("expand=%v want [%s]", got, p)
	}
}

package invariant

import (
	"os"
	"path/filepath"
	"testing"
)

// distributionRoot locates the repository root relative to this test's package.
func distributionRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve distribution root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("cannot find distribution root at %s: %v", root, err)
	}
	return root
}

// shippedModulePaths reports distribution module sources. Nezha currently ships
// no library/module surface; scripts run with the prelude only.
func shippedModulePaths(t *testing.T) map[string]string {
	t.Helper()
	return map[string]string{}
}

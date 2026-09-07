package invariant

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNaturalStringOrderingUsesStringObservationAuthority(t *testing.T) {
	root := distributionRoot(t)
	path := filepath.Join(root, "engine", "semantics", "value", "order.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	if !strings.Contains(source, "return l.CompareRunes(r), true") {
		t.Fatalf("natural string ordering must delegate to String.CompareRunes")
	}
}

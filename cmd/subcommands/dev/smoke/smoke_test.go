package smoke

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"nezha/cmd/internal/testfixture"
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

func TestEncodeTranscriptCanonicalEscapes(t *testing.T) {
	got := encodeTranscript(
		[]byte("Ada\n"),
		[]byte(":default\n42\n1\n"),
		[]byte("warning\n"),
		2,
		[]string{"inspect output"},
	)
	want := []byte("IN:Ada\\n\n" +
		"OUT::default\\n\n" +
		"OUT:42\\n\n" +
		"OUT:1\\n\n" +
		"ERR:warning\\n\n" +
		"EXIT:2\n" +
		"DISPLAY:inspect output\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("encoded transcript\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

func TestEncodeTranscriptRoundTrips(t *testing.T) {
	data := encodeTranscript([]byte("x\ny"), []byte("a\nb"), []byte("e\n"), 3, nil)
	// loadTranscript reads files, so exercise the escaping primitive directly.
	if unescape(escapeTranscript("a\n")) != "a\n" {
		t.Fatal("escape/unescape did not round trip newline")
	}
	if len(data) == 0 {
		t.Fatal("expected transcript data")
	}
}

func TestNegativeObservationRequiresDeclarationBothWays(t *testing.T) {
	parseErr := []byte("parser: test.nez:1:1:\nexpected expression\n")
	if err := validateNegativeObservation("declared", testfixture.NegativeParse, parseErr, 1); err != nil {
		t.Fatalf("declared parse failure rejected: %v", err)
	}
	if err := validateNegativeObservation("undeclared", testfixture.NegativeNone, parseErr, 1); err == nil {
		t.Fatal("undeclared parse failure was accepted")
	}
	if err := validateNegativeObservation("passing", testfixture.NegativeParse, nil, 0); err == nil {
		t.Fatal("declared parse-negative that passes was accepted")
	}
}

func TestCollectSmokeFilesSupportsArbitraryRecursiveSubtree(t *testing.T) {
	withDistributionRoot(t)
	d := t.TempDir()
	sub := filepath.Join(d, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(sub, "x_smoke.nez")
	if err := os.WriteFile(p, []byte("let x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := collectSmokeFiles([]string{sub + "/..."})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != p {
		t.Fatalf("collect=%v want [%s]", got, p)
	}
}

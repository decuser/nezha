package invariant

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var (
	makeTargetDecl     = regexp.MustCompile(`(?m)^([A-Za-z0-9_-]+):(?:\s|$)`)
	documentedMakeCode = regexp.MustCompile("`make\\s+([A-Za-z0-9_-]+)(?:\\s|`)")
)

// TestDocumentedMakeTargetsExist keeps current-facing repository guidance from
// naming Make targets that have been removed or renamed. Frozen release notes
// and historical proposals are intentionally excluded: they describe the tree
// that existed when they were written rather than current operator instructions.
func TestDocumentedMakeTargetsExist(t *testing.T) {
	root := distributionRoot(t)
	makeSource, err := os.ReadFile(filepath.Join(root, "makefile"))
	if err != nil {
		t.Fatal(err)
	}
	available := map[string]bool{}
	for _, m := range makeTargetDecl.FindAllStringSubmatch(string(makeSource), -1) {
		available[m[1]] = true
	}

	var docs []string
	for _, rel := range []string{"README.md", "docs", "test", "engineering/README.md", "engineering/ai/README.md"} {
		path := filepath.Join(root, rel)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() {
			docs = append(docs, path)
			continue
		}
		err = filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			slash := filepath.ToSlash(path)
			if d.IsDir() {
				if strings.Contains(slash, "/engineering/proposals/") || strings.Contains(slash, "/docs/releases/") {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) == ".md" {
				docs = append(docs, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	sort.Strings(docs)
	for _, path := range docs {
		text, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		check := func(target string) {
			if !available[target] {
				rel, _ := filepath.Rel(root, path)
				t.Errorf("%s documents missing Make target %q", filepath.ToSlash(rel), target)
			}
		}
		for _, m := range documentedMakeCode.FindAllStringSubmatch(string(text), -1) {
			check(m[1])
		}
		inFence := false
		for _, line := range strings.Split(string(text), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") {
				inFence = !inFence
				continue
			}
			if !inFence || !strings.HasPrefix(trimmed, "make ") {
				continue
			}
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				check(fields[1])
			}
		}
	}
}

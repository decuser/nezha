package treecheck

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Finding struct {
	Path   string
	Reason string
}

type Result struct {
	Files     int
	Justified int
	Allowed   int
	Errors    []Finding
	Orphans   []Finding
}

type Checker struct {
	Root      string
	AllowFile string

	files     map[string]bool
	justified map[string]string
	allowed   map[string]string
	allow     []allowRule
}

type allowRule struct {
	raw    string
	prefix string
	glob   string
}

func Check(root, allowFile string) (Result, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return Result{}, err
	}
	c := &Checker{
		Root: abs, AllowFile: allowFile,
		files:     make(map[string]bool),
		justified: make(map[string]string),
		allowed:   make(map[string]string),
	}
	if err := c.loadFiles(); err != nil {
		return Result{}, err
	}
	if err := c.loadAllow(); err != nil {
		return Result{}, err
	}
	c.applyAllow()
	c.seedBuiltins()
	c.checkStructuralPairs()
	c.resolveTextReferences()
	c.resolveDirectoryReadmes()

	var result Result
	result.Files = len(c.files)
	result.Justified = len(c.justified)
	result.Allowed = len(c.allowed)
	result.Errors = append(result.Errors, c.structuralErrors()...)
	errorPaths := make(map[string]bool, len(result.Errors))
	for _, finding := range result.Errors {
		errorPaths[finding.Path] = true
	}

	for path := range c.files {
		if errorPaths[path] {
			continue
		}
		if _, ok := c.justified[path]; ok {
			continue
		}
		if _, ok := c.allowed[path]; ok {
			continue
		}
		result.Orphans = append(result.Orphans, Finding{Path: path, Reason: "no recognized distribution relationship"})
	}
	sortFindings(result.Errors)
	sortFindings(result.Orphans)
	return result, nil
}

func (c *Checker) loadFiles() error {
	cmd := exec.Command("git", "-C", c.Root, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	out, err := cmd.Output()
	if err == nil {
		for _, raw := range bytes.Split(out, []byte{0}) {
			if len(raw) == 0 {
				continue
			}
			p := filepath.ToSlash(string(raw))
			if p == "" {
				continue
			}
			// git ls-files reports tracked paths from the index even when the
			// working-tree file has been removed. Treecheck describes the current
			// distribution, so absent paths must not be treated as present files.
			if info, statErr := os.Stat(filepath.Join(c.Root, filepath.FromSlash(p))); statErr != nil || info.IsDir() {
				continue
			}
			c.files[p] = true
		}
		return nil
	}

	return filepath.WalkDir(c.Root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(c.Root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == ".git" || strings.HasPrefix(rel, ".git/") {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == "nezha" {
			return nil
		}
		c.files[rel] = true
		return nil
	})
}

func (c *Checker) loadAllow() error {
	p := filepath.Join(c.Root, c.AllowFile)
	f, err := os.Open(p)
	if err != nil {
		return fmt.Errorf("reading allow file %s: %w", c.AllowFile, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		s := strings.TrimSpace(sc.Text())
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		s = filepath.ToSlash(strings.TrimPrefix(s, "./"))
		r := allowRule{raw: s}
		if strings.HasSuffix(s, "/") {
			r.prefix = s
		} else {
			r.glob = s
		}
		c.allow = append(c.allow, r)
	}
	return sc.Err()
}

func (c *Checker) applyAllow() {
	for p := range c.files {
		for _, r := range c.allow {
			matched := false
			if r.prefix != "" {
				matched = strings.HasPrefix(p, r.prefix)
			} else if r.glob != "" {
				matched, _ = filepath.Match(r.glob, p)
			}
			if matched {
				c.allowed[p] = r.raw
				break
			}
		}
	}
}

func (c *Checker) mark(path, reason string) {
	if c.files[path] {
		c.justified[path] = reason
	}
}

func (c *Checker) seedBuiltins() {
	roots := []string{".gitignore", "LICENSE", "README.md", "engineering/README.md", "engineering/audit-findings.md", "go.mod", "go.sum", "makefile", c.AllowFile}
	for _, p := range roots {
		c.mark(p, "repository root artifact")
	}
	for p := range c.files {
		switch {
		case strings.HasSuffix(p, ".go"):
			c.mark(p, "Go implementation/test source")
		case filepath.Base(p) == ".gitignore":
			c.mark(p, "directory ignore policy")
		case p == "engine/runtime/prelude/prelude_test.nez":
			c.mark(p, "Nezha-native prelude test")
		case (strings.HasPrefix(p, "test/behavior/") || strings.HasPrefix(p, "test/visual/")) && strings.HasSuffix(p, "_smoke.nez"):
			c.mark(p, "smoke-test specimen")
		case strings.HasPrefix(p, "test/structure/engine/") && strings.HasSuffix(p, "_engine.nez"):
			c.mark(p, "engine structural specimen")
		case strings.HasPrefix(p, "test/nativeffi/") && strings.HasSuffix(p, "_contract.nez"):
			c.mark(p, "native/FFI semantic conformance specimen")
		case strings.HasPrefix(p, "docs/releases/") && strings.HasSuffix(p, ".md"):
			c.mark(p, "release note")
		case strings.HasPrefix(p, "docs/releases/") && strings.HasSuffix(p, ".odt"):
			c.mark(p, "frozen release document")
		case strings.HasPrefix(p, "reference/") && strings.HasSuffix(p, ".nez"):
			c.mark(p, "canonical language reference specimen")
		case strings.HasPrefix(p, "engineering/profiling/"):
			c.mark(p, "engineering profiling workload/evidence")
		case strings.HasPrefix(p, "engineering/evidence/"):
			c.mark(p, "retained engineering evidence")
		case p == "engineering/ai/README.md":
			c.mark(p, "AI-assisted engineering working method")
		case p == "engineering/proposals/README.md":
			c.mark(p, "proposal disposition guide")
		case strings.HasPrefix(p, "engineering/proposals/") && strings.HasSuffix(p, ".md"):
			c.mark(p, "engineering proposal record")
		case strings.HasPrefix(p, "engineering/git-hooks/"):
			c.mark(p, "repository Git-hook support")
		case p == "engineering/session-history.md" || p == "engineering/runtime-optimization-results.md":
			c.mark(p, "durable engineering record")
		case p == "engine/language/grammar/grammar.ebnfx" || p == "engine/language/grammar/grammar.help" || p == "engine/language/grammar/grammar.ebnfx.ebnf.gold":
			c.mark(p, "authoritative grammar artifact")
		case p == "engine/runtime/prelude/prelude.nez" || p == "engine/runtime/prelude/prelude.help" || p == "engine/runtime/prelude/prelude.doc":
			c.mark(p, "prelude artifact")
		}
	}
}

func matchedProfileDriver(p string) bool {
	base := filepath.Base(p)
	if len(base) < len("00-.nez") || !strings.HasSuffix(base, ".nez") {
		return false
	}
	return base[0] >= '0' && base[0] <= '9' && base[1] >= '0' && base[1] <= '9' && base[2] == '-'
}

func (c *Checker) checkStructuralPairs() {
	for p := range c.files {
		if (strings.HasPrefix(p, "test/behavior/") || strings.HasPrefix(p, "test/visual/")) && strings.HasSuffix(p, "_smoke.nez") {
			gold := strings.TrimSuffix(p, ".nez") + ".gold"
			if c.files[gold] {
				c.mark(gold, "smoke-test gold")
			}
		}
		if strings.HasPrefix(p, "test/structure/engine/") && strings.HasSuffix(p, "_engine.nez") {
			for _, stage := range []string{"lex", "parse", "eval"} {
				gold := p + "." + stage + ".gold"
				if c.files[gold] {
					c.mark(gold, "engine structural gold")
				}
			}
		}
		if strings.HasPrefix(p, "test/conformance/syntax/") && strings.HasSuffix(p, ".input") {
			projection := strings.TrimSuffix(p, ".input") + ".tokens"
			c.mark(p, "syntax conformance input")
			if c.files[projection] {
				c.mark(projection, "syntax conformance token projection")
			}
		}
	}
}

func isNamedPackageSource(path string) bool {
	return false
}

func hasProposalFiles(files map[string]bool) bool {
	for p := range files {
		if strings.HasPrefix(p, "engineering/proposals/") {
			return true
		}
	}
	return false
}

func (c *Checker) structuralErrors() []Finding {
	var out []Finding
	proposalDispositions := map[string]bool{
		"active":     true,
		"completed":  true,
		"superseded": true,
		"parked":     true,
	}
	for p := range c.files {
		if p == "engineering/proposals/README.md" || !strings.HasPrefix(p, "engineering/proposals/") {
			continue
		}
		parts := strings.Split(filepath.ToSlash(p), "/")
		if len(parts) < 4 {
			out = append(out, Finding{Path: p, Reason: "proposal must live under a disposition directory"})
			continue
		}
		if !proposalDispositions[parts[2]] {
			out = append(out, Finding{Path: p, Reason: "unknown proposal disposition " + parts[2]})
		}
	}
	if hasProposalFiles(c.files) && !c.files["engineering/proposals/README.md"] {
		out = append(out, Finding{Path: "engineering/proposals", Reason: "proposal disposition tree missing README.md"})
	}
	for p := range c.files {
		if (strings.HasPrefix(p, "test/behavior/") || strings.HasPrefix(p, "test/visual/")) && strings.HasSuffix(p, "_smoke.nez") {
			gold := strings.TrimSuffix(p, ".nez") + ".gold"
			if !c.files[gold] {
				out = append(out, Finding{Path: p, Reason: "smoke specimen missing .gold"})
			}
		}
		if (strings.HasPrefix(p, "test/behavior/") || strings.HasPrefix(p, "test/visual/")) && strings.HasSuffix(p, ".gold") {
			ai := strings.TrimSuffix(p, ".gold") + ".nez"
			if !c.files[ai] {
				out = append(out, Finding{Path: p, Reason: "smoke gold has no .nez specimen"})
			}
		}
		if strings.HasPrefix(p, "test/structure/engine/") && strings.Contains(p, "_engine.nez.") && strings.HasSuffix(p, ".gold") {
			i := strings.Index(p, ".nez.")
			ai := p[:i+4]
			if !c.files[ai] {
				out = append(out, Finding{Path: p, Reason: "engine gold has no .nez specimen"})
			}
		}
		if strings.HasPrefix(p, "test/structure/engine/") && strings.HasSuffix(p, "_engine.nez") {
			for _, stage := range []string{"lex", "parse", "eval"} {
				gold := p + "." + stage + ".gold"
				if !c.files[gold] {
					out = append(out, Finding{Path: p, Reason: "engine specimen missing " + stage + " gold"})
				}
			}
		}
		if strings.HasPrefix(p, "test/conformance/syntax/") && strings.HasSuffix(p, ".input") {
			projection := strings.TrimSuffix(p, ".input") + ".tokens"
			if !c.files[projection] {
				out = append(out, Finding{Path: p, Reason: "syntax conformance input missing .tokens projection"})
			}
		}
		if strings.HasPrefix(p, "test/conformance/syntax/") && strings.HasSuffix(p, ".tokens") {
			input := strings.TrimSuffix(p, ".tokens") + ".input"
			if !c.files[input] {
				out = append(out, Finding{Path: p, Reason: "syntax conformance projection has no .input specimen"})
			}
		}
	}
	return out
}

func (c *Checker) resolveTextReferences() {
	changed := true
	for changed {
		changed = false
		var sources []string
		for p := range c.justified {
			if isTextPath(p) {
				sources = append(sources, p)
			}
		}
		sort.Strings(sources)
		for _, src := range sources {
			data, err := os.ReadFile(filepath.Join(c.Root, filepath.FromSlash(src)))
			if err != nil {
				continue
			}
			text := string(data)
			for target := range c.files {
				if target == src {
					continue
				}
				if _, ok := c.justified[target]; ok {
					continue
				}
				if strings.Contains(text, target) {
					c.mark(target, "referenced by "+src)
					changed = true
				}
			}
		}
	}
}

func (c *Checker) resolveDirectoryReadmes() {
	changed := true
	for changed {
		changed = false
		for p := range c.files {
			if filepath.Base(p) != "README.md" {
				continue
			}
			if _, ok := c.justified[p]; ok {
				continue
			}
			dir := filepath.ToSlash(filepath.Dir(p))
			prefix := dir + "/"
			for q := range c.justified {
				if q != p && strings.HasPrefix(q, prefix) {
					c.mark(p, "README for active distribution subtree")
					changed = true
					break
				}
			}
		}
	}
}

func isTextPath(p string) bool {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".go", ".nez", ".md", ".help", ".doc", ".sh", ".lang", ".ebnfx":
		return true
	}
	return filepath.Base(p) == "makefile" || strings.HasPrefix(p, "engineering/git-hooks/")
}

func sortFindings(v []Finding) {
	sort.Slice(v, func(i, j int) bool {
		if v[i].Path == v[j].Path {
			return v[i].Reason < v[j].Reason
		}
		return v[i].Path < v[j].Path
	})
}

package targetresolve

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Resolve expands CLI filesystem targets into deterministic, repository-relative
// path lists for command consumers. It preserves the target language formerly
// owned by the command layer.
func Resolve(tool string, targets []string, recursePlainDirs bool) ([]string, error) {
	return resolve(tool, targets, recursePlainDirs, false)
}

// ResolveAll is the exhaustive form of Resolve. It preserves target syntax and
// command directory policy but bypasses .nezhaignore selection policy.
func ResolveAll(tool string, targets []string, recursePlainDirs bool) ([]string, error) {
	return resolve(tool, targets, recursePlainDirs, true)
}

func resolve(tool string, targets []string, recursePlainDirs, includeIgnored bool) ([]string, error) {
	base, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("current directory: %w", err)
	}
	return resolveFrom(base, tool, targets, recursePlainDirs, includeIgnored)
}

// ResolveFrom is Resolve with an explicit base, used by tests and callers that
// must not depend on process-current-directory interpretation.
func ResolveFrom(base, tool string, targets []string, recursePlainDirs bool) ([]string, error) {
	return resolveFrom(base, tool, targets, recursePlainDirs, false)
}

// ResolveAllFrom is ResolveAll with an explicit base.
func ResolveAllFrom(base, tool string, targets []string, recursePlainDirs bool) ([]string, error) {
	return resolveFrom(base, tool, targets, recursePlainDirs, true)
}

type ignorePolicy struct {
	file  string
	root  string
	rules []string
}

func resolveFrom(base, tool string, targets []string, recursePlainDirs, includeIgnored bool) ([]string, error) {
	base = filepath.Clean(base)
	policy, err := loadPolicy(base, tool)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	out := []string{}
	for _, original := range targets {
		recursive := isRecursiveTarget(original)
		named := original
		if recursive {
			named = recursiveRoot(original)
		}
		root := absolute(base, named)
		info, err := os.Stat(root)
		if err != nil {
			return nil, fmt.Errorf("[@error, :target, %q]", "cannot find target: "+original)
		}
		isDir := info.IsDir()
		bypassPolicy := includeIgnored || policy.ignored(root, isDir)
		if isDir && (recursive || recursePlainDirs) {
			files, err := walkFiles(root, policy, bypassPolicy)
			if err != nil {
				return nil, err
			}
			for _, file := range files {
				identity := filepath.Clean(file)
				if !seen[identity] {
					seen[identity] = true
					out = append(out, display(base, identity))
				}
			}
			continue
		}
		if isDir {
			return nil, fmt.Errorf("[@error, :target, %q]", "directory target requires recursion: "+original)
		}
		identity := filepath.Clean(root)
		if !seen[identity] {
			seen[identity] = true
			out = append(out, display(base, identity))
		}
	}
	sort.Strings(out)
	return out, nil
}

func absolute(base, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(base, p))
}

func display(base, p string) string {
	b := filepath.Clean(base)
	q := filepath.Clean(p)
	if q == b {
		return "."
	}
	if rel, err := filepath.Rel(b, q); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
		return rel
	}
	return q
}

func isRecursiveTarget(target string) bool {
	return target == "..." || strings.HasSuffix(target, "/...") || strings.HasSuffix(target, string(filepath.Separator)+"...")
}

func recursiveRoot(target string) string {
	if target == "..." {
		return "."
	}
	n := len(target) - 4
	if n <= 0 {
		return "."
	}
	return target[:n]
}

func loadPolicy(base, tool string) (ignorePolicy, error) {
	file := findPolicyFile(base)
	if file == "" {
		return ignorePolicy{}, nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return ignorePolicy{}, err
	}
	section := ""
	rules := []string{}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		if section == "all" || section == tool {
			rules = append(rules, line)
		}
	}
	return ignorePolicy{file: file, root: filepath.Dir(file), rules: rules}, nil
}

func findPolicyFile(base string) string {
	current := filepath.Clean(base)
	for {
		candidate := filepath.Join(current, ".nezhaignore")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(current)
		if parent == "" || parent == current {
			return ""
		}
		current = parent
	}
}

func (p ignorePolicy) ignored(path string, isDir bool) bool {
	if p.root == "" {
		return false
	}
	rel := display(p.root, path)
	for _, rule := range p.rules {
		if ruleMatches(rule, rel, isDir) {
			return true
		}
	}
	return false
}

func ruleMatches(rule, rel string, isDir bool) bool {
	_ = isDir
	r := strings.ReplaceAll(filepath.ToSlash(rule), "\\", "/")
	p := strings.ReplaceAll(filepath.ToSlash(rel), "\\", "/")
	if r == "**/.*" {
		for _, part := range strings.Split(p, "/") {
			if strings.HasPrefix(part, ".") {
				return true
			}
		}
		return false
	}
	if strings.HasSuffix(r, "/**") {
		prefix := r[:len(r)-3]
		return p == prefix || strings.HasPrefix(p, prefix+"/")
	}
	return p == r
}

func walkFiles(root string, policy ignorePolicy, bypassPolicy bool) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, entry := range entries {
		p := filepath.Join(root, entry.Name())
		isDir := entry.IsDir()
		if !bypassPolicy && policy.ignored(p, isDir) {
			continue
		}
		if isDir {
			files, err := walkFiles(p, policy, bypassPolicy)
			if err != nil {
				return nil, err
			}
			out = append(out, files...)
		} else {
			out = append(out, p)
		}
	}
	return out, nil
}

package lint

import (
	"nezha/cmd/internal/targetresolve"
	"nezha/cmd/internal/testfixture"
	"strings"
)

func expandLintPaths(args []string) ([]string, error) {
	return expandLintPathsWithPolicy(args, false)
}

func expandLintPathsWithPolicy(args []string, all bool) ([]string, error) {
	var resolved []string
	var err error
	if all {
		resolved, err = targetresolve.ResolveAll("lint", args, true)
	} else {
		resolved, err = targetresolve.Resolve("lint", args, true)
	}
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(resolved))
	for _, path := range resolved {
		include, err := lintSourceCandidate(path)
		if err != nil {
			return nil, err
		}
		if include {
			files = append(files, path)
		}
	}
	return files, nil
}

// lintSourceCandidate is the single disposition point for files entering the
// linter's AST traversal. Parse-negative smoke fixtures are intentionally
// excluded; ordinary source, including malformed source, remains lint input.
func lintSourceCandidate(path string) (bool, error) {
	if !strings.HasSuffix(path, ".nez") || strings.HasSuffix(path, "_test.nez") {
		return false, nil
	}
	skip, err := testfixture.IsParseNegative(path)
	if err != nil {
		return false, err
	}
	return !skip, nil
}

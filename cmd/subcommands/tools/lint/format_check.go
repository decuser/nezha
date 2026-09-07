package lint

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"nezha/cmd/internal/targetresolve"
	"nezha/cmd/internal/testfixture"
	"nezha/engine/syntax"
	"nezha/engine/syntax/grammar"
	nezhafmt "nezha/internal/formatting"
)

// CheckFormatting returns the list of files that are not already formatted.
// It does not rewrite files.
func CheckFormatting(args []string, includePrelude bool) ([]string, error) {
	return checkFormatting(args, includePrelude, false)
}

func checkFormatting(args []string, includePrelude, all bool) ([]string, error) {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		return nil, err
	}
	var paths []string
	if all {
		paths, err = targetresolve.ResolveAll("lint", args, false)
	} else {
		paths, err = targetresolve.Resolve("lint", args, false)
	}
	if err != nil {
		return nil, err
	}
	var bad []string
	var checkErrs []error
	for _, path := range paths {
		if !strings.HasSuffix(path, ".nez") {
			continue
		}
		if !includePrelude && strings.HasSuffix(path, "engine/runtime/prelude/prelude.nez") {
			continue
		}
		skip, err := testfixture.IsParseNegative(path)
		if err != nil {
			return bad, err
		}
		if skip {
			continue
		}
		ok, err := checkFile(g, path)
		if err != nil {
			checkErrs = append(checkErrs, fmt.Errorf("%s: %w", path, err))
			continue
		}
		if !ok {
			bad = append(bad, path)
		}
	}
	return bad, errors.Join(checkErrs...)
}

func checkFile(g *grammar.Grammar, path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	formatted, err := nezhafmt.FormatSource(g, path, string(data))
	if err != nil {
		return false, err
	}
	return formatted == string(data), nil
}

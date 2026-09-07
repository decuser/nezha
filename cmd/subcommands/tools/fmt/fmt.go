package fmt

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	formatter "nezha/cmd/internal/formatting"
	"nezha/cmd/internal/targetresolve"
	"nezha/cmd/internal/testfixture"
	"nezha/engine/syntax"
	"nezha/engine/syntax/grammar"
)

// Config controls formatting behavior.
type Config struct {
	// Write rewrites files in place.
	Write bool
	// ListOnly lists files that would change; no writes.
	ListOnly bool
	// PrintToStdout prints formatted output; no writes.
	PrintToStdout bool
	// Backup creates a .bak file before overwriting when changes occur.
	Backup bool
}

// FormatPath formats a file, recursive target, or single path.
// Plain directories remain invalid for fmt; recursive target expansion is
// delegated to the shared target resolver.
func FormatPath(path string, cfg Config) (bool, error) {
	return formatTarget(path, cfg, false)
}

func formatTarget(path string, cfg Config, all bool) (bool, error) {
	var paths []string
	var err error
	if all {
		paths, err = targetresolve.ResolveAll("fmt", []string{path}, false)
	} else {
		paths, err = targetresolve.Resolve("fmt", []string{path}, false)
	}
	if err != nil {
		return false, err
	}

	recursive := strings.HasSuffix(path, "/...") || path == "..."
	changedAny := false
	var formatErrs []error
	for _, resolved := range paths {
		if !strings.HasSuffix(resolved, ".nez") {
			continue
		}
		skip, ferr := testfixture.IsParseNegative(resolved)
		if ferr != nil {
			return changedAny, ferr
		}
		if skip {
			continue
		}
		stdout := io.Writer(os.Stdout)
		if recursive {
			stdout = io.Discard
		}
		changed, ferr := formatFile(resolved, cfg, stdout)
		if ferr != nil {
			formatErrs = append(formatErrs, fmt.Errorf("%s: %w", resolved, ferr))
			continue
		}
		if changed {
			changedAny = true
		}
	}
	return changedAny, errors.Join(formatErrs...)
}

func loadGrammar() (*grammar.Grammar, error) {
	return grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
}

func formatFile(path string, cfg Config, stdout io.Writer) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	g, err := loadGrammar()
	if err != nil {
		return false, err
	}

	formatted, err := formatter.FormatSource(g, path, string(data))
	if err != nil {
		return false, err
	}

	changed := formatted != string(data)

	if cfg.PrintToStdout {
		if !changed {
			// Still print the current file when asked.
			_, _ = stdout.Write(data)
			return false, nil
		}
		_, _ = io.Copy(stdout, bytes.NewBufferString(formatted))
		return true, nil
	}

	if cfg.ListOnly {
		if changed {
			fmt.Fprintln(os.Stdout, path)
			return true, nil
		}
		return false, nil
	}

	if cfg.Write {
		if !changed {
			return false, nil
		}

		// Print changed filename.
		fmt.Fprintln(os.Stdout, path)

		if cfg.Backup {
			bak := path + ".bak"
			if err := os.WriteFile(bak, data, 0644); err != nil {
				return false, err
			}
		}

		if err := atomicWriteFile(path, []byte(formatted), 0644); err != nil {
			return false, err
		}
	}
	return changed, nil
}

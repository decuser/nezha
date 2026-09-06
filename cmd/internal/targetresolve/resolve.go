package targetresolve

import (
	"fmt"
	"os"

	"aiki/engine/runner"
	"aiki/engine/semantics/value"
)

// Resolve delegates filesystem target semantics to lib/target and translates
// only the typed result into Go paths for CLI consumers.
func Resolve(tool string, targets []string, recursePlainDirs bool) ([]string, error) {
	return resolve(tool, targets, recursePlainDirs, false)
}

// ResolveAll is the exhaustive form of Resolve. It preserves target syntax and
// command directory policy but bypasses .aikiignore selection policy.
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

func resolveFrom(base, tool string, targets []string, recursePlainDirs, includeIgnored bool) ([]string, error) {
	args := make([]value.Value, len(targets))
	for i, target := range targets {
		args[i] = &value.String{Val: target}
	}
	operation := "resolve"
	if includeIgnored {
		operation = "resolve_all"
	}
	result, err := runner.CallLibrary(
		"target",
		operation,
		&value.String{Val: base},
		&value.String{Val: tool},
		&value.List{Elements: args},
		func() value.Value {
			if recursePlainDirs {
				return value.TRUE
			}
			return value.FALSE
		}(),
	)
	if err != nil {
		return nil, err
	}
	list, ok := result.(*value.List)
	if !ok {
		return nil, fmt.Errorf("target.%s returned %s, want shaped list", operation, result.Type())
	}
	if list.Shape == "error" {
		return nil, fmt.Errorf("%s", list.Inspect())
	}
	if list.Shape != "targets" || len(list.Elements) < 1 {
		return nil, fmt.Errorf("target.%s returned %s, want @targets", operation, list.Inspect())
	}
	paths, ok := list.Elements[0].(*value.List)
	if !ok {
		return nil, fmt.Errorf("target.%s paths are %s, want list", operation, list.Elements[0].Type())
	}
	out := make([]string, len(paths.Elements))
	for i, elem := range paths.Elements {
		s, ok := elem.(*value.String)
		if !ok {
			return nil, fmt.Errorf("target.%s path %d is %s, want string", operation, i, elem.Type())
		}
		out[i] = s.Val
	}
	return out, nil
}

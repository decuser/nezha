package substrate

import (
	"nezha/engine/runtime/hal"
	"nezha/engine/semantics/value"
)

// AuthorityForSource returns only the grants explicitly declared for a
// canonical trusted source. Host bindings are translated to canonical HAL
// identities; non-host primitives retain their implementation names. Merely
// residing beneath lib/ confers nothing.
func (g *GoRuntime) AuthorityForSource(path string) value.Authority {
	bindings, ok := hal.AuthorityPrimitivesForSource(path)
	if !ok {
		bindings = nil
	}
	if g.isCurrentTestFile(path) {
		bindings = append(bindings, "_test_equal", "_test_not_equal", "_test_true", "_test_false", "_test_error", "_test_not_error", "_test_run", "_test_faults")
	}
	if len(bindings) == 0 {
		return value.NoAuthority()
	}
	grants := make([]string, 0, len(bindings))
	for _, name := range bindings {
		grants = append(grants, g.authorityKey(name))
	}
	return value.NewAuthority(grants...)
}

func (g *GoRuntime) isCurrentTestFile(path string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.testState.file != "" && g.testState.file == path
}

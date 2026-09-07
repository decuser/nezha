package hal

import (
	"path/filepath"
	"strings"
)

// authorityPolicy records the raw runtime bindings explicitly granted to each
// trusted Nezha source. It is architectural policy; substrate code translates
// host primitives to canonical HAL authority identities at runtime.
var authorityPolicy = map[string][]string{
	"engine/runtime/prelude/prelude.nez": {
		"_append",
		"_apply",
		"_empty",
		"_equal",
		"_first",
		"_length",
		"_make_shaped_list",
		"_prepend",
		"_print",
		"_rest",
		"_shape",
		"_to_str",
		"_type",
	},
}

// AuthorityPrimitivesForSource returns the declared raw runtime bindings for a
// canonical trusted source. Merely residing beneath lib/ confers no authority.
func AuthorityPrimitivesForSource(path string) ([]string, bool) {
	normalized := filepath.ToSlash(filepath.Clean(path))
	for suffix, bindings := range authorityPolicy {
		if normalized == suffix || strings.HasSuffix(normalized, "/"+suffix) {
			return append([]string(nil), bindings...), true
		}
	}
	return nil, false
}

// AuthorityPolicy returns an independent copy for invariant checking.
func AuthorityPolicy() map[string][]string {
	out := make(map[string][]string, len(authorityPolicy))
	for source, bindings := range authorityPolicy {
		out[source] = append([]string(nil), bindings...)
	}
	return out
}

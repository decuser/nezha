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
		"_channel",
		"_chr",
		"_delete",
		"_doc",
		"_empty",
		"_equal",
		"_first",
		"_help",
		"_inspect",
		"_length",
		"_load",
		"_make_shaped_list",
		"_ord",
		"_prepend",
		"_print",
		"_quit",
		"_read",
		"_recv",
		"_reset",
		"_rest",
		"_send",
		"_shape",
		"_spawn",
		"_stack_limit",
		"_to_decimal",
		"_to_number",
		"_to_str",
		"_to_symbol",
		"_truncate",
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

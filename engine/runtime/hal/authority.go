package hal

import (
	"path/filepath"
	"strings"
)

// authorityPolicy records the raw runtime bindings explicitly granted to each
// trusted Aiki source. It is architectural policy; substrate code translates
// host primitives to canonical HAL authority identities at runtime.
var authorityPolicy = map[string][]string{
	"engine/runtime/prelude/prelude.ai": {
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
	"lib/file/file.ai": {
		"_file_close",
		"_file_copy",
		"_file_delete",
		"_file_exists",
		"_file_list",
		"_file_entries",
		"_file_mkdir",
		"_file_mkdir_all",
		"_file_open",
		"_file_read_at",
		"_file_read_bytes",
		"_file_read_line",
		"_file_read_text",
		"_file_remove_all",
		"_file_rename",
		"_file_size",
		"_file_walk",
		"_file_symlink",
		"_file_read_link",
		"_file_permissions",
		"_file_chmod",
		"_file_lock",
		"_file_try_lock",
		"_file_unlock",
		"_file_stat",
		"_file_temp",
		"_file_temp_dir",
		"_file_write_at",
		"_file_write_bytes",
		"_file_write_text",
	},
	"lib/hash/ffi.ai": {
		"_hash_code",
		"_hash_new",
		"_hash_get",
		"_hash_put",
		"_hash_has",
		"_hash_del",
		"_hash_keys",
		"_hash_values",
	},
	"lib/math/ffi.ai": {
		"_ceil",
		"_floor",
		"_modulo",
	},
	"lib/math/approx.ai": {
		"_cos_inexact",
		"_sin_inexact",
		"_sqrt_inexact",
	},
	"lib/string/ffi.ai": {
		"_chars",
		"_lower",
		"_lower_rune",
		"_string_compare",
		"_string_contains",
		"_string_ends_with",
		"_string_index_of",
		"_string_join",
		"_string_last_index_of",
		"_string_repeat",
		"_string_replace",
		"_string_replace_first",
		"_string_reverse",
		"_string_split",
		"_string_starts_with",
		"_string_substring",
		"_string_trim",
		"_string_trim_end",
		"_string_trim_start",
		"_upper",
		"_upper_rune",
		"_string_is_whitespace",
		"_string_is_digit",
		"_string_is_alpha",
		"_string_is_upper",
		"_string_is_lower",
		"_string_is_alnum",
		"_string_is_numeric",
		"_string_is_alphabetic",
	},
	"lib/test/test.ai": {
		"_test_equal",
		"_test_error",
		"_test_false",
		"_test_faults",
		"_test_not_equal",
		"_test_not_error",
		"_test_run",
		"_test_true",
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

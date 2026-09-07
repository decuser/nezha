package substrate

// registerHAL installs the runtime binding surface separated by architectural
// responsibility. Only entries registered through registerHost have canonical
// HAL identities; the remaining registries implement non-HAL runtime roles.
func (g *GoRuntime) registerHAL() {
	// True host-facing operations. Canonical contracts currently exist for prelude I/O and file operations.
	g.registerHost(goHostOperation("_print"), g.halPrint)
	g.registerHost(goHostOperation("_read"), g.halRead)
	g.registerHost(goHostOperation("_file_open"), g.halFileOpenPath)
	g.registerHost(goHostOperation("_file_read_text"), halFileReadText)
	g.registerHost(goHostOperation("_file_read_bytes"), halFileReadBytes)
	g.registerHost(goHostOperation("_file_read_line"), g.halFileReadLine)
	g.registerHost(goHostOperation("_file_write_text"), halFileWriteText)
	g.registerHost(goHostOperation("_file_write_bytes"), halFileWriteBytes)
	g.registerHost(goHostOperation("_file_close"), g.halFileClose)
	g.registerHost(goHostOperation("_file_exists"), g.halFileExistsPath)
	g.registerHost(goHostOperation("_file_delete"), g.halFileDeletePath)
	g.registerHost(goHostOperation("_file_list"), g.halFileListPath)
	g.registerHost(goHostOperation("_file_entries"), g.halFileEntriesPath)
	g.registerHost(goHostOperation("_file_read_at"), halFileReadAt)
	g.registerHost(goHostOperation("_file_write_at"), halFileWriteAt)
	g.registerHost(goHostOperation("_file_stat"), g.halFileStatPath)
	g.registerHost(goHostOperation("_file_rename"), g.halFileRenamePath)
	g.registerHost(goHostOperation("_file_mkdir"), g.halFileMkdirPath)
	g.registerHost(goHostOperation("_file_mkdir_all"), g.halFileMkdirAllPath)
	g.registerHost(goHostOperation("_file_remove_all"), g.halFileRemoveAllPath)
	g.registerHost(goHostOperation("_file_temp"), halFileTemp)
	g.registerHost(goHostOperation("_file_temp_dir"), halFileTempDir)
	g.registerHost(goHostOperation("_file_copy"), g.halFileCopyPath)
	g.registerHost(goHostOperation("_file_size"), g.halFileSizePath)
	g.registerHost(goHostOperation("_file_walk"), g.halFileWalkPath)
	g.registerHost(goHostOperation("_file_symlink"), g.halFileSymlinkPath)
	g.registerHost(goHostOperation("_file_read_link"), g.halFileReadLinkPath)
	g.registerHost(goHostOperation("_file_lock"), g.halFileLock)
	g.registerHost(goHostOperation("_file_try_lock"), g.halFileTryLock)
	g.registerHost(goHostOperation("_file_unlock"), g.halFileUnlock)
	g.registerHost(goHostOperation("_file_permissions"), g.halFilePermissionsPath)
	g.registerHost(goHostOperation("_file_chmod"), g.halFileChmodPath)

	// Evaluator/language intrinsics.
	g.registerPrimitive("_apply", halApply)
	g.registerPrimitive("_import", g.halImport)
	g.registerPrimitive("_use", g.halUse)
	g.registerPrimitive("_export", halExport)
	g.registerPrimitive("_load", halLoad)
	g.registerPrimitive("_spawn", halSpawn)
	g.registerPrimitive("_channel", halChannel)
	g.registerPrimitive("_send", halSend)
	g.registerPrimitive("_recv", halRecv)

	// Language/value primitives implemented natively. Append remains context-free
	// but accepts the active semantic probe for hidden representation counters.
	g.registerContextFreeProbedPrimitive("_append", halAppend, halAppendWithProbe)
	for name, fn := range map[string]BuiltinFunc{
		"_first": halFirst, "_rest": halRest, "_length": halLength,
		"_prepend": halPrepend, "_empty": halEmpty, "_range": halRange,
		"_type": halType, "_stack_limit": halStackLimit, "_inspect": halInspect,
		"_equal": halEqual, "_ord": halOrd, "_chr": halChr,
		"_floor": halFloor, "_ceil": halCeil, "_truncate": halTruncate, "_modulo": halModulo,
		"_shape": halShape, "_make_shaped_list": halMakeShapedList, "_to_str": halToStr,
		"_to_decimal": halToDecimal, "_to_number": halToNumber, "_to_symbol": halToSymbol,
	} {
		switch name {
		case "_stack_limit":
			// These operations observe evaluator context for stack state or
			// semantic profiling. Keep the conservative context path.
			g.registerPrimitive(name, fn)
		default:
			g.registerContextFreePrimitive(name, fn)
		}
	}

	// Native/FFI library providers. Native realization does not imply host
	// authority; these are alternate implementations of library behavior.
	for name, fn := range map[string]BuiltinFunc{
		"_sqrt_inexact": halSqrt, "_cos_inexact": halCos, "_sin_inexact": halSin,
		"_upper": halUpper, "_lower": halLower, "_chars": halChars,
		"_string_substring": halStringSubstring, "_string_split": halStringSplit,
		"_string_index_of": halStringIndexOf, "_string_last_index_of": halStringLastIndexOf,
		"_string_contains": halStringContains, "_string_starts_with": halStringStartsWith,
		"_string_ends_with": halStringEndsWith, "_string_join": halStringJoin,
		"_string_replace": halStringReplace, "_string_replace_first": halStringReplaceFirst,
		"_string_repeat": halStringRepeat, "_string_reverse": halStringReverse,
		"_string_trim": halStringTrim, "_string_trim_start": halStringTrimStart,
		"_string_trim_end": halStringTrimEnd, "_string_compare": halStringCompare,
		"_string_is_whitespace": halStringIsWhitespace, "_string_is_digit": halStringIsDigit,
		"_string_is_alpha": halStringIsAlpha, "_string_is_upper": halStringIsUpper,
		"_string_is_lower": halStringIsLower, "_string_is_alnum": halStringIsAlnum,
		"_string_is_numeric": halStringIsNumeric, "_string_is_alphabetic": halStringIsAlphabetic,
		"_hash_code": halHashCode, "_hash_new": halHashNew, "_hash_get": halHashGet,
		"_hash_put": halHashPut, "_hash_has": halHashHas, "_hash_del": halHashDel,
		"_hash_keys": halHashKeys, "_hash_values": halHashValues,
		"_upper_rune": halUpperRune, "_lower_rune": halLowerRune,
	} {
		g.registerContextFreePrimitive(name, fn)
	}

	// Runtime/tooling/session services.
	for name, fn := range map[string]BuiltinFunc{
		"_module_roots": g.halModuleRoots,
		"_quit":         g.halQuit, "_reset": g.halReset, "_delete": g.halDelete, "_help": g.halHelp, "_doc": g.halDoc,
		"_test_equal": g.halTestEqual, "_test_not_equal": g.halTestNotEqual,
		"_test_true": g.halTestTrue, "_test_false": g.halTestFalse,
		"_test_error": g.halTestError, "_test_not_error": g.halTestNotError,
		"_test_run": g.halTestRun, "_test_faults": g.halTestFaults,
	} {
		g.registerPrimitive(name, fn)
	}
}

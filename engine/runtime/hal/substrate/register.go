package substrate

// registerHAL installs the runtime binding surface separated by architectural
// responsibility. Only entries registered through registerHost have canonical
// HAL identities; the remaining registries implement non-HAL runtime roles.
func (g *GoRuntime) registerHAL() {
	// True host-facing operations. Canonical contracts currently exist for prelude I/O and file operations.
	g.registerHost(goHostOperation("_print"), g.halPrint)
	g.registerHost(goHostOperation("_read"), g.halRead)

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

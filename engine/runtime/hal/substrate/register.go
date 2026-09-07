package substrate

// registerHAL installs the runtime binding surface separated by architectural
// responsibility. Only entries registered through registerHost have canonical
// HAL identities; the remaining registries implement non-HAL runtime roles.
func (g *GoRuntime) registerHAL() {
	// True host-facing operations. Canonical contracts currently exist for prelude I/O and file operations.
	g.registerHost(goHostOperation("_print"), g.halPrint)

	// Evaluator/language intrinsics.
	g.registerPrimitive("_apply", halApply)

	// Language/value primitives implemented natively. Append remains context-free
	// but accepts the active semantic probe for hidden representation counters.
	g.registerContextFreeProbedPrimitive("_append", halAppend, halAppendWithProbe)
	for name, fn := range map[string]BuiltinFunc{
		"_first": halFirst, "_rest": halRest, "_length": halLength,
		"_prepend": halPrepend, "_empty": halEmpty, "_range": halRange,
		"_type": halType, "_stack_limit": halStackLimit, "_inspect": halInspect,
		"_equal": halEqual, "_ord": halOrd, "_chr": halChr,
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

}

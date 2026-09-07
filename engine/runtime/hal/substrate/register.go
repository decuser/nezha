package substrate

// registerHAL installs the runtime binding surface separated by architectural
// responsibility. Only entries registered through registerHost have canonical
// HAL identities; the remaining registries implement non-HAL runtime roles.
func (g *GoRuntime) registerHAL() {
	// True host-facing operations. Canonical contracts currently exist for prelude I/O and file operations.
	g.registerHost(goHostOperation("_print"), g.halPrint)

	// Evaluator/language intrinsics.
	g.registerPrimitive("_apply", halApply)

	// Language/value primitives implemented natively.
	g.registerContextFreePrimitive("_append", halAppend)
	for name, fn := range map[string]BuiltinFunc{
		"_first": halFirst, "_rest": halRest, "_length": halLength,
		"_prepend": halPrepend, "_empty": halEmpty,
		"_type": halType, "_equal": halEqual,
		"_shape": halShape, "_make_shaped_list": halMakeShapedList, "_to_str": halToStr,
	} {
		g.registerContextFreePrimitive(name, fn)
	}

}

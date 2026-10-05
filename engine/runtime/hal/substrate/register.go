package substrate

// registerHAL installs the flat runtime binding surface. Substrate builtins are
// leaf functions; canonical host metadata is attached only where a primitive
// crosses the HAL boundary.
func (g *GoRuntime) registerHAL() {
	g.registerHost(goHostOperation("_print"), variadicArity(0), g.halPrint)
	g.registerHost(goHostOperation("_read"), fixedArity(0), g.halRead)

	for name, spec := range map[string]struct {
		arity Arity
		fn    BuiltinFunc
	}{
		"_append":           {fixedArity(2), halAppend},
		"_empty":            {fixedArity(1), halEmpty},
		"_equal":            {fixedArity(2), halEqual},
		"_first":            {fixedArity(1), halFirst},
		"_length":           {fixedArity(1), halLength},
		"_make_shaped_list": {fixedArity(2), halMakeShapedList},
		"_prepend":          {fixedArity(2), halPrepend},
		"_rest":             {fixedArity(1), halRest},
		"_shape":            {fixedArity(1), halShape},
		"_to_str":           {fixedArity(1), halToStr},
		"_type":             {fixedArity(1), halType},
	} {
		g.registerPrimitive(name, spec.arity, spec.fn)
	}
}

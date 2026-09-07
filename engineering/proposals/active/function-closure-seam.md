# Function Closure Seam

Nezha currently keeps function bodies and closure environments in the value
layer as opaque fields:

```go
Body interface{}
Env  interface{}
```

In normal execution these fields are always the function body syntax node and
the lexical environment captured at function creation. The evaluator constructs
and consumes the pair, so the runtime fact is stable even though the Go type
system cannot express it from inside the value package.

This is a live design seam, not dead residue. A function value is where syntax
and lexical scope become a callable closure. Making that relationship explicit
will matter when evaluation behavior is decomposed, but resolving it now would
force a larger representation decision.

Current disposition:

- retain the opaque fields for the compact specimen;
- do not let substrate or HAL consume function bodies;
- keep function application owned by the evaluator;
- revisit this seam during closure/evaluation-character work.

Possible future directions:

- move closure representation into a package that can name syntax and values;
- replace direct syntax pointers with evaluator-owned body handles;
- introduce an evaluator-owned closure value while preserving the public value
  vocabulary;
- deliberately allow value to depend on a small syntax-body interface.

The invariant for the current specimen is simple: only evaluator code should
construct or interpret function bodies and closure environments.

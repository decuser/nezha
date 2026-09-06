# Adding to Aiki

This guide describes how to extend the Aiki language at each layer.

## Architecture Overview

```
┌─────────────────────────────────────────────┐
│  User Code (.ai files)                      │
├─────────────────────────────────────────────┤
│  Lib (lib/*.ai)                             │
│  - Packages written in Aiki                 │
│  - Require: package, export                 │
├─────────────────────────────────────────────┤
│  Prelude (engine/runtime/prelude/)          │
│  - Aiki wrappers around runtime primitives  │
│  - Always available, no import needed       │
├─────────────────────────────────────────────┤
│  Runtime primitive architecture             │
│  (engine/runtime/primitives/)               │
│  - intrinsic/runtime/provider/host/service  │
│  - every raw primitive has one role         │
├─────────────────────────────────────────────┤
│  HAL host contracts + Go substrate          │
│  (engine/runtime/hal/)                      │
│  - HAL only for irreducible host crossings  │
│  - substrate realizes primitives/contracts  │
├─────────────────────────────────────────────┤
│  Evaluator (engine/semantics/evaluator/)    │
│  - Interprets AST nodes                     │
│  - One handler per grammar production       │
├─────────────────────────────────────────────┤
│  Parser/Lexer (engine/syntax/)              │
│  - Grammar-driven                           │
│  - EBNFX is source of truth                 │
└─────────────────────────────────────────────┘
```

A Go implementation is not automatically a HAL operation. The primitive-role
catalog in `engine/runtime/primitives/catalog.go` says why a raw runtime
primitive exists; `docs/hal.md` and the HAL host-operation catalog govern the
smaller set of irreducible host crossings.

## Executable Invariants

Aiki relies on executable couplings among independently maintained artifacts to
detect implementation and documentation drift. The current invariant catalog
is maintained in `test/invariant/README.md`; this guide deliberately does not
duplicate that evolving inventory.

The checks compare independent artifacts rather than allowing one artifact to
certify another derived from the same assumption. When extending Aiki, preserve
the relevant existing coupling. If a change introduces a new independently
maintained relationship, add an invariant for that relationship rather than
relying on manual review.

## Adding a Runtime Primitive

Example: adding `abs(n)` as a constitutive runtime primitive. This example is
not a host operation, so it does not acquire a HAL host identity merely because
its substrate implementation is written in Go.

### 1. Classify the primitive

Add `_abs` to `engine/runtime/primitives/catalog.go` with the architectural role
that describes why it exists. For this example:

```go
"_abs": primitives.RoleRuntime,
```

The primitive-role invariant requires every runtime registration to have exactly
one architectural role. If the new operation is an irreducible host crossing
instead, define the canonical HAL contract and authority described in
`docs/hal.md`; do not classify it as an ordinary runtime primitive.

### 2. Implement the substrate function

Create or edit a file in `engine/runtime/hal/substrate/`:

```go
// builtins_math.go
func halAbs(args []value.Value, ctx *hal.EvalContext) value.Value {
    if len(args) != 1 {
        return value.NewFault("abs: want 1 argument, got %d", len(args))
    }
    
    num, ok := args[0].(*value.Number)
    if !ok {
        return value.NewFault("abs: expected number")
    }
    
    if num.Sign() < 0 {
        return num.Neg()
    }
    return num
}
```

### 3. Register the runtime primitive

In `engine/runtime/hal/substrate/register.go`, use the registry path matching
the primitive's audited context needs. `abs` is context-free:

```go
g.registerContextFreePrimitive("_abs", halAbs)
```

New or unaudited primitives should use `registerPrimitive`, whose conservative
default retains evaluator context. Host operations use `registerHost` with their
canonical HAL operation instead.

### 4. Prelude Wrapper

In `engine/runtime/prelude/prelude.ai`:

```
let abs = (n) { _abs(n) }
```

### 5. Help Entry

In `engine/runtime/prelude/prelude.help`:

```
@func abs
@template "abs(n)"
@help "Returns absolute value of number."
```

### 6. Doc Entry

In `engine/runtime/prelude/prelude.doc`:

```
abs
Returns the absolute value of a number.

abs(n)

abs(-5)    # 5
abs(3)     # 3
abs(0)     # 0
===
```

### 7. Test

Create `test/behavior/abs_smoke.ai`:

```
println(abs(-5))
println(abs(5))
println(abs(0))
println("ABS_SMOKE_DONE")
```

Create `test/behavior/abs_smoke.gold`:

```
OUT:5
OUT:5
OUT:0
OUT:ABS_SMOKE_DONE
```

The `.gold` file format:
- `OUT:` - expected stdout
- `IN:` - simulated stdin
- `ERR:` - expected stderr
- `EXIT:` - expected exit code (default 0)

### 8. Verify

```bash
go build ./cmd/aiki
./aiki smoke test/behavior/
```

## Adding New Syntax

Example: adding `unless` as opposite of `if`.

### 1. Grammar Production

In `engine/syntax/grammar.ebnfx`:

Add to statement alternatives:
```
statement = package_stmt | let_stmt | assign_stmt | if_stmt | unless_stmt | ...
```

Add the production:
```
unless_stmt = "unless" expr block [ "else" block ]
    @error "expected 'unless condition { ... }'"
    @template "unless cond { body } | unless cond { body } else { alt }"
    @help "executes body if condition is false"
```

Add keyword:
```
KEYWORD     let if else while match return true false and or not package unless
```

### 2. Grammar Help

In `engine/syntax/grammar.help`:

```
unless_stmt

Executes body if condition is false. Opposite of if.

unless empty(list) {
    print(first(list))
}

unless found {
    print("not found")
} else {
    print("found it")
}
===
```

### 3. Handler Registration

In `engine/semantics/evaluator/evaluator.go`:

```go
var handlers = map[string]handlerFunc{
    // ...
    "unless_stmt": (*Evaluator).evalUnless,
    // ...
}
```

### 4. Implementation

In `engine/semantics/evaluator/statements.go`:

```go
func (e *Evaluator) evalUnless(node *syntax.Node, env *value.Env) value.Value {
    var cond value.Value
    var thenBlock, elseBlock *syntax.Node
    
    for _, child := range node.Children {
        switch child.Type {
        case "expr":
            cond = e.Eval(child, env)
        case "block":
            if thenBlock == nil {
                thenBlock = child
            } else {
                elseBlock = child
            }
        }
    }
    
    if value.IsError(cond) {
        return cond
    }
    
    // unless = if NOT condition
    if !value.IsTruthy(cond) {
        return e.Eval(thenBlock, env)
    } else if elseBlock != nil {
        return e.Eval(elseBlock, env)
    }
    
    return value.EMPTY
}
```

### 5. Test

Create `test/behavior/unless_smoke.ai`:

```
let x = 5
unless x > 10 {
    println("small")
}
unless x < 10 {
    println("big")
} else {
    println("also small")
}
println("UNLESS_SMOKE_DONE")
```

Create `test/behavior/unless_smoke.gold`:

```
OUT:small
OUT:also small
OUT:UNLESS_SMOKE_DONE
```

### 6. Verify

```bash
go build ./cmd/aiki
./aiki smoke test/behavior/
```

## Adding a Library Package

Before adding a package, decide what semantic role it has and record that role in
`engine/runtime/modules/stdlib_policy.go`.

Portable semantics and host/runtime capabilities are different contracts:

- a portable `X/native` module implements the public library semantics in Aiki;
- an optional `X/ffi` module is provider-backed and accelerates either the full
  native authority surface or an explicitly declared `AccelerationSurface`;
  every accelerated operation must preserve the native signature, help/doc, and
  behavior contract;
- host/runtime capabilities and deliberate provider interop do not require fake
  native twins, but their role must be declared honestly;
- bare `X` resolves to `X/native` when a native module is present and no explicit
  bare package overrides that default.

### 1. Add the Native Semantic Authority

For a portable package, create `lib/example/native.ai`:

```
package "example/native"

let twice = (n) {
    n + n
}

export(:twice)
```

A `/native` package may use constitutive Aiki runtime atoms exposed by the
prelude, but it must not import `/ffi` modules or call provider-role primitives
to implement its library algorithm.

Add matching `native.help` and `native.doc` files and tests against the bare
`example` import so the default path exercises the native implementation.

### 2. Add an FFI Acceleration Only if Needed

If a coarse provider boundary materially helps, add `lib/example/ffi.ai`:

```
package "example/ffi"

let twice = (n) {
    _example_twice(n)
}

export(:twice)
```

Register `_example_twice` as a provider primitive and grant it only to the FFI
source through HAL authority policy. Declare `example/ffi` as portable/ffi with
`example/native` as its semantic authority.

By default, an acceleration mirrors the complete native export surface. If only
a subset is accelerated, declare that subset with `AccelerationSurface` in
`stdlib_policy.go`. The policy invariants require every accelerated operation to
match the native callable shape, help/doc entry, and behavior contract. Functions
outside the declared acceleration surface remain native-only; provider-specific
semantics belong in an `interop` module instead of being disguised as
acceleration.

### 3. Add a Capability or Interop Module Honestly

If the behavior intrinsically depends on a host resource (files, processes,
terminal state, clocks, etc.), classify it as a host/runtime capability and use
HAL authority directly as needed. If the purpose is to expose a provider's own
semantics, classify it as interop. Neither case needs a fake pure-Aiki twin.

### 4. Verify

Run the relevant focused tests while cutting the change, then the repository
validation gate. The stdlib policy invariants check declaration completeness,
truthful `/native` and `/ffi` naming, native-path purity, acceleration surface
parity, and help/doc parity.

## Validation

The system enforces consistency through executable couplings rather than relying
on this document as authority. Among them:

1. grammar productions ↔ evaluator handlers and structural engine coverage;
2. grammar/prelude/library exports ↔ help and documentation surfaces;
3. formatted source ↔ parsed structure;
4. shipped modules ↔ registry discovery and distribution-tree relationships;
5. behavior ↔ blessed smoke transcripts;
6. HAL identities ↔ substrate registration, provenance, and exact authority;
7. stdlib semantic policy ↔ truthful `/native`, `/ffi`, capability, interop, and
   bare-default behavior.

Use focused checks while cutting a change, then `make validate-lang` for the
routine language/runtime repository gate. Use `make validate-all` when exhaustive
applicable repository selection is required, and `make rigorous` when the work
warrants the expensive self-host/fuzz gate. Select the required validation level explicitly; there is no generic validation alias.

## File Locations

```
engine/
  syntax/
    grammar.ebnfx      # Grammar source of truth
    grammar.help       # Grammar documentation
  semantics/
    evaluator/         # AST interpretation
    value/             # Value types
  runtime/
    hal/substrate/     # Go primitives
    prelude/
      prelude.ai       # Aiki wrappers
      prelude.help     # Quick reference
      prelude.doc      # Full documentation
lib/
  math/
    native.ai          # Portable Aiki semantic authority
    ffi.ai             # Provider-backed realization where declared
test/
  behavior/            # Behavior smoke specimens and golds
```


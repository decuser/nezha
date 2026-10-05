# Evaluation Character Inventory

This note begins the decomposition pass for the current Nezha specimen. It is an
inventory, not an implementation plan for `eval.spec`.

## Current specimen

The current language is a single loaded script evaluated with the embedded
prelude. It has lexical scope, lists and shaped lists as structure, functions,
rest parameters, imperative control, pattern matching, pipe syntax, and one
HAL-backed text I/O operations.

The runtime boundary is now clean enough for this inventory:

```text
evaluator-owned intrinsic:
- _apply

substrate leaf primitives:
- _print
- _read
- _first
- _rest
- _length
- _prepend
- _append
- _empty
- _type
- _equal
- _shape
- _make_shaped_list
- _to_str
```

Only the evaluator evaluates Nezha bodies. Substrate builtins are leaves.

## Candidate evaluation characters

These names are provisional. A character only survives if it is necessary,
recurring, compositional, and semantic rather than merely an implementation
convenience.

```text
produce        create a literal or constructed value
resolve        find a binding by name under lexical scope and authority
bind           introduce a new binding
update         change an existing binding
sequence       evaluate statements in order
branch         choose a block based on a condition
loop           repeat evaluation while a condition holds
discriminate   compare a value against patterns
capture        bind pattern variables
construct      build a list, shaped list, function, or fault/return carrier
access         project from indexed or shaped structure
operate        apply primitive operator semantics
invoke         call a callable
apply          invoke a function with a list of arguments
return         unwind from a function body with a value
fault          halt local evaluation with an error value
effect         cross the HAL boundary for observable text I/O
pipe           feed a value left-to-right into a callable expression
```

Some of these may collapse. For example, `branch` and `loop` may be forms of
control; `construct` and `produce` may split by value class; `apply` may be a
special case of `invoke`.

## Production mapping

### program / block

Likely characters:

```text
sequence
fault propagation
return propagation
```

Questions:

```text
Is statement sequencing an explicit character, or just container evaluation?
Does a block create scope, or do only functions/match bindings create scope?
```

### let_stmt

Value binding:

```text
resolve?      only through expression evaluation
bind          introduce name
construct     for shape definitions
```

Shape declaration:

```text
construct shape definition
bind shape name into the environment's shape table
```

Questions:

```text
Is shape definition ordinary binding or a separate structural declaration?
Are shape names values, type tags, or environment metadata?
```

### assign_stmt

Likely characters:

```text
resolve binding target
update existing binding
evaluate assigned expression
```

Questions:

```text
Does update search outward through enclosing lexical scopes?
Is prelude assignment protection a language rule, authority rule, or runner policy?
```

### if_stmt

Likely characters:

```text
test
branch
sequence
```

Questions:

```text
What exactly counts as truth?
Is truthiness only boolean true/false or broader?
```

### while_stmt

Likely characters:

```text
test
loop
sequence
update interaction
return/fault propagation
```

Questions:

```text
Is loop a primitive control character or a repeated branch?
Does while introduce scope? Current behavior should be documented explicitly.
```

### match_stmt / pattern

Likely characters:

```text
discriminate
capture
bind
sequence
construct? for shaped/list pattern structure
```

Pattern alternatives:

```text
_                         wildcard
NAME                      capture
literal                   equality test
[pattern, ...]            list-structure discrimination
[@shape, pattern, ...]    shaped-list discrimination
```

Questions:

```text
Does NAME always capture, or can it compare existing bindings?
Do captures introduce a fresh scope per arm?
How are partial bindings discarded after failed pattern attempts?
```

### return_stmt

Likely characters:

```text
return
produce return carrier
unwind through statement/block evaluation
```

Questions:

```text
Is return a value-level carrier or evaluator control effect?
Should ReturnType remain in the value universe long term?
```

### expr / infix_expr / unary_expr / BINOP

Likely characters:

```text
produce
resolve
operate
test for and/or behavior
```

Current invariant to confirm:

```text
binary operators evaluate left-to-right
and/or are operators, not function calls
```

Questions:

```text
Are and/or lazy or strict in the current evaluator?
Are arithmetic and comparison operations evaluation characters or primitive value operations?
```

### pipe_expr

Likely characters:

```text
pipe
invoke
sequence left-to-right
```

Questions:

```text
Is pipe syntax a desugaring into call/application, or a distinct evaluation form?
Can every pipe be expressed as ordinary call without semantic loss?
```

### postfix_expr / call

Likely characters:

```text
resolve callable
construct argument list/frame
invoke
bind parameters
sequence function body
return unwrap
fault propagation
```

Questions:

```text
What is the single authoritative call path after _apply moved into evaluator?
How should tail-call behavior be represented: semantic rule or realization optimization?
```

### index

Likely characters:

```text
access
operate on list/string structure
fault on wrong type or out-of-range
```

Questions:

```text
Are indexes exact numbers only?
What is the index base? Current behavior should be stated.
```

### access

Likely characters:

```text
access
shape lookup
fault on unshaped or unknown field
```

Questions:

```text
Is field access purely by shape definition position?
How do embedded shape fields compose?
```

### primary / literal

Likely characters:

```text
produce literal value
resolve name
```

Questions:

```text
Are symbols and shapes both values, or are shapes only structural tags?
```

### list_literal

Likely characters:

```text
construct list
construct shaped list
evaluate elements left-to-right
```

Questions:

```text
Is shaped-list construction syntax equivalent to prelude shaped(sym,elements), or distinct?
What arity/field validation occurs at construction time?
```

### func_literal / params / rest_param

Likely characters:

```text
construct function closure
capture lexical environment
bind fixed params
bind rest param
```

Questions:

```text
How should closure body and environment be represented without Body interface{} / Env interface{}?
Are rest params core language or prelude support? Current disposition: core language.
```

## Prelude mapping

Prelude names mostly expose leaf primitives under ordinary names.

```text
apply       evaluator-owned application intrinsic
print       HAL effect through _print
println     HAL effect through _print plus newline
first/rest  list access/deconstruction
length      list/string size operation
prepend     list construction/update-like construction
append      list construction/update-like construction
empty       list predicate
 type       value classification
equal       equality operation
shape       shaped value tag access
is_error    shaped error convention
shaped      construct shaped list
to_str      string representation
abs         pure Nezha control/expression example
```

Questions:

```text
Which prelude names are semantic primitives versus convenience vocabulary?
Does prelude remain part of the language specimen or become a library-shaped layer later?
```

## Non-characters

These should not become evaluation characters merely because they exist in Go:

```text
arity dispatch
builtin table lookup
HAL metadata validation
formatter decisions
lint diagnostics
treecheck justification
smoke/gold comparison
cleanroom setup
tail-call frame pooling
argument-store pooling
Go map lookup
```

Some of these are important realization invariants, but they are not language
semantics.

## Next inventory pass

The next pass should read evaluator files in this order:

```text
statements.go
expressions.go
functions.go
tailcalls.go
patterns.go
access.go
operators.go
literals.go
intrinsics.go
helpers.go
```

For each retained construct, record:

```text
production
current evaluator handler
evaluation acts
value effects
environment effects
fault/return behavior
realization-only machinery
open design questions
```

Do not design `eval.spec` until this table is complete.

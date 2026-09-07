# Current Language Specimen

This note records the compact Nezha specimen after the residue-removal and boundary cuts. It is a checkpoint, not a future design proposal.

## Goal

Nezha starts from a small, runnable script language whose behavior can be studied before introducing an evaluation specification or ABI.

The core test is:

```text
engine + runner are sufficient to run a script with the prelude
```

Everything else in the repository is tooling, validation, documentation, or engineering support around that core.

## Core architecture

```text
engine/
  language implementation

runner/
  script-with-prelude execution facade

cmd/
  command-line clients and repository gates
```

The runner owns the current execution model:

```text
load grammar
load help metadata
load/evaluate prelude
create user environment enclosed by prelude
evaluate one user script
```

The `cleanroom` gate checks that this model does not depend on the rest of the repository.

## Grammar surface

The retained grammar describes ordinary script evaluation:

```text
program
statement
let_stmt
field
assign_stmt
if_stmt
while_stmt
match_stmt
pattern
return_stmt
expr_stmt
expr
pipe_expr
infix_expr
unary_expr
postfix_expr
call
index
access
primary
list_literal
func_literal
params
param_list
rest_param
block
literal
BINOP
```

Retained language forms:

```text
let
assignment
if / else
while
match
return
expressions
pipe
calls
index
field access
lists
function literals
rest parameters
blocks
literals
shapes
lexical scope
```

Removed grammar surface:

```text
package
select
default
package_stmt
select_stmt
select_case
select_default
hardcoded recv(expr) select case
```

## Token surface

Retained keywords:

```text
let
if
else
while
match
return
true
false
and
or
not
```

Retained operators:

```text
+
-
*
/
<
>
<=
>=
|>
.
=
```

Retained literal/value tokens:

```text
NUMBER
STRING
RUNE
SYMBOL
SHAPE
NAME
```

## Prelude surface

The prelude is the public language support layer. Current retained names:

```text
apply
print
println
first
rest
length
prepend
append
empty
type
equal
shape
is_error
to_str
shaped
abs
```

The prelude is intentionally small. It exposes list operations, shaped-value support, equality/type inspection, string conversion, basic output, and the evaluator-privileged `apply` helper.

## Registered runtime primitives

Evaluator-owned intrinsic:

```text
_apply
```

`_apply` is not a substrate primitive. It is evaluator-owned because applying a Nezha function evaluates Nezha bodies and must use the evaluator's ordinary call path.

Registered non-HAL runtime primitives:

```text
_first
_rest
_length
_prepend
_append
_empty
_type
_equal
_shape
_make_shaped_list
_to_str
```

Registered HAL primitive:

```text
_print
```

The primitive registry is implementation support for the prelude and runner. It is not a separate language surface for ordinary user code.

## Value universe

Retained semantic value kinds:

```text
number
boolean
rune
string
symbol
list
function
fault
return
shape definitions
lexical environment / authority / stack frames
```

Removed value/capability kinds include old host, file, process, network, channel, store, opaque, bytes, terminal, and exit/reset signal values.

## HAL boundary

HAL is retained as the host-operation boundary. The current HAL surface is deliberately minimal:

```text
HAL.io.print -> _print -> print / println
```

Capability profiles, async fault plumbing, paging/help output, resource close hooks, program arguments, and REPL service fields have been removed.

## Tooling boundary

Top-level command surface is reserved for script execution and author workflows:

```text
nezha <file.nez>
nezha fmt
nezha lint
nezha debug
nezha version
```

Repository coherence checks live under `gate`:

```text
nezha gate smoke
nezha gate enginesmoke
nezha gate treecheck
nezha gate distfmt
```

LSP, tags, profiling, observation/event-bus tracing, and Aiki-native test-file hooks have been removed. Formatter and lint remain because they are drift anchors for the grammar and source structure.

## Non-goals for this specimen

The current specimen does not include:

```text
modules
package declarations
import/use/export
FFI
hostnative capability surfaces
concurrency
select/channel/spawn/send/recv
interactive input
script loading from inside scripts
inexact math providers
profiling/observer instrumentation
editor service APIs
language-native test framework
```

These may return later only as explicit designs. They are not inherited by default from Aiki.

## Current interpretation

The current Nezha language is:

```text
a single loaded script evaluated with a compact prelude,
lexical scope,
lists and shaped lists as structure,
functions and rest parameters,
imperative control where retained,
pattern matching,
left-to-right expression evaluation,
pipe syntax,
and one HAL-backed output operation.
```

This is the starting specimen for later evaluation-character work.

## Post-boundary checkpoint

After the final boundary cleanup set, the core invariant is:

```text
only evaluator code evaluates Nezha bodies
substrate builtins are leaf functions
runner is the only script-with-prelude facade
engine + runner pass cleanroom without cmd/test/engineering/samples
```

The remaining registered substrate surface is a flat builtin table. `_print` is
still recorded as the single HAL-crossing host effect. `_apply` is evaluator-owned
and exists so the prelude can expose first-class application without creating a
runtime-to-evaluator callback.

Remaining design seams are no longer residue cleanup:

```text
Function.Body / Function.Env interface{} closure seam
whether _print/HAL survives long term
whether ReturnType should remain a value carrier
how shapes, match, and access decompose into evaluation characters
```

The next phase is evaluation-character inventory, not more broad removal.

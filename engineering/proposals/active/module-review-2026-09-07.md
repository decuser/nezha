# Module Review — 2026-09-07

This review records the post-cleanup boundary state. It is a checkpoint for
future evaluation-character work, not a request to keep cutting mechanically.

## Review standard

A module remains in the core when it directly serves one of these purposes:

```text
language definition
script execution with prelude
reference evaluation semantics
minimal runtime support for the prelude
drift control for the language specimen
```

A module belongs outside the core when it is primarily:

```text
editor service
repository gate wrapper
command-line presentation
historical Aiki capability residue
future feature placeholder
```

## Core boundary

```text
engine/   language implementation
runner/   host-facing execution convenience
cmd/      command clients and gate namespace
test/     validation
tools/gates are clients, not engine dependencies
```

The `cleanroom` gate is the practical boundary test: `engine + runner` must be
sufficient to run a script with the embedded prelude.

## engine/syntax

Status: retain.

Purpose:

```text
embedded grammar source
lexer/parser/AST construction
grammar-backed node names
syntax diagnostics/help metadata
```

Strength:

```text
grammar is the strongest authority in the repo
parser nodes are typed by grammar production names
downstream handlers validate against declared productions
```

Current seam:

```text
none requiring cleanup before evaluation-character inventory
```

## engine/formatter

Status: retain as syntax drift anchor.

Purpose:

```text
format grammar-valid syntax trees
protect grammar/parser/AST presentation drift
serve fmt/distfmt command clients
```

Boundary note:

```text
formatter is a syntax-layer tool, not an independent language layer
```

The command-side formatter adapters were removed; `engine/formatter` is the only
formatting authority.

## engine/semantics/value

Status: retain, with one recorded design seam.

Retained value universe:

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

Removed capability residue:

```text
bytes
opaque
file / lock
channel
store
process
endpoint/listener/datagram
terminal
exit/reset signals
```

Live seam:

```text
value.Function.Body interface{}  // always *syntax.Node
value.Function.Env  interface{}  // always *value.Env
```

This is not cleanup residue. It is the live closure-representation seam where
syntax, lexical scope, and evaluation meet. Defer until evaluation-character or
closure representation work.

## engine/semantics/evaluator

Status: retain; this is the reference realization.

Important boundary now achieved:

```text
only evaluator code evaluates Nezha bodies
runtime/substrate builtins are leaf functions
_apply is evaluator-owned, not substrate-owned
```

Important semantic areas for inventory:

```text
name resolution
let binding
assignment/update
if/while control
match discrimination and binding
function creation and call
return unwinding
fault propagation
pipe evaluation
shape construction/access
```

Current seam:

```text
function closure Body/Env opacity
possible future extraction of call/apply semantics into declarative characters
```

## engine/runtime/hal

Status: retain, slimmed.

Current contract:

```text
HasBuiltin(name, authority)
GetBuiltin(name, authority)
```

Removed from the old contract:

```text
Execute(name,args)
EvalContext callback path
capability profiles
async fault source
path-based authority policy
```

HAL now documents the compact text-I/O host crossings:

```text
_print -> HAL.io.print -> print/println
_read  -> HAL.io.read  -> read
```

Boundary note:

```text
_print and _read remain host-effect metadata
read is the single public input operation; input/readln aliases are absent
```

## engine/runtime/hal/substrate

Status: retain, compact.

Current shape:

```text
one flat builtin table
one output sink for _print
one host metadata map for HAL-crossing operations
fixed arity declared at registration
leaf signature: func([]value.Value) value.Value
```

Removed:

```text
five role-segregated registries
intrinsic/provider/host/service role vocabulary
mutex from old concurrency model
context-callable machinery
second Execute path
```

Remaining question, not urgent:

```text
whether GoRuntime should eventually become plain data rather than a struct
```

## engine/runtime/prelude

Status: retain.

Current public support surface:

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

Boundary note:

```text
prelude is Nezha source with explicit embedded-prelude authority
ordinary scripts do not receive underscore authority
```

## engine/runtime/primitives

Status: retain as runtime primitive catalog if it remains strictly current.

Current role:

```text
catalog live substrate primitives for validation/drift control
not a broad Aiki primitive taxonomy
```

Evaluator-owned `_apply` is not a substrate primitive and should not be listed as one.

## engine/runtime/help

Status: retain if it continues to validate the actual grammar/prelude surface.

Boundary:

```text
help metadata is documentation/drift control
help/doc are not language builtins
```

## engine/program

Status: retain as canonical program composition layer.

Current role:

```text
load grammar
load and evaluate embedded prelude
create user env enclosed by prelude
evaluate one script or expression
normalize execution faults
```

Boundary:

```text
program depends on HAL contracts, never the concrete Go substrate
program owns canonical prelude/user scope and authority assembly
help/doc catalog validation is not part of execution
```

## runner

Status: retain as thin top-level host facade.

Current role:

```text
read source files
choose the standard Go runtime substrate
delegate canonical execution to engine/program
```

Removed:

```text
program args plumbing
session/repl-shaped API
resource close no-ops
engine/runner package placement
```

Boundary:

```text
runner imports engine
engine does not import runner
cmd imports runner
```

## cmd

Status: retain as command clients.

Current top-level workflows:

```text
nezha <file.nez>
nezha -e <expr>
nezha fmt
nezha lint
nezha debug
nezha version
```

Repository gates:

```text
nezha gate smoke
nezha gate enginesmoke
nezha gate treecheck
nezha gate distfmt
```

Removed:

```text
lsp
tags
profile
native Nezha test command
flat gate commands at top level
```

## test

Status: retain as validation, not language.

Current role:

```text
canary/contract/fuzz/property tests
smoke and enginesmoke fixtures
invariant tests for layer boundaries and metadata
```

Boundary:

```text
tests may inspect language behavior
tests must not add runtime surface or prelude authority
```

## reference

Status: canonical executable witness corpus for the current specimen.

Boundary:

```text
reference programs explain and exercise the retained language deliberately
reference is not part of the engine + runner execution boundary
reference should not preserve removed features by nostalgia
```

## Recommended next work

Stop residue cleanup. Begin evaluation-character inventory against the current
specimen.

Do not cut the closure `interface{}` seam until the inventory shows what closure
representation must become.

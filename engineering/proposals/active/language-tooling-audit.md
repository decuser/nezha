# Language / Tooling Audit

This note records the post-cleanup boundary for the compact Nezha specimen.

## Retained language surface

The initial specimen keeps ordinary script evaluation with prelude, lexical scope,
functions, rest parameters, lists, shaped lists, field access, assignment,
conditionals, loops, match, return, calls, indexing, infix/unary operators, and
pipe.

## Retained runtime support

The runtime keeps HAL output for `print` / `println`, list primitives, value equality, type, shape construction/inspection, and `to_str`. The evaluator owns `_apply` so function application has one calling convention and substrate builtins remain leaf operations.

## Removed from prelude/runtime surface

The following are not language for the initial specimen and were removed from the
public prelude and registered runtime surface:

- `stack_limit`
- `inspect`
- `ord`
- `chr`
- `to_decimal`
- `to_number`
- `to_symbol`

## Profiling / observation disposition

The user-facing profiling command, CPU/trace top-level flags, semantic profiling
probe, observation network, dynamic coverage package, language-service observer
package, evaluator profiling hooks, and the engine-wide observer callback
interface are removed. Debug remains command-side phase inspection rather than
an engine event bus.

## Command surface disposition

The command surface now distinguishes ordinary language workflows from repository gates.
Top-level commands are reserved for script execution and author workflows such as
`fmt`, `lint`, `debug`, and `version`. Repository coherence checks live under
`nezha gate`, including `smoke`, `enginesmoke`, `treecheck`, and `distfmt`.

This keeps drift anchors available without presenting repository-maintenance
commands as ordinary language surface. LSP and tags tooling were removed rather
than retained as language-facing commands.

## Remaining audit targets

- final prelude surface versus language specimen
- debug command surface versus command-side phase inspection
- grammar/help/doc alignment for the retained specimen

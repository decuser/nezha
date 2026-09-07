# Language / Tooling Audit

This note records the post-cleanup boundary for the compact Nezha specimen.

## Retained language surface

The initial specimen keeps ordinary script evaluation with prelude, lexical scope,
functions, rest parameters, lists, shaped lists, field access, assignment,
conditionals, loops, match, return, calls, indexing, infix/unary operators, and
pipe.

## Retained runtime support

The runtime keeps HAL output for `print` / `println`, `_apply`, list primitives,
value equality, type, shape construction/inspection, and `to_str`.

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
package, and evaluator profiling hooks are removed. Debug tracing still uses the
small `engine.Observer` interface for explicit developer commands; that is
tooling, not language.

## Next audit targets

- language-service/LSP surface versus language surface
- smoke/engine-smoke tooling surface
- debug/tags/treecheck as development tooling

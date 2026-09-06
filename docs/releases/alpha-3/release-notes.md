# Aiki Alpha 3 Release Notes

Aiki Alpha 3 is a consolidation and realization release. The visible language remains deliberately small, while the implementation beneath it has become substantially more exact, efficient, observable, and internally constrained.

Alpha 3 also tightens agreement among the grammar, evaluator, standard library, help and documentation, command-line tooling, validation, and binary distribution. The release was validated on Linux, macOS, and Windows.

## Language and values

### Exact adaptive numbers

Aiki continues to expose one language-level `number` type with exact ordinary arithmetic. Alpha 3 substantially changes how exact numbers are realized internally:

- compact small-integer representation where possible;
- compact exact rational representation where possible;
- exact finite-binary representation where useful;
- arbitrary-precision rational escape when required.

These representation changes are invisible to Aiki programs. The language property is exactness; the physical representation is an implementation choice.

### Persistent adaptive lists

Lists remain immutable language values. Alpha 3 introduces adaptive persistent realization beneath that semantic surface, allowing list operations to use flat, frontier, and persistent/forked representations as appropriate without changing observable list behavior.

### Lazy Boolean operators

`and` and `or` are lazy and return operand values according to Aiki truth rules. The right operand is evaluated only when required.

Binary expressions otherwise retain Aiki's strict left-to-right evaluation rule; there is no conventional operator-precedence hierarchy.

## Mathematics

The math library is now divided by semantic role:

- `math/native` is the Aiki semantic authority for exact, explicitly bounded-work mathematics;
- bare `math` resolves to `math/native`;
- `math/ffi` provides provider-backed acceleration for the exact `floor`, `ceil`, and `modulo` surface;
- `math/approx` provides explicitly approximate one-argument `sin`, `cos`, and `sqrt`.

FFI acceleration is governed by an explicit acceleration surface rather than requiring an FFI module to duplicate unrelated native operations.

## Runtime realization

Alpha 3 contains extensive runtime realization work intended to reduce host cost without changing Aiki semantics.

Major areas include:

- reduced call, environment, and argument-frame allocation;
- conservative tail-environment reuse;
- lazy evaluation-context realization;
- reduced traversal and temporary allocation;
- exact-size AST collection where practical;
- cached immutable numeric-literal realization;
- lower-cost parser failure bookkeeping;
- lower-cost immutable string observation.

The guiding rule is that semantic properties remain stable while their physical realization may improve underneath them.

## Profiling and coverage

Execution profiling has been rebuilt around a single observation network rather than dispersed counters.

The profiling surface can account for:

- calls and caller/callee relationships;
- runtime-layer and boundary crossings;
- allocation counts and bytes;
- inclusive and exclusive host cost;
- realization categories and selected depth/hops.

Coverage now combines a static model of coverable program/AST sites with dynamic observation-network hits, avoiding a second independent runtime hit-accounting system.

## Modules, machine facilities, and extensions

Alpha 3 expands the supplied programming surface while preserving the small core language.

The shipped library now includes facilities for:

- files, paths, processes, signals, terminal and system services;
- networking and I/O;
- explicit mutable stores;
- bytes and bit operations;
- profiling;
- machine values;
- reusable state machines;
- self-hosted language facilities;
- graphics and turtle programming.

Generic state-machine and machine-value facilities were extracted from emulator work so they can be used by ordinary Aiki programs without making any particular machine architecture part of the core language.

## Help, documentation, and module invariants

Shipped modules now carry explicit `@summary` metadata. Module help and documentation expose canonical module identity when a bare alias resolves to a native authority.

Alpha 3 strengthens executable relationships among:

- module exports;
- signatures;
- help;
- documentation;
- native/FFI policy;
- declared acceleration surfaces.

This reduces the number of independently maintained descriptions of the same library surface.

## Command-line tooling

The command-line application has been reorganized so top-level help and command dispatch share one command registry.

Current subcommands include:

`check`, `fmt`, `distfmt`, `lint`, `lsp`, `profile`, `tags`, `treecheck`, `smoke`, `enginesmoke`, `debug`, `test`, and `version`.

Recursive target handling is centralized and supports shared path semantics, `.aikiignore`, and all-file operation where appropriate.

Validation is exposed through explicit levels rather than the former generic `make validate` target:

- `make validate-lang`
- `make validate-all`
- `make rigorous`
- `make distcheck`
- `make historycheck`

## Repository and distribution organization

Development material has been consolidated under `engineering/`, including proposals, profiling material, Git hooks, evidence, and the AI-assisted engineering method.

User-facing samples and editor support now have explicit homes, while `test/` and `selfhost/` remain first-class correctness surfaces.

The binary distribution contains the executables, shipped libraries, documentation, and samples. Source-only engineering and validation material remains in the source repository.

Experiments are maintained separately from the core repository in `aiki-experiments`.

## Documentation

Alpha 3 includes updated release documents:

- *Report on the Programming Language Aiki*;
- *This Is Aiki — The Aiki Alpha 3 Release*.

The language report describes the current grammar and complete supplied library surface. *This Is Aiki* describes the language from the outside in and records the architecture, design choices, and character of the Alpha 3 implementation.

Historical Alpha 1 and Alpha 2 release documents remain preserved unchanged.

## Compatibility

Alpha 3 is still an alpha release. The language model is coherent, operational, and specified, but compatibility guarantees for future releases have not yet been established.

The reference implementation has been validated on Linux, macOS, and Windows.

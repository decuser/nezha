# Aiki CLI Recursive Target Resolution — R1

## Status

ACTIVE — R1a, R1b, and R1c are gated. R1d implementation is complete; user gate pending. R1 is independent of Experiment 005.

## Purpose

Make Go-style recursive target syntax a coherent Aiki CLI capability rather than
a special case owned by one command. The motivating gap is that repository-wide
`./...` is recognized by `aiki test`, while an arbitrary subtree such as
`./experiments/005-pdp11-40/experiment/kd11_a/...` is not currently expanded.
Because path/target interpretation is cross-cutting tooling architecture, this
work is intentionally separate from PDP-11 Gate 1.

The design question is broader than adding one `test` special case: determine
whether Aiki has one target-selection language or several accidental per-command
implementations, then centralize only the semantics that are genuinely shared.

## Required semantic review

R1 must inventory every CLI surface that accepts source targets, files,
directories, modules/packages, or recursive scopes. At minimum the review should
locate the target-selection behavior used by test, lint, formatting, coverage,
profiling, documentation/checking commands, and any other developer command that
walks Aiki source trees.

For each command record:

- accepted target forms today;
- whether directories recurse and from which root;
- whether `./...` is recognized specially;
- whether arbitrary `<prefix>/...` is recognized;
- extension/source filtering;
- ignored/excluded paths;
- deterministic ordering and duplicate handling;
- repository-root/module-root resolution;
- invalid/empty-target behavior;
- whether the command consumes source files, modules/packages, or another unit;
- where target parsing and filesystem traversal are implemented.

Do not assume that all commands should receive identical post-expansion filtering.
The reusable abstraction may be target-pattern expansion rather than a universal
"find every `.ai` file" helper.

## Intended target language

Subject to R1a review, the intended common forms are:

```text
file.ai        exactly that target
path           existing command-appropriate directory target behavior
./...          recursive scope rooted at the repository/current command root
./path/...     recursive scope rooted at ./path
multiple args  deterministic union with duplicates removed
```

`...` is a recursive wildcard component, not merely the literal special string
`./...`. Existing valid command behavior must remain compatible unless R1a finds
and documents a contradiction.

## R1a inventory result

Source inspection shows that Aiki does not currently have one target-selection
language. It has several command-local implementations with overlapping but
non-identical syntax and traversal policy.

| Command/surface | Current accepted target forms | Recursive behavior | Filtering / exclusions | Ordering / dedupe | R1c disposition |
| --- | --- | --- | --- | --- | --- |
| `aiki test` | no args (`.`), `_test.ai`, directory, `./...`, multiple targets | directory recursively walks; only literal `./...` is rewritten to `.`; arbitrary `<prefix>/...` is absent | selects `*_test.ai`; test walk has no directory exclusions | `WalkDir` order inside each target; target-argument order across targets; no dedupe | migrate |
| `aiki test -cover` source model | same target list as `test` | separately walks each target; file target expands coverage scope to its containing directory | `.ai`, excluding `_test.ai`, `test/`, `ai/`, `.git`, and prelude | map dedupe then sorted before parse | migrate traversal substrate but retain coverage-specific scope/filter policy |
| `aiki lint` semantic walk | file, directory, suffix `<prefix>/...`, multiple targets | directories recurse; arbitrary suffix `...` already works | `.ai`, excludes `_test.ai` and parse-negative fixtures; hidden directories skipped; prelude optional | walk/argument order; no dedupe | migrate |
| `aiki lint` formatting check | file or suffix `<prefix>/...`, multiple targets | recursive only for suffix `...`; plain directory is treated as a file and fails | `.ai` in recursive mode; hidden directories and parse-negative fixtures skipped; prelude optional | walk/argument order; no dedupe | collapse onto same resolver as semantic lint while preserving formatting policy |
| `aiki fmt` | file or suffix `<prefix>/...`, multiple targets | arbitrary suffix `...` already works; plain directory is not a recursive target | `.ai`; hidden directories and parse-negative fixtures skipped | each argument processed independently; no cross-target dedupe | migrate |
| `aiki distfmt` | `.ai`/`.go` file or suffix `<prefix>/...`, multiple targets | arbitrary suffix `...` already works; plain directory is not a recursive target | `.ai`/`.go`; hidden directories skipped; parse-negative exclusion applies to Aiki source | each argument processed independently; no cross-target dedupe | migrate |
| `aiki tags` | `.ai` file, directory, `./...`, bare `...`, multiple targets | directories recurse; only whole-target `./...`/`...` special-cased | `.ai`; `.git` skipped | global clean-path dedupe and final sort | migrate |
| `aiki smoke` | no args (`.`), file-or-directory via `WalkDir`, `./...`, multiple targets | only literal `./...` special-cased; arbitrary suffix `...` absent | `*_smoke.ai`; no directory exclusions | walk/argument order; no dedupe | migrate |
| `aiki enginesmoke` | no args (`.`), `_engine.ai`, directory, `./...`, multiple targets | only literal `./...` special-cased; arbitrary suffix `...` absent | `*_engine.ai`; no directory exclusions | global dedupe and final sort | migrate |
| top-level `aiki file.ai` | exactly one program file plus guest argv | no target-tree semantics | runner/program semantics | n/a | do not migrate |
| `aiki profile` | exactly one program file | no target-tree semantics | profiling target is one runnable program | n/a | do not migrate |
| `aiki debug` | first positional filename only | no target-tree semantics | debug target is one source document | n/a | do not migrate |
| `aiki check --ffi-use` | exactly one program file | no target-tree semantics | follows module imports from one root program | n/a | do not migrate |
| `aiki treecheck` | `-root dir` option, no positional targets | recursively owns a repository-policy scan, not source-target selection | repository structural policy | checker-owned | do not migrate |
| `aiki lsp` | no filesystem target arguments | workspace/document protocol owns files | language-service policy | n/a | do not migrate |
| `aiki experiment` | creation name/destination semantics | not source-target selection | experiment creation policy | n/a | do not migrate |

There is also an older `engine/runner/smoke.go` collector with the same literal
`./...`-plus-`WalkDir` shape as the CLI smoke collector. It is not a second CLI
syntax authority that should survive migration; R1c must either route any live
consumer through the shared resolver or establish that it is dead/internal code
and remove or retain it explicitly.

### Architectural decision

The shared unit should be a **generic CLI filesystem target resolver whose
semantics are implemented in Aiki**, not the module/package catalog and not a
universal Aiki-source finder. Go may provide narrow filesystem and CLI adaptation
mechanics, but it must not become the authority for recursive-target or ignore
semantics.

The existing `engine/language/workspace` catalog resolves module imports and
language-service visibility. Making it interpret CLI recursive patterns would
mix two distinct authorities. Conversely, centralizing only the string rewrite
for `...` would leave the duplicated filesystem walks, ordering, deduplication,
and invalid-target behavior in place.

The repository already has the required language-side foundation. `lib/path`
implements portable path manipulation primarily in Aiki, while `lib/file`
provides host-backed filesystem capability. Normal module-registry initialization
scans shipped library roots independently of CLI target expansion, so importing a
target-resolution library does not create a target-selection bootstrap cycle.

R1b should therefore introduce an Aiki library (name/API to be fixed by the cut,
with `lib/target` as the working shape) that owns:

- classification of exact targets versus recursive `...` targets;
- `./...`, bare `...` where retained for compatibility, and arbitrary
  `<prefix>/...` normalization;
- recursive traversal decisions, including whether to descend into a directory;
- deterministic returned ordering;
- cross-target deduplication using normalized path identity;
- invalid/nonexistent target semantics;
- explicit base-directory interpretation;
- parsing and matching of repository target-exclusion policy; and
- explanation/reason information sufficient to say why a discovered target was
  included or excluded.

The host/runtime side should remain intentionally mechanical. Existing
`file.walk()` is not an adequate primitive for policy-aware resolution because it
performs the complete host walk before Aiki receives the result, preventing Aiki
from pruning ignored directories. Conversely, implementing recursion with
`file.list()` plus `file.stat()` would require an avoidable HAL crossing for each
entry. R1b should therefore add one narrow portable filesystem primitive, exposed
through `lib/file`, that returns the **sorted immediate entries of one directory
with basic directory metadata**. Aiki then performs recursion and policy decisions
without per-entry stat calls. The exact shaped result is a R1b API detail; it must
not encode lint/test/fmt policy.

The Go CLI still needs a narrow supported way to invoke the Aiki target library.
Affected commands are currently direct Go implementations, and creating ad-hoc
REPL source strings or duplicating Aiki semantics in `cmd/internal` is not an
acceptable bridge. R1b must establish or use a small Go-to-Aiki library-call
adapter whose job is only to load the library, pass typed values, invoke the
exported function, and translate its result/error. The adapter is plumbing, not
semantic authority.

Callers continue to own:

- whether a plain directory is accepted and, if so, whether it recurses;
- source suffix or fixture selection (`*_test.ai`, `*_smoke.ai`, `.ai`, `.go`,
  etc.);
- semantic notions such as coverage scope and module/import traversal; and
- presentation/output behavior.

Tool-specific **discovery exclusions** move out of command-local walkers and into
the shared Aiki target-policy input where they represent target selection. A
command-specific semantic filter remains local when it is about what the command
does with an already selected file rather than whether that file belongs to the
target set.

### Repository exclusion policy

R1a also adopts the direction of one visible repository policy surface rather
than separate `.lintignore`, `.fmtignore`, `.testignore`, and similar files. The
working form is a root-level `.aikiignore` interpreted by the Aiki target library.
It has one small matching language and supports a common section plus tool-specific
sections, conceptually:

```text
[all]
vendor/**
generated/**

[lint]
legacy/**

[fmt]
third_party/**
```

The first concrete repository policy use case for R1b is routine test isolation:

```text
[test]
experiments/**
```

Recursive repository test discovery should therefore omit `experiments/**`, while
an explicitly named experiment file or recursive experiment target remains eligible
because explicit selection wins. `test -cover` must apply the same target exclusion
to coverage-source discovery so execution and coverage scope cannot diverge.

The precise parser syntax and supported pattern operators are R1b decisions and
must remain deliberately smaller than full Git ignore semantics unless evidence
requires more. Initial design constraints are:

- one policy file governs Aiki target discovery; do not introduce a family of
  per-tool ignore files;
- no nested `.aikiignore` precedence system in the first implementation;
- recursive discovery honors applicable ignore rules **before descent**;
- an explicitly named exact file target is not silently discarded by an ignore
  rule; explicit selection wins, or a future strict mode must report the exclusion
  explicitly rather than silently doing nothing;
- structural host/repository exclusions that are true resolver invariants need
  not be user policy entries; and
- the resolver retains enough reason information for a future `--explain-targets`
  or equivalent diagnostic surface.

The lookup rule for a repository `.aikiignore` when invoked below the repository
root remains a R1b contract item. It must be explicit and testable rather than
accidentally coupled to Git or module-root discovery.

### Compatibility and migration contract

Later cuts must preserve existing valid syntax while adding arbitrary subtree
recursion where target-tree semantics exist. The common contract is:

```text
file                 exact filesystem target
plain directory      accepted/rejected according to the command's existing policy
./...                 recursive current/base scope
...                   retained only for commands that already accept it, unless R1b tests justify common acceptance
./path/...            recursive scope rooted at ./path
multiple targets      deterministic union; duplicate resolved paths appear once
```

Relative targets are resolved against an explicit resolver base supplied by the
CLI caller. The command may choose the process working directory as that base,
but the resolver and its tests must receive it explicitly rather than discover a
repository root implicitly. This preserves current CLI path meaning and avoids
entangling target expansion with module-root discovery.

R1b should prove the resolver first with focused unit tests and migrate the
smallest representative pair: one command that currently supports arbitrary
suffix recursion (`lint`) and one that does not (`test`). Both lint's formatting
and semantic passes must share the resolver, and `test` migration includes its
coverage-source discovery so `test` and `test -cover` cannot acquire different
target languages. That pair exercises both compatibility and the motivating
extension without broad rollout. R1c then migrates `fmt`, `distfmt`, `tags`,
`smoke`, and `enginesmoke`, and removes the superseded command-local walkers only
after parity evidence.

R1c also exposes the resolver's policy override consistently as `--all` on
applicable recursive-source commands. Normal resolution honors `.aikiignore`;
`--all` uses the same target syntax, ordering, deduplication, and command-specific
plain-directory/filtering semantics while bypassing ignore policy. This is the
CLI-owned exhaustive-selection mechanism. Makefile targets must not recreate
knowledge of ignored repository trees such as `experiments/`.

## Cut plan

### R1a — inventory, architecture review, and centralization decision

Status: GATED.

Non-behavioral. Survey every applicable CLI command and its target-expansion code.
Map duplicated walkers/parsers and behavioral differences. Decide whether there
is already an appropriate lower-level authority to extend or whether a new shared
target resolver is warranted. Specify compatibility, ordering, deduplication,
root-resolution, exclusion, error, and filtering contracts. Produce the bounded
migration plan for later subcuts. **Do not change target behavior in R1a.**

### R1b — shared target-resolution substrate

Status: GATED.

Implement the Aiki-owned target/policy abstraction selected by R1a, the narrow
filesystem-entry substrate and Go-to-Aiki invocation bridge it requires, and
focused unit/CLI evidence for files, directories, `./...`, arbitrary subtree
`...`, multiple targets, deterministic ordering, deduplication, invalid targets,
and ignore-before-descent behavior. Migrate only the minimum command surface
needed to prove the abstraction before broad rollout.

### R1c — applicable-tool migration

Status: GATED, exact command list determined by R1a.

Move each applicable tool onto the shared resolver while preserving its
command-specific filtering and prior accepted syntax. Delete duplicate target
walkers only after parity evidence exists. Every migrated command must gain
focused compatibility witnesses including arbitrary subtree recursion where that
semantics is applicable.

### R1d — reconciliation and repository gate

Status: IMPLEMENTATION COMPLETE; USER GATE PENDING.

Search for remaining accidental target-expansion authorities, reconcile help and
documentation, and establish thin validation orchestration around CLI-owned target
policy:

```text
make validate-lang   normal language/runtime repository gate; ordinary CLI policy
make validate-all    exhaustive applicable repository gate; CLI --all selection
make rigorous        validate-all plus expensive/deep conformance, selfhost, fuzz, etc.
```

The Makefile must not enumerate experiments or duplicate ignore/target semantics.
`validate-lang` is the routine language gate and therefore excludes experiment
tests through ordinary `.aikiignore`; `validate-all` explicitly requests exhaustive
CLI selection. `rigorous` is the strongest gate and includes `validate-all` before
its expensive checks. Run focused CLI regression and the appropriate authoritative
repository gates. Do not claim universal `...` support for commands that R1a
explicitly determines have no meaningful target-tree semantics.

## Invariants

- One recursive-target syntax must not create multiple filesystem authorities.
- Target expansion does not change language/module semantics.
- Existing `aiki test ./...`, file targets, and directory targets remain valid.
- Expansion order is deterministic and duplicate targets are not executed twice.
- Tool-specific source/module filtering remains owned by the appropriate layer.
- Repository-root resolution is explicit; tests must not depend on accidental
  current-working-directory behavior.
- No PDP-11 or Experiment-005 semantic code belongs in this tranche.

## Gate

Each subcut is independently testable. R1d leaves three explicit repository
validation levels: `make validate-lang`, `make validate-all`, and `make rigorous`.
There is intentionally no generic validation alias because it obscures which
validation population is being claimed. Additional focused commands are
implementation evidence, not extra user gates unless diagnosing a failure.

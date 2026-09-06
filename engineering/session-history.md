# Aiki Engineering Ledger

This file is the compact cumulative ledger of durable Aiki engineering outcomes.
It is not a session transcript or a cut-by-cut work log. Detailed rationale,
implementation plans, gate evidence, and superseded approaches belong in
`engineering/proposals/`, `engineering/evidence/`, or focused result documents.

## Language and semantic authority

- 2026-08-14 — Grammar became the sole syntax authority; parser-facing syntax
  knowledge was centralized around the grammar rather than duplicated tables.
- 2026-08-15 — Grammar analysis was centralized and post-grammar consistency
  checks were added.
- 2026-08-16 — `and` and `or` became lazy logical control operators. Aiki binary
  expression evaluation remains strict left-to-right rather than conventional
  precedence-based evaluation.
- 2026-08-17 — Engine authority and invariant surfaces were centralized and
  strengthened; public contract and portable-systems gaps were closed.
- 2026-08-18 — Native and FFI roles were separated and preservation evidence was
  established for the resulting authority boundaries.
- 2026-08-20 — Adaptive exact Number completed. `number` remains exact while its
  representation is hidden and adaptive; compact and arbitrary-precision
  representations are implementation choices.
- 2026-08-21 — Adaptive persistent List completed. Lists remain immutable while
  flat/frontier/fork realization is hidden and adaptive.
- 2026-08-21 — Immutable string observation was optimized without changing string
  semantics.
- 2026-08-21 — Environment, number-literal, parser-mismatch, and call-argument
  realization work completed without changing language semantics.
- 2026-08-29 — Math ownership was split by semantic role: `math/native` remains
  the explicit-work semantic authority, `math/ffi` accelerates exact
  `floor`/`ceil`/`modulo`, and `math/approx` owns intentionally approximate
  `sin`/`cos`/`sqrt`.
- 2026-08-29 — Exact FFI modulo was brought into semantic parity with native
  modulo for ordinary exact Number values, including arbitrary-precision
  rationals and matching zero-divisor faults.
- 2026-08-29 — Stdlib acceleration policy gained an explicit acceleration surface
  so a provider may accelerate a declared subset of an authority module without
  pretending to twin unrelated exports.

## Runtime realization and performance

- 2026-08-14 — Profiling and computational-visibility work established reusable
  runtime measurement workloads.
- 2026-08-16 — Profiler calibration established a baseline for later runtime
  optimization work.
- 2026-08-19 through 2026-08-21 — Runtime execution overhead was reduced across
  audit bookkeeping, fixed-width machine values, shaped faults, call sites,
  environment storage, immutable collections, string observation, literal
  realization, parser mismatch handling, and argument frames.
- 2026-08-21 — Runtime optimization synthesis was recorded in
  `engineering/runtime-optimization-results.md`.
- 2026-08-23 — The profiling network became the single execution-observation
  authority for counts, call realization, graph relationships, selected timing,
  and targeted allocation observation.
- 2026-08-23 — Coverage was reconciled to static coverable-site anatomy plus
  dynamic hits from the profiling network rather than a second runtime hit
  system.
- 2026-08-27 — Host-time attribution was admitted to the profiling network and
  authoritatively gated.
- 2026-08-27 through 2026-08-29 — Generic state-machine/host-native realization
  work reduced machine-emulation overhead and was incorporated into mainline.

## CLI, validation, and distribution

- 2026-08-14 — `treecheck` established explicit distribution-structure policy.
- 2026-08-14 — Repository provenance and release-preparation issues were repaired.
- 2026-08-15 — Release distribution became relocatable.
- 2026-08-15 — Language self-description/help surfaces were expanded and treated
  as checked contract surfaces.
- 2026-08-29 — Recursive source-target semantics were centralized in Aiki-owned
  `lib/target`, with a narrow Go bridge and `.aikiignore` policy.
- 2026-08-29 — Recursive commands gained arbitrary `<prefix>/...` selection and a
  consistent `--all` policy override where applicable.
- 2026-08-29 — Duplicate command-local walkers were removed; repository
  `treecheck` remains intentionally separate because it owns repository-policy
  traversal rather than language-source target selection.
- 2026-08-29 — Validation levels were made explicit: `validate-lang`,
  `validate-all`, and `rigorous`. The ambiguous generic `make validate` target
  was removed.
- 2026-08-29 — `cmd/aikiapp` became the reusable CLI application surface and
  `cmd/aiki` the thin core entry point.

## Engineering infrastructure

- 2026-08-14 — Negative-fixture tooling and distribution integrity checks were
  formalized.
- 2026-08-17 — Host/HAL capability gates and authority metadata were strengthened.
- 2026-08-18 — Native/FFI preservation witnesses and mutation evidence were
  established; current retained material lives under `engineering/evidence/`.
- 2026-08-21 — Historical session/bootstrap cruft began moving out of the release
  surface in favor of durable proposals, evidence, and compact cumulative records.
- 2026-08-29 — Repository engineering material was consolidated under
  `engineering/`: AI working method, proposals, evidence, profiling workloads,
  Git hooks, and cumulative engineering records.
- 2026-08-29 — `editors/` and `samples/` became explicit top-level surfaces;
  `extra/`, root `ai/`, root `hooks/`, and root `proposals/` were retired.
- 2026-08-29 — Obsolete bug-list authority was retired; unresolved implementation
  findings belong in `engineering/audit-findings.md` or a bounded proposal.

## Experiments and external consumers

- 2026-08-15 — The numbered experiment framework was established as a place for
  substantial integration/reconstruction work.
- 2026-08-15 through 2026-08-18 — Thompson 7094 regex and Four-Way Life exercises
  drove HAL, concurrency, profiling, and native/FFI refinements.
- 2026-08-19 through 2026-08-29 — PDP-11/40 reconstruction work exercised CPU,
  memory, bus, device, state-machine, host-native, profiling, and observer
  surfaces at system scale.
- 2026-08-29 — Numbered experiments were externalized to the independent
  `aiki-experiments` repository. Core Aiki retains generic mechanisms;
  experiment-specific providers, proposals, and drivers belong with the external
  experiment collection.

## Release and documentation milestones

- 2026-08-18 — Alpha 2 documentation reconciliation completed around the language
  and native/FFI contract surfaces then current.
- 2026-08-29 — Historical Alpha 1 and Alpha 2 documents were separated into
  frozen release directories under `docs/releases/`.
- 2026-08-29 — Post-Alpha-2 architectural work includes adaptive Number/List,
  runtime realization reductions, profiling/coverage consolidation, recursive
  target semantics, validation-level cleanup, math semantic separation,
  importable CLI application structure, and experiment externalization.
- Before the next release documentation is frozen, current language/help/docs,
  samples, distribution contents, and validation semantics require one coherence
  audit against mainline behavior.

## Current durable authorities

- Language syntax: `engine/syntax/grammar.ebnfx` and the implementation surfaces
  checked against it.
- Current architectural decisions: `docs/decisions.md`.
- Current engineering organization and working-method ownership:
  `engineering/README.md` and `engineering/ai/README.md`.
- Detailed design history and disposition: `engineering/proposals/`.
- Retained measurement/preservation evidence: `engineering/evidence/` and
  `engineering/profiling/`.
- Cross-project runtime optimization synthesis:
  `engineering/runtime-optimization-results.md`.
- Historical release documentation: `docs/releases/`.

## Recording rule

Add an item here only when it is durable enough to matter after the individual
work session is forgotten: an accepted semantic/architectural decision, a major
completed capability, an authoritative gate that changes project status, a
repository-ownership change, or a release milestone. Keep each item terse. Put
plans, investigation narrative, temporary failures, restart instructions, and
cut-level evidence in their proper proposal/evidence location instead.

# Aiki documentation

This directory contains the current documentation for Aiki plus frozen release
artifacts from prior milestones. The directory is shipped in the binary user
distribution as reference documentation. Contributor-oriented documents may name
source-tree paths such as `engine/`, `test/`, or `engineering/`; those paths resolve
in a source checkout and are not implied to be present in the binary distribution.

## Current documentation

- `docs/adding-to-aiki.md` — extending the implementation and standard library.
- `docs/debug.md` — debugging facilities and workflows.
- `docs/decisions.md` — current architectural and language decisions.
- `docs/hal.md` — host abstraction layer architecture and policy.
- `docs/profiling.md` — profiling facilities and usage.
- `docs/style.md` — repository and implementation style guidance.
- `docs/testing.md` — validation surfaces and testing policy.

Current architectural truth belongs in these documents, source, and tests.
Historical engineering execution material belongs under `engineering/`.

## Frozen release documentation

Historical release documents are retained unchanged under `docs/releases/`.
They describe the named milestone rather than current Aiki and must not be
silently updated to match later releases.

### Alpha 1

- `docs/releases/alpha-1/release-notes.md`
- `docs/releases/alpha-1/language-report.odt`
- `docs/releases/alpha-1/this-is-aiki.odt`

### Alpha 2

- `docs/releases/alpha-2/release-notes.md`
- `docs/releases/alpha-2/this-is-aiki.odt`

## Alpha 3 release candidates

The Alpha 3 documents describe the current release candidate and remain editable
until that release is frozen:

- `docs/releases/alpha-3/language-report.odt`
- `docs/releases/alpha-3/this-is-aiki.odt`

When Alpha 3 is released, this material becomes frozen historical documentation
under the same rule as prior releases.

Future release documentation should use the same per-release layout.

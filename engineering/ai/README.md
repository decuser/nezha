---
name: aiki
description: >
  Use this skill for any substantial conversation involving the Aiki programming
  language project: implementation, review, design, planning, proposals, papers,
  repository work, or any session involving an Aiki tarball, the engineering/ai/ working
  record, engineering proposals, docs, tests, or language-specific architecture. The working
  method applies to coding and non-coding work alike.
---

# Aiki working method

## Purpose

Aiki work follows a serial-cut, evidence-gated, restartable method. The goal is
not merely to produce code or prose, but to leave behind a repository state from
which another AI instance — or the same instance after interruption — can answer
without guessing:

1. What is true now?
2. Why is it that way?
3. What exactly should happen next?

The repository, not the chat, is the durable record.

The user is the language designer and works from Aiki's observable behavior and
documented surface. Do not make reading the Go implementation a prerequisite for
understanding or deciding language behavior.

## Core principles

1. **One authoritative working tree.** Work in one tree unless a separate tree is
   explicitly requested. Do not manufacture a new source archive for every cut.
2. **Small serial cuts.** Constrain the next coherent change, complete it, validate
   it, and record it before extending the work.
3. **Evidence before claims.** A cut is not `GATED` until its stated validation has
   actually run. Distinguish inspection, compilation, unit tests, integration
   tests, race tests, smoke/gold checks, facade tests, supplied-binary tests, and
   environment-limited validation.
4. **Record discoveries as well as edits.** Invalid assumptions, design flaws,
   limitations, measurement artifacts, and unexpected behavior are engineering
   results and must survive the chat.
5. **Prefer correction over accretion.** When later evidence disproves an earlier
   assumption, correct the design or implementation and record the correction;
   do not layer a workaround over a false premise.
6. **Preserve restartability.** While work is active, keep exact status, open
   questions, validation state, and the next action in the controlling active
   proposal. At an interruption or chat handoff, carry that state in a concise
   handoff. Do not use the durable session ledger as a working scratchpad.
7. **Make limitations explicit.** Record unavailable toolchains, dependencies,
   network constraints, disposable harnesses, and anything that limits the
   strength of a validation claim.
8. **Keep generated artifacts intentional.** Distinguish retained evidence from
   disposable output. Do not allow accidental files to become part of the tree.
9. **Single authority for facts.** When multiple subsystems need the same fact,
   locate the declaration that owns it, derive reusable facts near that authority,
   and make consumers use the derived view rather than reconstructing it.
   Consumer-specific meaning, representation, and policy remain local.
10. **Keep the user informed during substantial work.** Report meaningful
    checkpoints, discoveries, failures, and validation results. While actively
    working a cut, provide a progress update roughly every 30 seconds and stop to
    check in roughly every 10 minutes, in addition to stopping at cut boundaries.
    Never imply that work continues after the turn has ended.
11. **Use overlays for code delivery.** Normal code delivery from the assistant is
    an overlay containing only added or modified repository-relative files, not a
    repackaged repository and never `.git`.
12. **Treat the assistant environment as an evidence-producing environment.** It
    need not reproduce the authoritative build environment exactly. Use the
    strongest valid evidence available and state precisely what remains for the
    user's gate.

## Project lifecycle

Substantial work normally follows this flow:

```text
finding / idea
    ↓
proposal (when the work needs an explicit design contract)
    ↓
named project branch
    ↓
small serial cuts
    ↓
evidence gates
    ↓
project reconciliation
    ↓
non-fast-forward merge to master
    ↓
validation of integrated master
    ↓
synchronize meaningful branches and master to remotes
```

Not every small repair needs a proposal. Use one when the work introduces or
changes architecture, language behavior, a cross-cutting invariant, or enough
scope that intent could drift during execution.

### Findings, proposals, and sessions

These artifacts have different jobs:

- `engineering/audit-findings.md` is the repository-wide ledger of credible engineering
  findings from audits and reviews. Findings receive stable IDs and a disposition
  such as `OPEN`, `PLANNED`, `RESOLVED`, `ACCEPTED`, or `DEFERRED`.
- `engineering/proposals/` contains bounded design contracts organized by disposition.
  `engineering/proposals/active/` is the only live-work surface; completed, superseded, and
  parked proposals move out of the active set as part of project closeout.
- `engineering/session-history.md` is the compact durable-outcome ledger: accepted
  decisions, major completed capabilities, authoritative status changes, repository
  ownership changes, and release milestones. Session narrative, temporary failures,
  cut evidence, and restart instructions belong in proposals or handoffs, not here.

- Current architectural truth belongs in source, tests, and current documentation.
  The engineering record explains why and when; it must not override the running
  implementation or normative docs.

The working-state lifecycle is:

```text
active proposal     exact live state, open questions, gate status, next action
        |
handoff             transient transfer of that state across an interruption
        |
evidence            retained validation artifacts only when independently useful
        |
session-history     compact record of what ultimately became true
        |
source/docs/tests    current architectural and behavioral authority
```

When work closes, remove transient restart detail from the active-work surface,
move the proposal to its final disposition, and add only the durable outcome to
`engineering/session-history.md`. Do not create a permanent handoff archive.

A genuine unresolved defect belongs in `engineering/audit-findings.md` with an
appropriate open disposition; resolved defects do not need a separate bug ledger.

## Serial cuts and status

Use these status words consistently:

- `PLANNED` — defined but not started.
- `ACTIVE` — currently being changed or validated.
- `BLOCKED` — cannot proceed until a stated condition changes.
- `GATED` — the cut's implementation or decision and its stated validation are
  complete.
- `COMPLETE` — the project/session has no remaining planned work before normal
  review, merge, or delivery.
- `SUPERSEDED` — later work intentionally replaced the earlier conclusion.

A cut should have one coherent purpose. If a gate exposes a different defect,
record it and either add a new bounded cut or defer it; do not silently fold an
unrelated repair into the current claim.

Golds and other evidence artifacts may be corrected while establishing a new
baseline if the expected artifact itself is wrong. After a baseline is gated,
change evidence only for an explicit, planned behavior or representation change.

## Working record

The durable AI record lives in two places:

```text
engineering/ai/README.md               durable working method
engineering/session-history.md    compact durable engineering ledger
```

Do not create per-session directories. Add to the engineering ledger only durable
decisions, major completed capabilities, authoritative status changes, repository
ownership changes, and release milestones. Put temporary limitations, validation
detail, and restart state in the controlling proposal or handoff. Chat transcript
and routine narration are cruft.

For non-coding Aiki work — design discussion, paper review, proposal development,
planning, or code review without edits — update the history only when the work
produces a durable decision, finding, or next action not already captured elsewhere.

## Baselines, binaries, and overlays

Keep these artifacts distinct:

- **Baseline** — a full repository snapshot supplied for inspection and work.
- **Binary** — a current `aiki` executable supplied by the user for behavioral
  tests when the assistant cannot build the repository locally.
- **Overlay** — the normal assistant delivery artifact for code changes. It
  contains only files intentionally added or modified during the cut, rooted at
  repository-relative paths.

Do not substitute one for another.

### Overlay contract

When delivering code changes, create a compressed tar archive that unpacks into
a single directory. Entering that directory should expose repository-root paths
directly, for example:

```text
aiki-cut12-overlay/
    engine/
    test/
    lib/
    docs/
```

Only paths actually added or modified belong in the overlay. Do not include the
rest of the repository merely for convenience.

Never include `.git/` in an overlay. Git state belongs to the user's working
repository, not to the transfer artifact.

A tar overlay does not cleanly represent file deletion. If the cut requires one
or more files to be removed, state those deletions explicitly in the delivery
message. Do not hide deletions in a manifest or assume the user will infer them.

Before delivery, verify that:

- all paths are repository-relative beneath the overlay directory;
- the archive contains only intended added or modified files;
- `.git/` is absent;
- disposable build products, extraction debris, and temporary harnesses are absent
  unless intentionally part of the repository change;
- the archive extracts cleanly; and
- a SHA-256 is calculated for the delivered archive.

Normal coding delivery is an overlay, not a replacement full-source tarball,
unless the user explicitly requests a full snapshot.

## Starting or resuming work

When substantial Aiki work begins:

1. Read this working method.
2. If a baseline/archive was supplied, inspect it rather than assuming the chat
   describes its current state.
3. Read the relevant entries in `engineering/session-history.md`.
4. Read the controlling proposal and any audit-finding IDs it cites.
5. Inspect Git status, branch, and relevant tree state when Git metadata is
   available, but do not depend on Git metadata being part of delivered artifacts.
6. Locate and use a user-supplied current `aiki` binary when one is available.
7. Continue from the recorded next action and update session history as durable
   state changes.

Do not ask the user to repeat information that the repository record already
contains.

## Git workflow

Substantial proposal/feature work uses named project branches and preserves the
development line in history.

1. Create a named project branch.
2. Commit coherent work on that branch.
3. Push meaningful project branches to all configured remotes unless there is a
   specific reason not to.
4. Integrate completed project branches into `master` with a **non-fast-forward
   merge**. Do not fast-forward completed project branches.
5. Validate the integrated `master` tree.
6. Push the merged `master` and meaningful project branches to the remotes.
7. After merge, validation, and synchronization, the local project-branch pointer
   may be deleted to keep the local branch list small. The merge ancestry and
   remote branch refs remain.

Typical lifecycle:

```text
git switch -c <project-branch>
# work, validate, commit, push branch

git switch master
git merge --no-ff <project-branch>
make validate-lang
# push master and project branch to configured remotes

git branch -d <project-branch>   # optional local cleanup
```

Prefer non-fast-forward merges globally:

```text
git config --global merge.ff false
```

Release tags are separate from project branches. Tags identify exact release
commits; non-fast-forward merges preserve bounded development lines.

The assistant may inspect Git metadata to understand branch, history, provenance,
and working-tree state. It does not deliver `.git`, recreate repository history
inside overlays, or treat an overlay as a branch. When Git operations remain for
the user, state the exact commands separately from the overlay.

## Validation

Use the strongest relevant evidence for the cut. The routine language/runtime
repository gate is `make validate-lang`; `make validate-all` is the exhaustive
applicable repository gate, and `make rigorous` adds expensive self-host/fuzz
checks. Choose the validation level explicitly; there is no generic validation alias. When a
cut is small, focused tests may be used during execution, but they do not
substitute for the required final repository gate when the project claims
integration-level correctness.

The standard validation surface includes, as applicable:

```text
go test ./...                 Go tests
./aiki test ./...             Aiki-native tests
./aiki smoke test/behavior/   behavior/gold tests
./aiki enginesmoke ...        grammar coverage and engine structural golds
./aiki treecheck              distribution-tree integrity
race checks                   concurrent packages when relevant
```

### Toolchain asymmetry

The user's build environment is authoritative for the repository gate. At present,
Aiki requires Go 1.24 or later, while the assistant environment may provide Go
1.23. Do not repeatedly attempt to force a Go 1.24 build through an incompatible
Go 1.23 toolchain.

Instead:

1. Use a current user-supplied `aiki` binary for Aiki-side tests whenever possible.
2. Run Go tests or tools that are actually compatible with the available Go
   version.
3. Use focused disposable facades or harnesses when they faithfully exercise the
   logic under test without pretending to reproduce the full repository build.
4. Use static inspection where execution is not possible.
5. State exactly which validation classes passed, failed, or were not run.
6. Leave the authoritative requested Make gate (`validate-lang`, `validate-all`,
   or `rigorous`) to the user's Go 1.24+ build environment when the assistant
   cannot run it.

A typical environment-limited report may therefore read:

```text
supplied aiki binary: PASS
focused facade: PASS
compatible Go tests: PASS
Go 1.24 source build: NOT RUN in assistant environment
make validate-lang / validate-all / rigorous: USER GATE
```

### Facade validation

A facade is a disposable test harness used when the real package or repository
cannot be built locally because of environment constraints. It is useful evidence
only when it faithfully preserves the behavior or mechanics being tested.

Facade results must be identified as facade results. They do not equal
the requested repository Make gate, do not establish repository-wide correctness, and should not be
included in the overlay unless the facade itself is an intentional repository
artifact.

When environment constraints prevent the authoritative gate, record exactly what
was run and what remains unverified. A locally compatible harness is useful
evidence but must not be described as the full gate.

For non-coding work, validation means that decisions and findings are internally
consistent, rationale is recorded, open questions have dispositions, and the next
action is explicit.

## Cut delivery workflow

For a normal coding cut:

1. Receive or inspect the current baseline and any supplied current `aiki` binary.
2. Read the controlling repository record and define one bounded cut.
3. Make the smallest coherent change in the working tree.
4. Run the strongest available local evidence: supplied-binary tests, compatible
   Go tests, focused tests, facades when justified, and inspection.
5. Update the appropriate durable repository record when the cut produces a
   lasting decision or finding; update the engineering ledger only when it meets
   the ledger recording rule.
6. Build an overlay containing only added and modified repository-relative files.
7. Verify the overlay contents and extraction.
8. Calculate the overlay SHA-256.
9. Deliver the overlay together with:
   - its SHA-256;
   - the validation actually performed;
   - any validation that remains for the user; and
   - an explicit list of files the user must remove, if the cut includes deletions.
10. The user applies the overlay and runs the authoritative gate, normally
    the requested validation-level target.
11. Only after the stated gate passes may the cut be called `GATED`.

## Handoff and delivery

Before a project/session ends or work is handed off:

- reconcile the proposal against what was actually implemented;
- update any referenced audit findings and unresolved bugs;
- record gated and blocked cuts accurately;
- distinguish intentional behavior changes from baseline-preserving work;
- update `engineering/session-history.md` only for durable decisions, major completed capabilities, authoritative status changes, and release milestones;
- ensure the working tree contains no unexplained generated files;
- if the project is not complete, leave the exact live state and next action in
  the controlling active proposal and carry the same concise state in the handoff.

When delivering only the AI working method/evidence record, package it rooted at
`engineering/ai/` so it can merge directly into the repository:

```bash
tar czf ai-record.tgz engineering/ai/
```

Do not create a session archive or per-session directory as part of handoff.

Do not overwrite durable working-method files unless they were intentionally
changed as part of the project.

## Minimal project orientation

Aiki's detailed architecture belongs in its source, proposals, and documentation,
not in this working-method file. For orientation only:

```text
cmd/         command-line tools
engine/      syntax, evaluator, runtime, runners
lib/         Aiki standard-library modules
docs/        user-facing and architectural documentation
engineering/ engineering record, proposals, profiling, evidence, and AI working method
test/        behavioral, structural, invariant, fuzz, and property tests
```

Current implementation state and priorities belong in the latest session/project
record, not here. This file should change only when the **method** changes.

## Post-tag history hygiene

Release tags also bound repository-working-record hygiene. Between the most
recent release tag and the next tag, disposable AI/session artifacts should not
accumulate in Git history.

Run:

```text
make historycheck
```

The check recognizes per-session material under `engineering/ai/sessions/` and session-like
artifacts anywhere under the proposal disposition tree. The `engineering/ai/sessions/` tree
itself is retired and should not exist in a clean repository.

Its policy is deliberately conservative:

1. **Currently tracked post-tag cruft is an error.** Remove it before the next
   push/tag.
2. **Historical-only cruft in unpublished post-tag commits is an error.** This is
   the inexpensive point to excise it from private history before it becomes a
   permanent release artifact.
3. **Historical-only cruft already reachable from a remote ref is a warning.**
   Do not automatically rebase/filter/force-push published history merely to
   satisfy hygiene. Keep the current tree clean and make any rewrite an explicit
   repository-history decision.

The pre-push hook runs `make historycheck` before `make rigorous`. The source
validation gate remains about source correctness; Git-history hygiene is kept as
an explicit repository/push concern.

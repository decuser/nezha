# Working With AI on Nezha

This repository may use AI assistance, but AI is not an authority for Nezha.

The human maintainer owns design judgment, commits, releases, and acceptance of every change.

## Operating Contract

Work in small bounded cuts.

Before a cut begins, state the exact plan:

```text
allowed
not allowed
gate
```

Do not expand scope while implementing a cut. If a dependency or surprising failure appears, stop and report it before broadening the change.

Prefer overlays for generated work. An overlay must contain only files changed by the cut. Deletions are listed separately as explicit `rm` commands and are not hidden inside the archive.

Do not mix unrelated removals. A successful cut is narrow, reviewable, and easy to revert.

## Verification

A cut is not done because it looks plausible.

Use the repository gates appropriate to the change, usually:

```sh
make validate-lang
make validate-all
make rigorous
```

For language-surface changes, also run direct lexer/parser/formatter/evaluator checks with `nezha debug` and small `.nez` specimens.

## Design Discipline

Nezha is not Aiki renamed.

Nezha is a contained descendant used to test the evaluation-character hypothesis. AI assistance must preserve that discipline:

- do not reintroduce broad library, host, module, or runtime-service machinery casually;
- do not replace removed mechanisms under new names;
- do not hide construct semantics below the evaluation boundary;
- do not treat implementation convenience as a semantic argument;
- keep syntax, evaluation, and realization separate.

## Trust Rules

When uncertain, say so.

Do not invent repository state. Inspect the tree or ask for current output.

Do not claim a command, target, or file exists unless it has been checked in the current tree.

Do not paper over validation failures. Report the exact failing command and the smallest proposed fix.

## Expected AI Uses

Useful AI work includes:

- scanning for stale references;
- proposing bounded cuts;
- drafting overlays;
- summarizing validation output;
- preparing design notes;
- comparing intended invariants against current code paths.

Poor AI work includes:

- broad opportunistic cleanup;
- deleting tests to make gates pass;
- turning local failures into design changes;
- silently changing semantics;
- producing large rewrites without a reviewed remove/keep list.

The default mode is conservative engineering assistance, not autonomous development.

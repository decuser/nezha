# Proposals

`engineering/proposals/` is organized by **disposition**, not chronology. The top-level
directory is an index only; proposal documents belong in one of these
directories:

```text
active/       work currently being designed or executed
completed/    implemented or fully gated work retained as design history
superseded/   proposals replaced by a later controlling design
parked/       credible proposals not currently active
```

## Lifecycle

A new proposal starts in `active/`. Closing a project includes moving its
proposal to the correct disposition directory in the same closeout commit.

`active/` is therefore the authoritative live-work surface. While a proposal is
active, it should contain enough current state to resume without reconstructing
the work from chat: completed and pending cuts, unresolved questions, validation
status, known limitations, and the exact next action. A handoff may carry the
same information transiently, but it is not a second durable authority.

A proposal should not remain in `active/` merely because its document once used
the word `ACTIVE`. Repository state and validation evidence determine disposition.
On closeout, move the proposal to its final disposition and reduce any restart-only
material that no longer contributes useful design history. Record only the durable
outcome in `engineering/session-history.md`.

Do not create proposal files directly under `engineering/proposals/`. `aiki treecheck`
enforces this layout.

A parked proposal is not rejected. It is retained because the idea remains
credible but is not part of current work. A superseded proposal is retained to
show design history and should identify the controlling replacement where one
exists.

The compact `engineering/session-history.md` ledger should use current durable
concept names rather than preserve obsolete per-cut paths. Detailed historical
paths and design evolution belong in the proposal record itself.

# Proposal: Profiling Network

## Status

**COMPLETED**

## Motivation

Aiki's existing profiler is individually strong but structurally open. It
captures semantic counts, source-site attribution, number/call/list/environment
realization counters, Go MemStats totals, and pprof label correlation through
separate mechanisms with separate lifecycles.

The result is that we can see endpoints but not edges. A profiled PDP-11 run can
report a small number of Aiki calls and millions of Go allocations, but there is
no authoritative path from one to the other. We cannot answer: which calls,
through which boundary, caused which realization work?

The profiler is presently a federation of independent accountants. This project
replaces that federation with one profiling nervous system.

## Architectural invariants

### Single accounting authority

After migration, every Aiki profiling fact is owned by the profiling network.
There are no independent semantic, number, call, list, environment, site, or
boundary counter/probe systems maintained alongside it.

External facilities such as Go `pprof`, runtime tracing, and `runtime.MemStats`
remain external measurement sources. They do not become competing Aiki
accounting authorities.

### Capture once, derive thereafter

Every reported fact is either:

1. captured once as an irreducible observation, or
2. derived after execution from more primitive captured facts.

A fact must not be both independently counted and derivable. For example,
per-primitive call totals derive from call edges, boundary crossings derive from
classified edges, and file/module/layer rollups derive from node metadata.

### No compatibility distortion

Legacy accounting mechanisms are migration oracles, not design authorities.
They may prove parity or reveal a missing observation, but they may not dictate
the network's execution model, identity model, capture path, or storage design.

When migrating an old accountant, ask what real execution fact it observed. If
the fact is legitimate but missing, add the smallest native observation to the
network. If preserving the old category would distort the network, stop and
resolve the design question rather than encoding the distortion.

### Capture the graph that already exists

The network observes execution. It does not construct a new execution model to
make profiling easier. Caller identity is explicit execution context carried by
the evaluator/runtime, not reconstructed later from printable names, environment
frame names, or stack heuristics.

## Model

### Primitive observation vocabulary

The nervous system has four primitive forms:

```text
node identity       a runtime entity and immutable classification metadata
edge transition     direct caller -> callee execution transition
classified event    irreducible occurrence that is not derivable from edges
numeric measure     irreducible quantity such as bytes, copies, or slots
```

This vocabulary is deliberately smaller than the current collection of
accountant-specific structures. Old categories survive only when they correspond
to a real fact expressible in this vocabulary.

### Nodes

Nodes are runtime entities:

```text
function         Aiki user function or source-identified lambda
primitive        registered runtime/provider/intrinsic/host/service callable
root             measured execution root
```

Each node has a compact integer `NodeID`. Identity is never a bare printable
name. Function identity includes source identity (file/position/name where
available); primitive identity includes its registration/classification
authority and name. Anonymous functions at different source sites are distinct.
If an entity has no durable source identity, an explicit instance discriminator
may be used.

Node metadata supports post-run coordinate-system views:

```text
semantic         function / source site
structural       file / module / library
architectural    semantic / runtime / HAL / native / FFI / intrinsic
boundary         primitive role / crossing class
```

Identity and symbol/classification resolution occur at registration or first
encounter. The event hot path uses only integer IDs.

### Edges

Direct edges carry dynamic measures between caller and callee nodes:

```text
calls            count of call transitions on this edge
alloc_bytes      optional allocation measure attributed to this edge
time_ns          optional elapsed-time measure attributed to this edge
```

The foundational edge store is bounded, preallocated, integer-keyed, and
concurrency-safe. No map lookup, string construction, symbol resolution,
formatting, or per-event allocation occurs during steady-state edge capture.
Overflow is explicit evidence; it is never silently dropped.

### Events and measures

Not every real execution fact is naturally an edge. Examples include tail-frame
reuse, numeric realization choices, list backing allocation, and environment
storage transitions. Such facts are represented as schema-assigned integer event
or measure IDs and recorded into preallocated atomic storage.

Accountant-specific structs are not recreated inside the network. Reporting
views are derived from the network snapshot.

### Execution context

Caller identity is explicit. The network provides a compact cursor containing
`(network, current NodeID)`. A call from that cursor records the direct edge and
returns the child cursor for the callee execution context.

This avoids reconstructing caller identity from environment frame names and
works with ordinary calls, tail-call environment reuse, substrate dispatch, and
spawn by moving/updating the same explicit observation context at the real
execution boundaries.

## Depth and traversal

Depth zero records events/measures only and is the flat-accounting mode.
Depth one and above record direct caller->callee edges.

Higher-hop views are post-execution graph traversals over the captured direct
edges. This project does **not** claim path-sensitive dynamic call-chain counts
unless a later requirement explicitly justifies additional path capture. This
clarification preserves the capture/analysis separation and avoids inventing
more hot-path work merely to render an N-hop report.

```text
level 0          flat event/measure accounting
level 1          direct caller -> callee edges
level N          N-hop post-run traversal of the captured edge graph
full             unrestricted post-run traversal of the captured edge graph
```

## Off-path and on-path cost

The disabled path must remain a single nil-pointer/context check at each probe
site once migration reaches that site.

Level-zero steady-state recording is an atomic integer update into preallocated
storage. Level-one steady-state edge recording is an integer pair lookup in a
preallocated fixed-capacity table plus an atomic increment. No allocation or
symbol work occurs per event.

Dynamic node registration may allocate at first encounter; the assigned node ID
is then retained by the execution entity/context so subsequent events use only
integer identity.

## Capture vs. analysis separation

Execution captures compact state only:

```text
caller NodeID
callee NodeID
event ID
measure ID + delta
```

Analysis happens after execution:

```text
resolve IDs -> names / files / modules / layers / roles
build caller->callee network
compute derived node totals
roll up by file / module / library / layer / boundary class
produce traversal views
render reports
```

Analysis may be arbitrarily expensive without perturbing the measurement.

## Allocation attribution

Go allocation attribution remains an optional expensive dimension. Routine
level-one profiling first establishes causal call/boundary structure cheaply.
A diagnostic run may select one registered primitive and bracket only calls to
that primitive with `runtime.ReadMemStats`. The measured allocator delta is
attached to the already-existing caller->callee node identity; call population
continues to come only from the edge graph. No allocation table exists in
routine flat/edge profiling.

`runtime.MemStats` is process-global, so unrelated concurrent allocator traffic
can contaminate a bracket. The delta also describes the work inside the selected
primitive call and is therefore inclusive of nested work; recursively nested
selected calls can overlap and must not be summed as though they partition the
program's total allocation. Targeted attribution is a diagnostic witness, not a
replacement for the measured-interval substrate totals or Go allocation pprof.

## Legacy infrastructure disposition

The migration target is complete removal of independent Aiki accounting
mechanisms:

```text
REMOVE AFTER MIGRATION
  Counters profiling accounting
  SemanticSite / siteKey / siteCounter accounting
  NumberRealizationCounts accounting
  CallRealizationCounts accounting
  ListRealizationCounts / ListRealizationProbe accounting
  EnvRealizationCounts / EnvRealizationProbe accounting
  independent boundary-count accounting, if any

PRESERVE AS NERVE/RESULT INTERFACES, NOT ACCOUNTING AUTHORITIES
  SemanticProbe / AttributionProbe execution-facing observation contracts
  SemanticMeasurement and realization-count report DTOs derived from Network

PRESERVE AS EXTERNAL OR CLASSIFICATION INPUT
  primitives.Role / primitive catalog authority
  module/realization policy authorities
  pprof CPU/alloc profile integration
  runtime.MemStats / runtime trace facilities
```

During migration, an old accountant may temporarily run beside the network only
as a parity oracle. Such duplication is transitional and must be removed in the
same migration cut once its replacement is proven. No new permanent dual-write
path is allowed.

## Reporting target

The existing level-zero profile output remains behaviorally compatible while its
source of truth moves to the network.

Level-one reporting adds derived views such as:

```text
Boundary crossings
  semantic -> runtime       ...
  runtime  -> intrinsic     ...

Top primitives
  _equal                    ...
  _length                   ...

Caller -> callee
  equal -> _equal           ...
  ...
```

A boundary crossing is an edge classification, not a histogram of callee node
classes.

## Cuts

### Cut P1 — Nervous-system foundation

**Status: GATED**

Purpose: build the profiling network correctly before asking any legacy
accountant to migrate into it.

Scope:

- Add the neutral network substrate under `engine/observe`.
- Define source-safe node identity and immutable classification metadata.
- Define schema-assigned event and measure IDs.
- Define explicit execution cursors carrying current `NodeID`.
- Implement level-zero event/measure storage as preallocated atomic counters.
- Implement level-one direct edge capture as a bounded, preallocated,
  concurrency-safe integer table with explicit overflow evidence.
- Prove steady-state edge capture allocates zero memory.
- Prove identical printable names do not collapse distinct node identities.
- Do **not** migrate, duplicate, or alter any existing accountant in this cut.

Gate:

```text
identity collision tests                          PASS
flat event/measure capture tests                  PASS
caller-context/edge capture tests                 PASS
concurrent edge accounting                        PASS
steady-state edge capture allocations == 0        PASS
bounded edge overflow is explicit                 PASS
legacy accounting behavior unchanged              PASS by non-integration
make validate                                     USER GATE
```

### Cut P2A — Semantic/site authority migration

**Status: GATED**

Purpose: make the network the sole authority for semantic work and source
attribution before call-graph identity is introduced.

Scope:

- Define the semantic event schema from the real evaluator choke points.
- Capture unattributed level-zero semantic events directly in the network.
- Capture attributed semantic work once as `(site node,event)` observations and
  derive flat totals from those scoped observations; do not dual-write site and
  flat counters.
- Treat source sites as immutable network subjects with source/function/detail
  metadata; the registry resolves identity, while accounting remains solely in
  network storage.
- Route the still-unmigrated realization accountant through a temporary adapter
  that owns no accounting state. It may observe only realization facts.
- Remove semantic totals and source-site maps from legacy `Counters`; the old
  semantic/site accountant must not remain as a production write path.
- Preserve coverage as a separate test-tool concern; it is not a profiling fact
  in this proposal.

Gate:

```text
network semantic totals match established evaluator witnesses  PASS
attributed totals derive from scoped facts without dual write  PASS
source attribution behavior preserved                          PASS
legacy semantic/site accounting removed                        PASS
no permanent semantic dual writes                              PASS
make validate                                                   USER GATE
```

### Cut P2B — Execution identity and direct call edges

**Status: GATED**

Purpose: attach the P1 cursor/edge substrate to the actual evaluator call
boundaries without reconstructing callers from names or environments.

Scope:

- Carry network execution cursors through evaluator call environments.
- Register source-identified user-function nodes at first execution and retain
  their IDs in the active observation context.
- Register primitive nodes from the authoritative primitive catalog/role
  classification rather than from report-time name guesses.
- Record direct caller->callee transitions at `applyFunctionOwned` / substrate
  dispatch boundaries using only integer IDs on the steady-state path.
- Make per-primitive totals and boundary-crossing views derivable from edges; do
  not add independent primitive or boundary counters.

Gate:

```text
caller identity is explicit cursor state                         PASS
same-name functions/primitives remain distinct                   PASS
per-primitive totals derive from incoming edges                  PASS
boundary crossings derive from classified edges                 PASS
steady-state call edge capture allocates zero                    PASS
make validate                                                     USER GATE
```

### Cut P3A — Number realization accountant migration

**Status: GATED**

Purpose: move arithmetic-result and call-return numeric realization facts into
the network without preserving the legacy numeric accountant as a parallel
authority.

Scope:

- Register the intentional numeric realization vocabulary as network events.
- Record arithmetic-result representation, binary certified/fallback, promotion,
  and call-return representation directly at their existing evaluator choke
  points.
- Derive the existing number-realization report DTOs from the network snapshot.
- Remove all numeric fields, mutators, and snapshots from legacy `Counters` in
  this same cut.
- Do not change numeric representation or evaluator execution to accommodate
  profiling.

Gate:

```text
number arithmetic realization parity witnesses    PASS
number call-return realization parity witnesses   PASS
legacy numeric accountant removed                 PASS
no numeric dual writes                            PASS
make validate                                     USER GATE
```

### Cut P3B — Call realization accountant migration

**Status: GATED**

Purpose: reconcile call-realization facts with the execution graph and remove
the independent call accountant.

The migration deliberately retires the legacy `user_entry` and `substrate`
semantics. Those values counted entry through two evaluator implementation
branches rather than the actual target population; direct `_apply` dispatch and
tail transitions therefore made them incomplete as call facts. The network does
not preserve that distortion.

Scope:

- Define truthful `function` and `primitive` transition populations. At edge
  depth they are derived from caller->callee edges; at flat depth the same real
  transition is captured once as a classified event because edges are not
  retained. Never count both forms in one run.
- Include direct intrinsic dispatch (including `_apply`) and tail transitions in
  the target population without rerouting execution through profiler-specific
  machinery.
- Fix the discovered returned-tail-call cursor omission so both tail-return
  forms advance the same explicit profiling cursor.
- Represent irreducible tail-reuse and argument-store realization facts as the
  smallest native network events/measures. Argument-frame realization is
  captured as mutually exclusive new/reused x inline/promoted events; legacy
  aggregate report values are derived from those events.
- Keep argument arity as classified events and `args_evaluated` as a numeric
  measure.
- Remove call fields, mutators, and snapshots from legacy `Counters` in this
  same cut.
- Do not preserve any call bucket whose only meaning is an old evaluator branch
  rather than an execution fact.

Gate:

```text
flat and edge call populations agree on the same transitions   PASS
returned tail calls advance the explicit cursor                PASS
call realization report is network-derived                     PASS
legacy call accountant removed                                 PASS
no call dual writes                                             PASS
make validate                                                   USER GATE
```

### Cut P3C — List realization accountant migration

**Status: GATED**

Purpose: move persistent-list realization facts into native network events and
measures, then remove the list accountant state. Promotion, extension, growth,
and fork are irreducible physical append events; copied elements and allocated
backing slots are numeric measures emitted by the existing `List.Append`
realization result. No list node or synthetic graph edge is introduced.

Gate:

```text
selfhost list-realization parity witness       PASS
PDP list-realization parity witness            PASS
legacy list accountant state removed           PASS
no list dual writes                            PASS
make validate                                  USER GATE
```

### Cut P3D — Environment realization accountant migration

**Status: GATED**

Purpose: reconcile environment allocation/binding-pressure observations with
the network, remove the final profiling accountant state from legacy `Counters`,
and leave coverage as a separate test-tool concern. The surviving observations
are real environment realization events at existing choke points: physical
environment creation, logical call lifetime, compact-binding allocation, map
promotion, local new/update, and monotone local-binding high-water crossings.
The reported `*_local_max_*` buckets remain post-run derivations from those
high-water crossings; they are not independently counted. No environment node,
edge, or execution-model change is introduced merely for compatibility.

P3B-P3D share the following completion rule:

```text
intentional legacy realization facts reconciled   PASS
all surviving realization facts network-owned     PASS
old realization probes/counter state removed      PASS
no permanent dual writes                          PASS
make validate                                     USER GATE per cut
```

### Cut P4A — Graph production and reporting

**Status: GATED**

Purpose: expose the network as the causal profiling product without allowing
allocation attribution or legacy report compatibility to dictate graph capture.

Scope:

- Preserve ordinary `aiki profile` level-zero behavior and output.
- Add `-depth 0|N|full`; every positive depth captures the same direct edge
  authority, while N-hop/full traversal is post-execution analysis over those
  edges rather than path-sensitive hot-path accounting.
- Produce resolved node/edge analysis, top primitive populations, true
  boundary-crossing rollups, and file/module/library/layer call rollups.
- Add direct callers and depth-limited/full reachable-subgraph queries.
- Report edge-table overflow explicitly; a causal witness is not valid when
  edges were silently lost.
- Do not invent inclusive/exclusive “cost” from call counts. Such views are
  valid only for measures whose semantics support them.

Gate:

```text
PDP-11 operational profile produces causal graph  PASS
graph answers callers-of-primitive query          PASS
graph answers crossings-by-boundary query         PASS
depth-limited/full traversal works                 PASS
file/module/layer rollups derive from graph        PASS
edge overflow explicit                             PASS
level-zero output compatibility                    PASS
make validate                                     USER GATE
```

### Cut P4B — Targeted allocation attribution and profiling closeout

**Status: GATED**

Purpose: add the optional expensive allocation dimension on the network's
terms, remove obsolete pre-network reporting/probe compatibility scaffolding,
and close the profiling tranche.

Scope:

- Add optional targeted allocation attribution without making per-call
  `runtime.ReadMemStats` part of routine profiling.
- Attribute the expensive measurement to existing node/edge identity rather
  than creating a second allocation accountant.
- Reconcile/remove obsolete measurement/probe/report structures whose sole
  purpose was the pre-network accounting federation.
- Confirm the network is the single Aiki dynamic profiling authority.
- Reconcile proposal, profiling documentation, and session history.

Gate:

```text
targeted allocation witness uses network identity  PASS
routine level-0/edge capture unaffected            PASS
obsolete profiling federation scaffolding removed PASS
single Aiki accounting authority confirmed         PASS
make validate                                     USER GATE
```

### Follow-on tranche — Coverage without duplicate runtime accounting

Coverage is intentionally deferred until after P4 because it combines two
different authorities rather than belonging wholly to the dynamic profiler:

```text
static program anatomy     what could execute
profiling network          what did execute
coverage analysis          compare the two
```

The profiling network remains the sole authority for dynamic execution
observations. Coverage must not preserve a second runtime hit accountant merely
because its consumer is a test command. Conversely, the network must not be
distorted into owning the static coverable-site universe. File/module/library/
layer/function/source-site classifications should provide shared coordinates
where useful.

#### Cut P5A — Coverage model reconciliation

**Status: GATED**

Purpose: define the static coverable-site model and reconcile it with the
network's existing dynamic observations before changing coverage behavior.

Scope:

- Inspect the current `CoverHit(file,line)`/coverage path and its report
  denominator.
- Define a principled static universe of coverable Aiki sites from program/AST
  structure rather than forcing the network to manufacture never-executed
  nodes.
- Determine whether existing network observations identify every desired
  runtime coverage hit. If one genuine execution observation is missing, add
  the smallest native nerve ending on the network's terms; do not reproduce a
  coverage-specific counter inside the network.
- Define shared site identity/classification needed to compare static anatomy
  with observed execution by file/function/module/library/layer.
- Preserve grammar/engine structural coverage as a distinct concern unless
  evidence shows that it is the same execution question.

Reconciliation established in P5A:

- The legacy dynamic hit is exactly one entry to `Evaluator.Eval`, recorded as
  `file:line`; imported modules and spawned execution are seen because they
  execute through the same evaluator callback. Existing semantic-work events
  are not a complete substitute because they do not fire for every Eval entry.
- The legacy summary denominator is not a true static universe: once a file has
  at least one hit, every physical source line in that file is counted; files
  with no hit never enter the denominator. Comments, blank lines, and
  punctuation-only lines therefore depress the percentage of observed files,
  while wholly unobserved files are invisible.
- The new static authority is parsed Aiki anatomy in `engine/coverage`. A
  coverable source site is a source-backed non-`TERMINAL` AST node; line
  coverage is derived from distinct lines containing at least one such site.
  Static sites carry file, lexical function, module/library, and semantic-layer
  coordinates and exist independently of runtime NodeIDs.
- The one missing dynamic fact is represented by an optional `EvalProbe` nerve
  at `Evaluator.Eval` and a network-backed `observe.EvalCapture`. P5A proved
  the observation belonged in the nervous system; P5B subsequently refined its
  storage from generic scoped-event lookup to dense node-hit accounting after
  repository-wide coverage exposed unacceptable hot-path growth. The semantic
  fact is unchanged: one observation means one real AST evaluation entry.
- Grammar-production coverage from `enginesmoke` remains structurally distinct:
  it asks whether static grammar productions are exercised by structural
  specimens, not which Aiki AST execution sites ran in tests.

Gate:

```text
static coverable-site authority defined                 PASS
dynamic coverage observation source identified          PASS
no profiler/static-analysis responsibility distortion   PASS
current coverage semantics reconciled explicitly        PASS
make validate                                            PASS
```

#### Cut P5B — Coverage migration and enrichment

**Status: GATED**

Purpose: make runtime coverage consume the static program model plus profiling
network observations, eliminating duplicate runtime hit accounting and exposing
richer coverage views.

Scope:

- Replace evaluator `Counters`/`CoverHit` runtime coverage accounting with a
  query over the static coverable-site universe and network observations.
- Resolve each dynamic AST site's NodeID at first encounter and cache only the
  compact registry binding on that AST object. Steady-state Eval coverage must
  be integer-only network node-hit capture: no source-site construction, mutex,
  map lookup, generic scoped-event probing, or allocation per Eval entry.
- Remove the residual evaluator coverage accountant once parity/intentional
  behavior changes are established.
- Preserve or explicitly revise ordinary coverage output with witnesses.
- Add useful rollups such as file, function, module/library, and layer where the
  shared identity model supports them.
- Keep repository-role scope (supported libraries, experiments, examples,
  auxiliary material) as test-command path policy layered over generic coverage
  coordinates. Do not embed Aiki repository taxonomy in the profiling network or
  portable `engine/coverage` model. Experiments/examples remain visible evidence
  without defining the supported-surface headline.
- Keep coverage as analysis over two authorities: static anatomy plus the one
  dynamic nervous system.
- Align capture lifetime with execution lifetime: repository-wide test coverage
  must not keep one bounded network alive across unrelated test programs. Each
  independent test execution owns its own network capture; completed immutable
  observations are merged only in post-run coverage analysis. Do not solve
  capacity exhaustion by inflating a repository-global dynamic table.

Gate:

```text
no duplicate runtime coverage accountant                PASS
coverage derives from static anatomy + network hits      PASS
current/intentional coverage behavior witnessed          PASS
file/function/module/layer views supported as designed   PASS
steady-state Eval hit has constant bounded cost          PASS
network lifetime bounded to one test execution           PASS
repository-wide coverage does not progressively degrade PASS
make validate                                            PASS
```

## Completion audit

The completed tranche was audited after the final P5B validation specifically
for duplicate dynamic truth and migration debris. The audit established:

- `observe.Network` is the sole mutable Aiki execution-accounting substrate.
  Flat events, numeric measures, caller→callee edges, attributed/scoped events,
  targeted allocation edges, and dense AST node hits all terminate there.
- `ProfileProbe`, `SemanticCapture`, and `EvalCapture` are adapters/identity
  caches only. They own no independent execution totals. `NodeRegistry` and AST
  `NodeBinding` state identify observations; they are not accounting stores.
- Coverage owns static program anatomy and post-run rollups only. Each test
  execution has one bounded network lifetime; completed immutable hit views are
  merged after execution. There is no `CoverHit`, evaluator `Counters`, or
  repository-global runtime coverage counter.
- Number, call, list, and environment realization report structures are DTOs
  derived from network snapshots. Call/boundary/file/module/layer reports are
  likewise derived rather than independently counted.
- Go `pprof` and `runtime.MemStats` remain explicitly external measurement
  sources/correlation mechanisms, not parallel Aiki accounting authorities.
- Remaining uses of the word `Counters` in `value.FixedStore` describe the
  unrelated Aiki fixed-store representation. Historical references in session
  history/completed proposals remain historical evidence, not live machinery.
- Stale profiling tests/helper names that still said `Counters` were renamed at
  closeout so repository vocabulary reflects the final architecture.

Final user evidence: `make validate` PASS after the corrected P5B lifetime and
hot-path design. Repository-wide ordinary tests passed 2222/2222 in 32.341s;
coverage passed 2222/2222 in 45.721s (~41% elapsed overhead), with no progressive
slowdown or node-capacity failure. Experiment 002 coverage cardinality was
observed to vary slightly because its operator monitor contains an intentionally
asynchronous `spawn`/interrupt path; deterministic execution is still expected
to produce deterministic observation sets.

The profiling-network tranche is complete. Future changes should modify the
nervous system only when they introduce a genuinely new runtime fact, execution
transition, classification dimension, or measurement dimension that cannot be
derived from existing observations.

## What this proposal does not cover

- Boundary calling convention / fast ABI. That is a separate proposal informed
  by evidence from this network.
- Prepared execution surfaces, indexed locals, or evaluator restructuring beyond
  carrying the observation context required to report the execution already
  occurring.
- Value representation changes.
- OS-managed memory / mmap investigation.

## Working rule

> **Build one nervous system that captures execution truth. Existing accountants
> may prove migration parity, but they may neither survive beside it nor dictate
> its anatomy.**

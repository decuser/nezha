# Profiling and Computational Visibility

Aiki treats computational activity as part of the observable surface of the
language. Profiling therefore has two deliberately distinct views of the same
execution:

```text
Aiki source computation
        |
semantic engine activity
        |
Go substrate realization
```

The semantic view answers **what computation occurred**. The substrate view
answers **what this implementation had to do to realize it**. Aiki does not
collapse them into a synthetic cost number.

## Aiki semantic measurement

The `profile` module returns ordinary Aiki data.

```aiki
let profile = import("profile")

let work = () {
	1 + 2
}

profile.counts(work)
profile.measure(work)
```

`profile.counts(function)` measures a zero-argument call and returns counts for:

```text
arithmetic
comparison
call
iteration
index
send
receive
store_read
store_write
```

`profile.measure(function)` returns `[counts, sites]`. Each source-site entry is:

```text
[unit, count, file, line, column, function, detail, source]
```

The site data answers where the semantic work arose in the Aiki program. The
`detail` field identifies the called function or primitive when applicable.
Source attribution is opt-in so `counts` keeps the cheaper counter-only path.

`profile.timed(function)` measures one zero-argument computation with direct-call
host-time attribution enabled. It returns:

```text
[counts, elapsed_ns, timing_rows]
```

Each timing row is:

```text
[caller, callee, calls, inclusive_ns, self_ns, average_ns,
 file, module, layer, boundary]
```

Timing rows are ordered by descending self time. The call count is derived from
the profiling network's existing edge population; timing does not maintain a
second call counter. `elapsed_ns` brackets the complete measured function.
Setup performed before `profile.timed` remains outside the interval.

`profile.experiment(size, function)` measures `function(size)`. If `size` is a
list, every supplied value is measured independently. Aiki assigns no meaning
to the size value; the measured function does.

`profile.complexity(results)` classifies the observed growth of each semantic
unit across at least three positive experiment sizes. It compares constant,
logarithmic, linear, n-log-n, quadratic, and cubic families and reports an
empirical best fit plus a score. The result says what growth is consistent with
the observations; it is not a proof of asymptotic complexity.

## Command-line view

Profile an Aiki program with:

```bash
./aiki profile program.ai
```

For summary counters without source-site attribution:

```bash
./aiki profile --counts program.ai
```

The command also reports realization measurements for the same user-evaluation
interval. With `--counts`, adaptive Number profiling is separated into two
populations:

```text
Number arithmetic realization
  small_integer
  compact_rational
  binary_carrier
  big_rational
  binary_certified
  binary_fallback
  promoted_big

Number call-return realization
  small_integer
  compact_rational
  binary_carrier
  big_rational
```

These names describe hidden runtime realization, not Aiki numeric types. The
call-return view is separate so host-returned binary carriers can be measured
without conflating them with arithmetic results. Representation counting is
active only while semantic profiling is enabled; it does not add a global probe
to ordinary Number arithmetic.

The command also reports Go realization cost:

```text
elapsed
alloc_bytes
mallocs
gc_cycles
```

### Profiling network views

Level zero remains the ordinary flat profile. Direct caller/callee capture is
opt-in:

```bash
./aiki profile --counts -depth 1 program.ai
```

Any positive depth captures the same direct execution edges. Larger numeric
depths and `full` control post-execution traversal rather than adding
path-sensitive accounting to the hot path:

```bash
./aiki profile --counts -depth 2 -from parse program.ai
./aiki profile --counts -depth full -from parse program.ai
```

Useful graph queries include:

```bash
./aiki profile --counts -depth 1 -callers _equal program.ai
./aiki profile --counts -depth 3 -from evaluate -hops 2 program.ai
```

Positive-depth reports derive, from the same captured edges, top primitive
populations, caller/callee edges, true architectural boundary crossings, and
file/module-library/layer rollups. A boundary crossing is classified from both
ends of an edge; it is not a separately maintained callee-class counter. Edge
table overflow is reported explicitly.

The graph does not synthesize inclusive/exclusive "cost" from call counts.
When host-time attribution is explicitly enabled, inclusive/self views are
derived from measured dynamic call spans whose semantics support that
interpretation.

### Host-time attribution

Host-time attribution is opt-in:

```bash
./aiki profile --counts -time program.ai
./aiki profile --counts -time -depth 1 -top 30 program.ai
```

`-time` implies direct-edge capture when `-depth` is otherwise zero. The same
profiling network stores optional inclusive and self nanosecond measures on the
existing caller->callee identity. Reports include top timed nodes, timed edges,
file/module/layer/boundary rollups, and true timed boundary crossings.

Inclusive time contains nested calls and is intentionally non-additive. Self
(exclusive) time subtracts synchronous nested call spans and is the primary
rollup for asking where host time was spent in a single-threaded workload.
Spawned computations use independent dynamic timing stacks: concurrent branch
wall spans are not subtracted from their parent as synchronous child work.
Because concurrent wall intervals overlap, sums across concurrent branches may
exceed whole-run elapsed time.

Timing reads the host monotonic clock at dynamic call boundaries and therefore
adds substantial overhead on call-heavy programs. Counts remain the cheap first
pass; timed attribution is a diagnostic mode. Performance conclusions should
compare an un-timed control with timed runs and should prefer workloads long
enough for timer overhead not to dominate.

### Targeted allocation attribution

Allocation attribution is deliberately diagnostic rather than part of routine
edge capture. Select one registered primitive on a positive-depth run:

```bash
./aiki profile --counts -depth 1 -alloc-target _append program.ai
```

Only calls whose callee is that primitive are bracketed with Go
`runtime.ReadMemStats`. The resulting `alloc_bytes` and `mallocs` deltas are
attached to the same caller->callee identity already present in the profiling
network. Call counts are not duplicated, and ordinary level-zero/edge profiles
do not allocate allocation-attribution storage or perform per-call MemStats
reads.

This is an expensive, targeted witness. Go MemStats is process-global, so
unrelated concurrent allocator activity can contaminate a bracket. A bracket is
also inclusive of work performed inside the selected primitive call; recursively
nested calls to the same selected primitive can therefore overlap. Do not sum
these edge deltas as though they partition total program allocation. The normal
measured-interval `alloc_bytes`/`mallocs` totals and Go allocation pprof remain
complementary views.

Prelude loading, module-registry setup, lexing, and parsing of the entry program
occur before that interval. Module work triggered by the measured program is
inside it.

## Correlated Go profiling

A CPU profile can be recorded during the same execution:

```bash
./aiki profile --cpu run.cpu.pprof program.ai
```

The Go CPU profile is labeled at Aiki semantic-function regions and HAL
primitive regions. The labels are:

```text
aiki_layer
aiki_function
aiki_file
aiki_line
aiki_primitive
```

For example:

```bash
go tool pprof -tags ./aiki run.cpu.pprof
go tool pprof -tagfocus='aiki_function=append' ./aiki run.cpu.pprof
```

This makes it possible to relate an Aiki operation to the Go work observed
while realizing it. CPU sampling remains statistical; Aiki semantic counters
are deterministic for deterministic executions. They are complementary
measurements, not interchangeable ones.


An ordinary Go allocation profile can also be written:

```bash
./aiki profile --allocs run.allocs.pprof program.ai
```

Go's pprof label information is currently used only by CPU and goroutine
profiles, so Aiki does **not** claim source-label correlation for allocation
profiles. The measured-interval allocation counters answer how much allocator
activity occurred; the Go allocation profile answers which Go allocation sites
were hot.

A runtime trace can also be recorded:

```bash
./aiki profile --trace run.trace program.ai
```

## Concurrency

Semantic counters are safe for concurrent spawned computation. Spawned work
inherits the active measurement context. A measurement ends when its measured
function or program returns, however; profiling does not introduce task or
join semantics. If all spawned work is intended to be included, the program
must synchronize that work using its normal Aiki protocol, such as a completion
channel.

## Measurement overhead

When profiling is disabled, semantic probes are nil and do not construct source
records. `profile.counts` uses atomic counters only. `profile.measure` adds
source-site aggregation. Correlated CPU profiling additionally labels Aiki
function and HAL regions; label contexts are cached so repeated calls do not
rebuild label maps.

The profiling mechanism itself must be measured. Performance conclusions
should compare repeated baseline and instrumented runs and should not be drawn
from a single short execution.

## Sweeps

`make profilesweep` runs six Aiki-owned drivers covering evaluator loops,
proper tail recursion, list append, store/bits, regex/FFI, and synchronized
concurrency. Results are written beneath `profile-out/` by default (override
with `PROFILE_DIR=...`). Each workload emits the semantic/substrate text report,
a correlated Go CPU profile, and an ordinary Go allocation profile. See
`engineering/profiling/README.md`.

The purpose of the sweep is to establish a baseline before optimization.
Mechanisms such as adaptive Number realization, alternate list representation,
or concurrency tuning should be justified by measured semantic work and
substrate realization rather than by intuition alone.

## Design principle

> **Aiki exposes both what a computation does and what it costs to realize that computation.**

The semantic account belongs to Aiki. The realization account belongs to the
implementation. Correlating them makes the HAL boundary itself inspectable.

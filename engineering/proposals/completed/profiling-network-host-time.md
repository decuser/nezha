# Profiling Network Host-Time Attribution

Status: GATED — authoritative repository gate passed 2026-08-27
Date: 2026-08-27

## Purpose

Complete the profiling-network design's deferred `time_ns` edge measure so
call-heavy system workloads can answer where host wall time is actually spent.
The immediate acceptance workload is Aiki Experiment 005's real V6 `tmrk`
short transfer, but this is a general Aiki profiling capability and makes no
PDP-11 performance change.

## Contract

- The existing profiling network remains the sole accounting authority.
- Direct caller->callee edges remain the sole call-count authority.
- Host timing is an optional measure attached to the same dynamic call identity;
  it never increments or reconstructs call populations.
- Count-only and source-attributed profiling remain clock-free unless timing is
  explicitly requested.
- Timed calls record inclusive elapsed host nanoseconds and self nanoseconds.
  Self time subtracts synchronous nested call spans.
- Tail-call reuse ends the predecessor span and starts the successor as a sibling
  dynamic segment; it must not create false inclusive nesting.
- Isolated/spawned computations use independent dynamic timing stacks. Concurrent
  wall spans are not subtracted from the spawning branch as synchronous child
  work.
- Timing reports derive node, caller/callee, file, module/library, layer, and
  boundary views from the same network identities.
- Inclusive time is non-additive by definition. Concurrent wall spans can make
  summed self/inclusive rows exceed whole-run elapsed time.
- Timing overhead is diagnostic and must be documented/measured; routine
  `profile.counts` remains the cheap first pass.

## Surfaces

CLI:

```text
aiki profile --counts -time program.ai
```

`-time` implies direct-edge capture when no positive `-depth` is supplied.

Aiki:

```text
profile.timed(function)
```

returns:

```text
[counts, elapsed_ns, timing_rows]
```

Each timing row is:

```text
[caller, callee, calls, inclusive_ns, self_ns, average_ns,
 file, module, layer, boundary]
```

## Cut P3

Implementation only:

1. optional network timing storage;
2. dynamic per-branch timing stack;
3. evaluator/runtime call-boundary integration including tail calls and `apply`;
4. CLI and Aiki report surfaces;
5. timing semantics/overhead documentation and focused tests.

No Experiment-005 CPU, machine, UNIBUS, TM11, RK11, KL11, or VT52 behavior is
changed in this cut.

## Gate

Assistant-side evidence should include dependency-free Go tests/builds, Aiki
format checks, and static inspection. The authoritative Go 1.24+ repository gate passed on 2026-08-27. Timed
profiling was then used on the short `tmrk 0 / 100 / 1` workload to identify
actual host-time concentration before emulator optimization proceeded.

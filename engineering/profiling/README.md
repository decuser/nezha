# Profiling workloads

Small, durable workloads for exercising `aiki profile` and comparing runtime
realization across revisions. They are engineering diagnostics, not ordinary
language samples and not generated result storage.

## General sweep

The numbered workloads cover distinct runtime paths:

- `01-evaluator-loop.ai` — evaluator arithmetic/comparison/iteration;
- `02-tail-recursion.ai` — interpreted calls and proper tail recursion;
- `03-list-append.ai` — persistent list construction;
- `04-store-bits.ai` — store and fixed-width bit operations;
- `05-regex-ffi.ai` — provider-backed regular-expression work;
- `06-concurrency.ai` — channel/send/receive/spawn activity.

Run all six with `sweep.sh`. Generated profiles and reports belong outside the
repository or in a disposable local output directory.

## Durable focused witnesses

These workloads remain because they exercise implementation properties that are
still useful to measure directly:

- `selfhost-three-level.ai` — host interpreter -> selfhost -> selfhost;
- `adaptive-number-rational.ai` — compact exact-rational realization;
- `adaptive-number-host-math.ai` — approximate-provider return boundary and
  exact post-boundary Number realization;
- `adaptive-list.ai` — persistent-list frontier extension and historical fork;
- `string-observation.ai` — repeated mixed-width UTF-8 rune observation.

Historical acceptance measurements belong in the corresponding engineering
records and completed proposals, not beside these executable workloads.

## State-machine provider regression gate

`state-machine-intrinsic-gate.ai` and its shell wrapper exercise the mainline
`state_machine/native` and `state_machine/ffi` realizations directly.
`make intrinsicgate` checks semantic parity and retains the admitted provider's
>=10x performance expectation. It is a focused engineering regression gate, not
part of the ordinary validation hierarchy.

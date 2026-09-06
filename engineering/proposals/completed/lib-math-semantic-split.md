# Aiki Math Semantic Split — M1–M4

## Status

ACTIVE — M1 GATED; M2 GATED; M3 implementation complete, user gate pending. The handoff supplied by the user is authoritative for this tranche.

## Purpose

Make the math library express the distinction between semantic equivalence and approximate numerical service without turning implementation representation into language semantics.

Core invariant:

> Promotion may change realization; it must not create a second language construct.

The intended module shape is:

```text
math/native
    floor
    ceil
    modulo
    sin(x, terms)
    cos(x, terms)
    sqrt(x, iterations)

math/ffi
    floor
    ceil
    modulo

math/approx
    sin(x)
    cos(x)
    sqrt(x)
```

`math/ffi` means the same Aiki semantics with an accelerated/provider-backed realization. `math/approx` means an intentionally approximate numerical service. The latter must not promise `float64`, IEEE-754, or any particular future physical representation.

## M1 — Adaptive-realization architecture review — GATED

No semantic/code changes. Trace numbers/rationals, persistent lists, call argument storage/dispatch, and the HAL/FFI/provider boundary before authorizing math changes.

Result:

- Numbers are one semantic `Number` with adaptive small-integer, compact-rational, exact binary64-carrier, and arbitrary `big.Rat` realizations. Decimal source literals are exact rationals. Provider code receives ordinary `value.Value` / `*value.Number` and semantic accessors; no FFI-specific numeric payload exists.
- Lists are one semantic persistent list with private adaptive frontier backing. Promotion/growth/fork preserve logical persistence; shaped lists are the same list plus metadata.
- Ordinary user calls may use reusable inline argument frames (capacity two) or promoted spill storage; rest calls use durable argument storage and ordinary lists. `_apply` and `spawn` remain separate direct user-function paths but currently agree with ordinary minimum-arity/rest semantics.
- Providers receive ordinary `[]value.Value`; boundary crossing itself performs no numeric realization conversion. Provider registration does not encode arity; implementations check it locally.
- `math/ffi.floor` and `ceil` already use exact arbitrary-precision semantics. `_modulo` is narrow only because its provider implementation locally rejects non-integral operands.
- Approximate trig/sqrt providers explicitly convert to host float, compute approximately, then return an ordinary Aiki `Number` denoting the returned finite value exactly. Approximation belongs to the operation contract, not to a second Number type.

Conclusion: the intended native/ffi/approx split fits the existing adaptive-realization architecture cleanly. No provider-boundary redesign is justified before the math cuts.

Follow-on recorded outside this tranche: investigate unifying `_apply` user-function dispatch with ordinary call entry so arity/rest/environment logic cannot drift again. Do not fold that work into M2–M4.

## M2 — Approximate-service split — GATED

Add `math/approx` and relocate the current one-argument provider-backed `sin`, `cos`, and `sqrt` surface there.

Constraints:

- no numerical redesign;
- no new number type or approximation/provenance bit;
- document approximation as the service contract, not host representation;
- remove `sin`, `cos`, `sqrt` from `math/ffi`;
- add only the module policy/authority/docs/tests required to make the new module a truthful shipped module;
- do not migrate unrelated production consumers yet;
- do not reclassify `math/ffi` as a portable acceleration twin until M3 actually establishes parity.

Gate: focused Aiki math/module tests with the supplied current binary, compatible focused Go tests when available, lint/format/invariant checks applicable to touched surfaces. Stop at the M2 boundary for user gate.

## M3 — Exact FFI parity — GATED

Make `math/ffi` a genuine semantic acceleration twin of `math/native` for exactly:

```text
floor
ceil
modulo
```

In particular, remove `_modulo`'s integral-only restriction and implement the full exact-rational Euclidean semantics, including negative divisors and division-by-zero fault behavior, using semantic Number accessors/results rather than representation-specific shortcuts.

Strengthen native/FFI parity tests across compact integers, compact rationals, arbitrary-size integers/rationals, signs, zero-divisor faults, and type/domain faults. Once parity is established, update stdlib policy so `math/ffi` is classified as an FFI realization whose semantic authority is `math/native`.

Implementation result:

- `_modulo` now implements the exact rational Euclidean operation over semantic `Number` accessors and returns through `NewNumberFromRat`; the integral-only restriction is gone.
- zero divisor now preserves the language-level `division by zero` fault text used deliberately by `math/native`.
- native/FFI Aiki parity witnesses cover compact rationals, promoted integers/rationals, all modulo sign combinations, zero divisor, and type faults.
- stdlib policy now supports an explicit `AccelerationSurface` for a truthful accelerated subset of a broader native authority. Empty continues to mean the complete authority surface for existing twins. `math/ffi` declares exactly `floor`, `ceil`, and `modulo` against `math/native`; executable invariants require that subset's exports, signatures, help, and docs to agree.
- This policy extension was required because the previous invariant modeled only whole-module twins and would otherwise reject the intentionally narrower exact math FFI surface.

## M4 — Consumer and documentation reconciliation — IMPLEMENTATION COMPLETE; USER GATE PENDING

Audit all Aiki imports/usages and deliberately choose among:

- `math/native` for explicit-work/reference semantics;
- `math/ffi` for exact accelerated `floor`/`ceil`/`modulo`;
- `math/approx` for one-argument approximate trig/sqrt service.

Do not mass-rename imports. Reconcile current-facing library documentation, help, policy/invariants, release/architecture prose where still current, and tests/examples whose existing apparent behavior depended on extra arguments being ignored by `math/ffi.sin/cos/sqrt`.

Gate the reconciled tree with the strongest available focused evidence, then leave the user the authoritative repository validation level required by current project procedure.

Implementation result:

- audited every executable `math/native`, `math/ffi`, and `math/approx` import in the tree rather than mass-renaming; exact floor/modulo consumers remain on `math/ffi`, explicit-work/reference consumers remain on `math/native`, and approximate trig/sqrt consumers use `math/approx`;
- moved the adaptive-number host-math profiling witness to `math/approx` while preserving its purpose as a provider-return binary-carrier witness;
- renamed the approximate spirograph sample from `spirograph-ffi.ai` to `spirograph-approx.ai` so the sample name and import surface agree;
- updated current-facing Design Decision D1, adaptive-number evidence prose, and profiling documentation to distinguish exact `math/ffi` acceleration from approximate `math/approx` service without rewriting historical session/proposal records;
- assistant-side `aiki16.5` lint and `distfmt -n` are clean for reconciled Aiki files; exact modulo pipeline/HOF consumers execute with expected output; the `math/approx` profiling witness executes successfully. M3 provider semantics were gated by the user before M4 began.

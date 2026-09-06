# PDP-11/40 KY11-D Programmer's Console — Experiment 005 Cut 1

## Status

COMPLETED — Cut 1 implementation, focused console evidence, and later authoritative full-repository gates are complete.

Parent proposal: `engineering/proposals/active/pdp11-40-reconstruction-005.md`.

## Purpose

Reconstruct the PDP-11/40 programmer's console as the actual DEC component,
KY11-D, and establish its direct black-box connection to the console-control
surface of the actual processor, KD11-A.

The user-facing term **console** may be used for KY11-D and **CPU** for KD11-A,
but source, contracts, probes, and machine wiring retain the hardware identities.

Cut 1 does not implement instruction execution, core memory, a UNIBUS transfer
engine, or peripherals. It establishes the panel, the processor-side console
control state, their connection, a visible terminal presentation, and neutral
observability.

## Authority

Primary authority:

- DEC-11-H40SA-B-D, *PDP-11/40,-11/35 (21 inch chassis) System Manual*,
  Chapter 3, especially section 3.2, Tables 3-1 and 3-2, and section 3.5.2.

Cross-checks:

- DEC, *PDP-11/40 Processor Handbook* (1972), console operation;
- DEC, *PDP-11 Bus Handbook* (1979), where processor console operations become
  bus transactions;
- DEC, *PDP-11 UNIBUS Design Description* (1979), where INIT and bus semantics
  are implicated.

Experiment 004 is not an authority and no PDP-11-specific source or tests from it
are admitted.

## Architectural correction from the first Cut 1 candidate

The first candidate placed R(ADRSC), manual EXAM/DEP sequencing, and START/CONT
interpretation inside a generic console object. The DEC manual states that the
KY11-D is the programmer's panel and that console operation interacts with the
processor through processor microprogram control. Therefore those semantics are
processor-owned.

The corrected split is:

```text
operator
   |
   v
KY11-D
   |  direct console/control interface
   v
KD11-A console-control state
   |
   +---- future instruction execution       Cut 4
   |
   +---- future UNIBUS transactions         Cut 2 + Cut 4 integration
```

KY11-D does not become a UNIBUS master. EXAM/DEP are operator actions received by
KD11-A; the processor performs the required bus operation when the UNIBUS and
processor execution machinery exist.

## Black-box rule

Every reconstructed hardware module owns a directory and declares four surfaces:

```text
inputs
outputs
state
probes
```

A module changes only its own state. Functional interaction occurs through named
ports/signals. No module reaches into another module's state.

Observation is orthogonal to functional wiring. Probe/trace consumers may observe
structured events but may not influence state transitions or results.

## KY11-D black box

### State

- OFF / POWER / LOCK switch position;
- ENABLE / HALT switch position;
- 18-bit Switch register;
- 18-bit ADDRESS display driven by processor output;
- 16-bit DATA display driven by processor output;
- RUN, PROC, BUS, CONSOLE, USER, and VIRTUAL indicators driven by processor
  output where the corresponding processor state is admitted.

### Inputs

Operator actions:

- power switch position;
- ENABLE/HALT position;
- Switch register value;
- LOAD ADRS;
- EXAM;
- DEP;
- START;
- CONT.

Processor presentation inputs:

- ADDRESS display value;
- DATA display value;
- status-indicator values.

### Outputs to KD11-A

- power/control state changes;
- ENABLE/HALT state;
- current Switch register value;
- momentary LOAD ADRS, EXAM, DEP, START, and CONT actions.

### Probes

Structured events include at least:

```text
ky11d.power
ky11d.mode
ky11d.switch
ky11d.action
ky11d.display
```

## KD11-A console-control black box admitted in Cut 1

This is not the complete processor of Cut 4. It is the processor-owned control
surface needed to make the KY11-D boundary truthful.

### State

- execution/control observation: console, running, or waiting;
- R(ADRSC), the processor's preserved console address;
- previous manual console operation for EXAM/DEP sequencing;
- ENABLE/HALT input state;
- pending console request, if any.

The Cut 1 shell may not claim instruction execution occurred. Running/waiting
state is established only by an admitted processor observation; START/CONT may
request execution but cannot synthesize it.

### Inputs from KY11-D

- ENABLE/HALT state;
- Switch register value accompanying an operator action;
- LOAD ADRS, EXAM, DEP, START, CONT;
- power/control changes where relevant to the documented contract.

### Outputs

To KY11-D:

- ADDRESS/DATA display updates that are established by admitted console-control
  state;
- status observations when known.

To future machine components:

- INIT request;
- run/resume/single-instruction/halt request;
- DATI/DATO semantic request for future UNIBUS execution.

### Probes

Structured events include at least:

```text
kd11a.control
kd11a.state
kd11a.console_request
kd11a.console_complete
```

## State-machine rule

State machinery is explicit and first-class. Both device control surfaces expose
a mostly pure transition relation of the form:

```text
current state + event -> next state + result
```

The mutable device wrapper performs only four jobs: read current state, apply the
transition, store returned state, and emit observation. Observers are not inputs to
the transition relation. Storage layout is not part of the transition contract.

KY11-D is a small physical/operator state machine. It does not interpret CPU
semantics. Momentary controls may therefore be state-preserving transitions that
emit a signal.

KD11-A owns the console-control state machine. Its console-control state is one
explicit value containing execution observation, ENABLE/HALT, R(ADRSC), prior
manual operation, and pending request. It interprets KY11-D actions and produces
processor or bus requests. In Cut 1, requests requiring later components remain
explicit dependencies.

Examples:

```text
KY11-D CONT + KD11-A observed running
    -> no effect

KY11-D CONT + HALT + KD11-A console control
    -> single-instruction request

KY11-D CONT + ENABLE + KD11-A console control
    -> resume request, no INIT

KY11-D START + HALT + KD11-A console control
    -> INIT request, remain under console control

KY11-D START + ENABLE + KD11-A console control
    -> INIT + run-from-R(ADRSC) request
```

## Manual addressing

With KT11-D absent, console physical-address projection follows the documented
base PDP-11/40 rule. LOAD ADRS supplies the Switch-register address to processor
console control and R(ADRSC).

EXAM/DEP are word oriented. Ordinary odd addresses ignore bit 0 for the transfer.
Repeated same operations increment by one word; alternating EXAM and DEP do not
increment. Processor internal-register console addresses are the documented
exception and increment by one.

These rules belong to KD11-A console control, not KY11-D panel state.

## Visible operator surface

Cut 1 includes a terminal frontend for KY11-D. It is a host presentation of the
emulated panel, not a debugger and not a replacement command architecture.

It must visibly render:

- ADDRESS;
- DATA;
- Switch register;
- OFF/POWER/LOCK;
- ENABLE/HALT;
- RUN, PROC, BUS, CONSOLE, USER, VIRTUAL.

Host commands such as `exam` or `cont` mean "operate the corresponding KY11-D
control." Any later debugger commands must be visibly separated from this panel
surface.

The frontend attaches outside the KY11-D model. Replacing the terminal frontend
with another presentation must not change machine semantics.

## Observability

The Aiki source provides the model: semantic observers receive events and must
not affect results; probes receive measurements separately. Experiment 005 uses
that same separation.

Cut 1 implements structured event observation sufficient to trace KY11-D and
KD11-A console-control transitions. Human-readable rendering is a consumer of
those events, not the event format itself.

No observer may pace, authorize, suppress, or alter a hardware transition.

## File ownership

Cut 1 uses device-owned directories:

```text
experiments/005-pdp11-40/experiment/
    machine.ai
    main.ai
    observe/
        observer.ai
    ky11_d/
        state_machine.ai
        panel.ai
        terminal.ai
        ky11_d_test.ai
    kd11_a/
        state_machine.ai
        control.ai
        kd11_a_control_test.ai
    console_integration_test.ai
```

Do not return to a flat collection of device files as later components accrue.

## Emulator supervisor and device-view workspace

The reconstructed KY11-D is not the emulator supervisor. Experiment 005 provides
`aiki-pdp` as the modern host supervisor/debugger. It owns the single authoritative
machine instance and a localhost observation/control broker.

Device presentation is a separate client layer:

```text
aiki-pdp (machine owner)
    |
    +-- coalesced latest snapshots --> tabbed workspace
    |                                  [KY11-D] [KD11-A] ...
    |
    +-- coalesced latest snapshots --> detached device view
    |
    +<-- declared KY11-D panel commands only
```

The default workspace selects KY11-D. Any admitted device may be opened as the
full set, a subset, a single tabbed view, or a detached terminal. Detaching changes
only presentation; all views observe the same machine instance. Output-only views
redraw in place with ANSI cursor/screen control rather than scrolling. Scrolling
trace/log output remains a distinct surface.

The workspace uses raw TTY input for tab selection and interactive KY11-D input.
The supervisor provides editable persistent command history and host controls:
`^T`/`status`, `^C`/`break`, and `^D`/`quit`. These are emulator controls and are
not represented as KY11-D hardware actions.

Observer delivery uses a latest-snapshot mailbox per client. Stable snapshots are
not republished, and intermediate snapshots may coalesce, so a slow or hidden
view cannot pace the emulated machine. This mechanism was reimplemented for 005;
Experiment 004 was consulted only as historical usability evidence.

## Gate

Cut 1 becomes GATED only when all of the following are true:

1. DEC-derived KY11-D physical state and controls pass focused tests;
2. R(ADRSC), START/CONT, EXAM/DEP sequencing, and bus/CPU requests are owned by
   KD11-A control rather than the KY11-D panel;
3. KY11-D and KD11-A interact only through the declared connection surface;
4. operations requiring UNIBUS or instruction execution report the missing
   dependency rather than succeeding through private substitute behavior;
5. the terminal frontend visibly exposes the KY11-D panel state;
6. structured observation demonstrates KY11-D and KD11-A transitions while a
   silent observer produces identical semantic results;
7. focused formatter, linter, and Aiki tests pass with the current Aiki binary;
8. `make validate` passes before the parent Cut 1 is marked GATED.

No Cut 2 implementation begins as part of this proposal.

## Repository integration correction

The first delivered console overlay was incomplete as an Aiki experiment tree:
`experiment/PROCEDURE.md` and executable `experiment/run.sh` were missing. A
user-run `make validate` therefore correctly failed at `aiki treecheck`. Those
required experiment artifacts are now part of Cut 1. Repository format/lint
checks are performed through `make fmt` and `make lint`; ad-hoc directory
invocations are not the recorded gate.

## Current evidence

Focused Cut 1 evidence with the supplied current `v0.4.0-alpha-39` Aiki binary:

```text
KY11-D tests            16/16 PASS
KD11-A control tests    26/26 PASS
integration tests       18/18 PASS
formatter/linter              PASS
scripted terminal acceptance  PASS
git diff --check              PASS
make historycheck             PASS
```

The refined tests exercise the pure transition relations directly: input state is
unchanged, returned next state carries the new architectural state, state-preserving
no-effect and momentary-control transitions are explicit, manual-address progression
is visible in KD11-A state, and wired observation exposes both `ky11d.transition`
and `kd11a.transition`.

The real `aiki test` runner produced these counts. A scripted terminal session
confirmed visible octal ADDRESS/DATA/Switch state, honest UNIBUS dependency on
EXAM, honest KD11-A execution dependency on HALT+CONT, and structured trace
events from both devices.

The full repository gate is not complete in the analysis environment.
`GOTOOLCHAIN=local make validate` stops at the build step because `go.mod`
requires Go 1.24 and the available host toolchain is Go 1.23.2. This proposal and
parent Cut 1 therefore remain ACTIVE rather than GATED.

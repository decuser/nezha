# Nezha — Evaluation Architecture Proposal

## Premise

Nezha is a deliberately contained descendant of Aiki.

Aiki is the fuller realization: a working language with modules, concurrency, runtime services, profiling, optimized representations, FFI, and other machinery.

Nezha begins by removing that breadth.

Its purpose is to test a narrower hypothesis:

> The evaluation mechanics of a small, well-structured language can be fully expressed using a finite and small set of declarative evaluation characters.

The hypothesis is local first. It need only hold for Nezha.

If successful, other well-conceptualized languages may exhibit recognizable variations of the same evaluation morphology without sharing either Nezha's grammar or its exact character set.

## Architectural Principle

Nezha separates three things sharply:

```text
syntax
evaluation
realization
```

The intended architecture is:

```text
syntax.ebnfx
     +
 eval.spec
     |
     v
eval-spec engine
     |
     v
binary evaluation ABI
   /       |       \
  /        |        \
 v         v         v
interp     VM      native
```

### EBNFX

EBNFX is authoritative for:

- lexical structure;
- grammar;
- structural relationships;
- syntax diagnostics;
- templates;
- help;
- documentation.

There must be no structural-language knowledge duplicated elsewhere. If structural knowledge bleeds into another component, it should be moved into or derived from EBNFX.

### Eval-Spec

Eval-spec is authoritative for evaluation mechanics.

It describes what happens when valid language structures are evaluated, using a deliberately small set of semantic characters and composition rules.

No realization may contain private Nezha construct semantics.

### Realization

Realization is deliberately open.

An interpreter, VM, or native compiler may realize the semantic contract differently, but may not alter it.

Representation, allocation, environment layout, numeric strategy, frame reuse, instruction selection, optimization, and similar decisions belong here.

## Evaluation Character Hypothesis

The character set is not designed in advance.

It is discovered by decomposing the actual evaluator of contained Aiki.

Candidate acts may eventually resemble:

```text
produce
resolve
bind
update
test
choose
construct
operate
invoke
return
discriminate
```

but these names are provisional.

A character is accepted only if:

1. it is necessary to express Nezha evaluation;
2. it recurs compositionally;
3. it describes language behavior rather than one implementation;
4. both an interpreter and a compiler can realize it naturally.

The objective is not a universal semantics framework.

The objective is a small closed evaluation alphabet for Nezha.

If the alphabet expands into an open-ended semantic encyclopedia, the experiment has gone in the wrong direction.

## Initial Nezha Language

Nezha begins as a contained Aiki.

Likely retained core:

```text
values
    number
    boolean
    rune
    string
    symbol
    list
    function

binding
    let
    assignment
    lexical lookup

control
    if / else
    while
    return

expressions
    literals
    names
    lists
    unary operations
    binary operations
    grouping
    calls
    indexing

functions
    lexical functions
    parameters
    invocation

patterns
    match
```

Likely removed initially:

```text
package
import
use
export
load

spawn
channel
send
recv
select

HAL
FFI
host resources

shapes / access
pipe

libraries / prelude where unnecessary

profiling / observation

tail-call optimization
environment reuse
argument-store reuse
other realization optimizations
```

These boundaries are provisional until tested against the extracted implementation.

## Binary Evaluation ABI

Eval-spec is compiled.

It is not intended to remain as a generic semantic interpreter in the runtime path.

The eval-spec engine produces a real binary interface representing normalized evaluation.

The ABI should contain enough information for a realization to proceed without knowing Nezha constructs.

A realization may know:

```text
binding
control edges
value dependencies
operations
invocation
return
construction
etc.
```

It must not need to know:

```text
while_stmt
let_stmt
match_arm
function_literal
```

If a realization requires construct-specific knowledge, either:

- the ABI is incomplete; or
- semantics have leaked below the boundary.

The ABI should avoid prematurely encoding a stack machine, register machine, or AST evaluator. A graph-like semantic representation may prove more suitable.

## No Runtime Abstraction Tax

Declarative evaluation must not imply permanent runtime indirection.

The eval-spec engine should specialize or generate realization machinery ahead of execution.

For an interpreter, the target is performance comparable to an equivalent simple handwritten Aiki evaluator with Aiki's optimizations removed.

The intended path is:

```text
eval-spec
   ↓
generation / specialization
   ↓
ordinary evaluator code
```

not:

```text
program
   ↓
generic eval-spec executor
   ↓
character dispatcher
   ↓
runtime
```

A slow direct eval-spec executor may be useful for testing, but it is not the production design.

## Development Cuts

### Cut 1 — Nezha Exists

Status: done.

Goal: establish Nezha as a separate working descendant of the Aiki baseline without changing the retained core semantics.

Done:

- cloned the Aiki Alpha 3 baseline into a new Nezha working repository;
- removed the broad Aiki library/module/host-facing surface to establish a contained descendant;
- established dedicated Nezha identity (`cmd/nezha`, product renaming, `.nezhaignore`, binary names);
- moved `cmd/aikiapp` to `cmd/cli`;
- systematically updated source, tests, gold files, and structure artifacts (`.ai` → `.nez`);
- removed Aiki-specific distribution breadth, visual/canvas surfaces, module-loading surfaces, and REPL UX;
- produced baseline tarball `nezha-baseline-v0.1.0-alpha-1.tar.gz`;
- build and inherited retained behavior preserved.

Gate satisfied:

- Nezha builds;
- Nezha runs retained inherited programs as expected;
- lexer, parser, formatter, and evaluator paths work over `.nez` source;
- baseline tests remain usable;
- Aiki source baseline remains available for comparison.

### Cut 2 — Contain the Language

Goal: reduce the inherited engine to the Nezha core while preserving the existing handwritten evaluator architecture.

Remove feature families incrementally.

Likely order:

```text
host / HAL / FFI facilities
modules and package facilities
concurrency
shapes / access / pipe
profiling and observation
Aiki-specific optimization machinery
unneeded libraries / prelude surface
```

After each subtraction:

```text
build
tests
small semantic smoke corpus
```

Do not combine unrelated removals merely to finish the cut faster.

Gate: every surviving path under `engine/` can be justified as necessary for:

```text
syntax
core evaluation
values
minimal realization
```

At this point Nezha is the specimen to be dissected.

### Cut 3 — Break Down the Characters

Goal: express the complete surviving handwritten evaluator in terms of a small evaluation alphabet.

For every retained grammar construct:

1. write down exactly what its current evaluator does;
2. separate semantic requirement from realization detail;
3. factor repeated semantic acts;
4. propose characters;
5. test each character against both interpreter and compiler realizations;
6. continue until every evaluation path is accounted for.

No arbitrary escape into Go is permitted in the eventual model.

This cut may initially remain descriptive. The important artifact is the candidate evaluation algebra, not tooling.

Gate: every surviving Nezha construct has a complete evaluation description using the candidate character set. No evaluator behavior is unexplained.

### Cut 4 — Eval-Spec and ABI

Goal: make the evaluation description executable without making it runtime machinery.

Build:

```text
eval.spec
eval-spec engine
binary evaluation ABI
```

Then construct the first realization from it: a simple interpreter.

Compare behavior against the contained handwritten evaluator.

Compare performance against the equivalent unoptimized handwritten evaluator.

Gate:

- eval-spec completely describes Nezha evaluation;
- ABI contains everything required by the interpreter;
- handwritten realization code contains no private Nezha construct semantics;
- generated/specialized interpreter behavior matches the reference;
- no substantial inherent runtime abstraction penalty is introduced.

## Later Work

Only after those gates:

```text
VM realization
native realization
optimization
Aiki-level representation strategies
language extensions
cross-language experiments
```

A second language should eventually be tested against the model.

The interesting question would then be whether it requires:

```text
same alphabet, different composition
```

or:

```text
recognizable alphabet + a small number of new characters
```

rather than whether Nezha's exact vocabulary is universal.

## Core Invariants

1. EBNFX owns structure.
2. Eval-spec owns evaluation.
3. Nothing below the evaluation ABI owns Nezha constructs.
4. Realization remains replaceable.
5. Declarative semantics must not require runtime semantic indirection.
6. Smallness is a requirement, not an aesthetic preference.
7. When a proposed language feature badly deforms eval-spec, question the feature before enlarging the semantic machinery.

The experiment succeeds if Nezha can ultimately be described by something close to:

```text
syntax.ebnfx
eval.spec
```

with interpreter, VM, and native execution remaining replaceable realizations beneath a stable binary contract.

# Aiki

Aiki is an experimental programming language designed as a learning field. Its goal is to make language behavior inspectable enough to support learning, debugging, and careful reasoning about programs.

Aiki uses a small syntax, one exact numeric model, left-to-right evaluation, explicit grouping, lazy logical control operators (`and`/`or`), recoverable errors as values, proper tail calls, and an inspectable prelude and library surface.

Development notes and project updates: [https://decuser.github.io](https://decuser.github.io)

## Status

Aiki is currently at its **Alpha 3** milestone (`v0.4.0-alpha-36`). The language is real, usable, and extensively validated, but syntax, libraries, tooling, and interfaces may continue to change as the design is refined. See the [Alpha 3 release notes](docs/releases/alpha-3/release-notes.md) for the milestone changes since Alpha 2.

Alpha 3 was validated on Linux, macOS, and Windows. Prebuilt Intel/amd64 release archives are provided for all three platforms.

## Alpha Expectations

Aiki is usable, but it is still alpha software. Syntax, libraries, tooling, and interfaces may change as the language is refined. Alpha 3 includes substantial runtime and allocation optimization, but performance remains an implementation concern rather than a language compatibility promise. Some areas are intentionally incomplete rather than treated as stable commitments.

## Getting Started

Aiki's normal user installation is a release archive. Unpack it in a stable
location and add that directory to `PATH`:

```bash
mkdir -p ~/opt
tar xzf aiki-<version>-linux-amd64.tar.gz -C ~/opt
export PATH="$HOME/opt/aiki-<version>-linux-amd64:$PATH"
```

Put the `PATH` line in your shell startup file if you want it to persist. Then
Aiki is available like any other command:

```bash
aiki                 # start the REPL
aiki program.ai      # run a program
```

The release directory is relocatable: the `aiki` executable finds its shipped
`lib/` and `vendor/` modules relative to itself rather than relative to the
directory from which you run it. Named packages are resolved only from those
explicit distribution roots and `~/.aiki/lib`; Aiki does not recursively scan
the current working directory for packages. Local source files are imported
explicitly by relative path, for example `import("./helper", ...)`.

On Linux, Aiki uses Ebitengine for graphics and therefore needs its native system
libraries even when using a prebuilt binary. On Debian or Ubuntu:

```bash
sudo apt install gcc libc6-dev libgl1-mesa-dev \
    libxcursor-dev libxi-dev libxinerama-dev \
    libxrandr-dev libxxf86vm-dev libasound2-dev \
    pkg-config
```

### Building Aiki from source

Development additionally requires Go 1.24 or later and `make`. From the Aiki
repository:

```bash
make validate-lang
```

`make validate-lang` is the routine language/runtime gate and honors normal CLI
target policy. Use `make validate-all` for exhaustive applicable repository
validation, including targets normally omitted by `.aikiignore`; use
`make rigorous` for the exhaustive gate plus the expensive self-host and fuzz
checks.

Use `./aiki` when you specifically want the development build in the source
tree. To generate the user distribution and prove that it runs independently of
the source tree:

```bash
make distcheck
```

`make dist` writes both the unpacked user distribution and its `.tar.gz`
archive beside the source tree. For example, from `~/forge/dev/aiki` it produces
sibling paths named `aiki-<version>-<os>-<arch>/` and
`aiki-<version>-<os>-<arch>.tar.gz`. The user distribution contains the built
`aiki` and `aiki-canvas` executables, its shipped library, current and historical
documentation, and sample programs; it is separate from the development source
tree and does not require Go.

For a portable development/restart snapshot, use:

```bash
make baseline
```

`make baseline` writes `aiki-baseline-<version>.tar.gz` beside the source tree.
It captures the working repository including `.git` so branch, history, refs,
and engineering work state are preserved, while omitting the built top-level
`aiki` and `aiki-canvas` executables. This is a development baseline, not the user
distribution.

## Try Aiki

Sample programs are in `samples/`. Useful starting points include:

```text
samples/pipeline.ai
samples/newton-rec.ai
samples/newton-imp.ai
samples/face.ai
samples/turtle-star.ai
```

Samples are examples rather than a batch validation suite: some are interactive
and some open a canvas. Automated behavioral and graphical guarantees live under
`test/behavior/` and `test/visual/`.

Portable library semantics default to their Aiki-native implementation when a
native module exists. Programs may opt into provider-backed `/ffi` modules
explicitly and may freely mix native and FFI use. To inspect that dependency
without executing a program:

```sh
aiki check --ffi-use program.ai
```

The diagnostic follows literal direct and transitive imports and reports the FFI
modules reached, or `FFI imports: none`.

## Documentation

Milestone documents are frozen under `docs/releases/`:

```text
docs/releases/alpha-1/language-report.odt
docs/releases/alpha-1/this-is-aiki.odt
docs/releases/alpha-1/release-notes.md
docs/releases/alpha-2/this-is-aiki.odt
docs/releases/alpha-2/release-notes.md
docs/releases/alpha-3/language-report.odt
docs/releases/alpha-3/this-is-aiki.odt
docs/releases/alpha-3/release-notes.md
```

Current implementation and contributor documentation is indexed by
`docs/README.md`, including:

```text
docs/adding-to-aiki.md
docs/debug.md
docs/decisions.md
docs/testing.md
docs/profiling.md
docs/hal.md
docs/style.md
```

Engineering history and process live under `engineering/`; they are not current
language documentation.

## Forthcoming

A few little books about Aiki are on the way, including material aimed particularly at non-programmers, younger learners, and readers who want to explore programming without first learning a large amount of machinery. They will complement the repository documentation and take a slower, more playful route into the language.

For example, Aiki's `turtle/simple` library keeps the first steps deliberately small:

```aiki
use("turtle/simple")

new(400)
pencolor(:green)
pen_size(3)

forward(50)
right(90)
forward(50)
right(90)
forward(50)
right(90)
forward(50)
```

## Developing Aiki

Run the routine language/runtime validation gate with:

```bash
make validate-lang
```

Use `make validate-all` for exhaustive applicable repository validation and
`make rigorous` for the exhaustive gate plus expensive/deep checks.

Aiki distinguishes checking, blessing, and validation:

```bash
make check          # ordinary correctness checks; never writes golds
make bless          # after a verified intentional change, update blessed golds
make validate-lang  # routine language/runtime gate
make validate-all   # exhaustive applicable repository gate
make rigorous       # exhaustive gate plus expensive/deep checks
```

See `docs/testing.md` before updating gold files and `docs/adding-to-aiki.md` before extending the implementation.


## Experiments

The long-running empirical and machine-reconstruction work now lives in the
separate `aiki-experiments` repository. It intentionally consumes Aiki from
outside this source tree and serves as an independent integration/regression
surface. The Aiki distribution itself contains the language/runtime executables, shipped
libraries, documentation, and samples; it no longer ships experiment collections or
the source repository's validation suite.

## Editor Support

Aiki exposes editor-independent language services through `aiki lsp`. Xed and
VS Code use thin clients over that service; nvi uses classic tags generated by
Aiki. Development installs are available from the source repository (the binary user
distribution does not include the editor-adapter sources or Make targets):

```bash
make install-xed-plugin
make install-vscode-plugin
./aiki tags -o tags path/to/source.ai
```

The editor adapters do not implement Aiki semantics. Diagnostics, definitions,
formatting, completion, and hover come from the same language-service core.
In the source repository, see `editors/xed/README.md` and
`editors/vscode/README.md` for editor-specific setup and live checks.

## Self-Description and Self-Hosting

Aiki includes an independent interpreter written in Aiki under `selfhost/`. It
lexes, normalizes, parses, evaluates, and self-host-loads Aiki source modules
without reusing the Go lexer/parser/evaluator implementation. Cross-implementation
invariants compare lexical, syntactic, module, and behavioral results. The final
bootstrap invariant runs the Aiki-written interpreter through itself and requires
a third-level Aiki program to produce the specified result.

The Go implementation remains the production runtime and bootstrap substrate;
self-hosting is a conformance and boundary-sufficiency proof, not a replacement
for the Go runtime.

## Repository Layout

```text
cmd/                     command entry points
engine/syntax/           lexer, parser, grammar, grammar help
engine/semantics/        evaluator and runtime values
engine/runtime/          HAL, prelude, runtime support
lib/                     Aiki library packages
samples/                 example programs
editors/                 editor support
test/                    tests and expected outputs
docs/                    current language, tool, and contributor documentation
engineering/             engineering method, proposals, profiling, and durable records
```

## Roadmap

Aiki will continue to evolve toward a smaller, clearer, more inspectable language and implementation. Work will focus broadly on hardening the language, improving its tools and documentation, extending its usefulness as a systems and teaching environment, and continuing to test whether its design commitments hold under broader use.

The project does not currently promise API, library, or syntax stability. Changes will be driven by clarity, coherence, and observable behavior rather than compatibility for its own sake.

## Feedback

Aiki is in alpha. Bug reports, reproducible failures, documentation corrections, and discussion of language behavior are welcome.

## AI and Authorship

Aiki was conceived, architected, designed, specified, and directed by Will Senn (decuser). Generative AI was used extensively as an implementation instrument under continuous human direction, supervision, and control. It produced source code, tests, tooling, documentation support, and related engineering material. It did not possess design authority.

The principal architectural commitments of Aiki—exact rational arithmetic, isolated spawn concurrency, the HAL boundary, left-to-right evaluation, shaped data, the scope-gated registry, the value model, the module system, and the executable-documentation stance—were established by Will Senn and imposed on the implementation. Generative AI was sometimes asked to suggest alternatives, formulations, tests, or approaches. Those suggestions had no standing by virtue of being generated. They became relevant only when examined, accepted, modified, or rejected by the designer.

Language semantics, architecture, constraints, requirements, and acceptance criteria remained under the designer's authority and were not derived from generated implementation. Output was evaluated solely against observable behavior, specifications, executable documentation, tests, gold files, structural invariants, and other externally imposed validation gates. Generated material was accepted, revised, rejected, or replaced according to those criteria. When it was correct, it was correct because it satisfied requirements established by the designer, not because the generated artifact was treated as authoritative.

The designer worked from the language's observable surface, specifications, and documented semantics rather than from the implementation source. This separation was intentional: the implementation was required to conform to the language, not the language to the implementation. Design authority was exercised through specifications, acceptance criteria, executable documentation, behavioral tests, gold files, structural invariants, and a gated engineering record.

Generative AI operated under a comparable separation: receive an explicit task, produce candidate material, return it for evaluation. Context could accumulate; authority did not. Continuity of conversation never became continuity of design ownership. The methods evolved as the project matured, but the authority boundary did not: implementation assistance, criticism, and suggestion never became independent design authority.

Generative AI therefore participated substantially in the engineering of Aiki. Nothing entered the language merely because an AI suggested or generated it. Adoption remained an act of design judgment. The AI produced code under Will Senn's direction, within an architecture he determined, against behaviors, constraints, and validation standards he controlled. The generated implementation realizes a design whose authority lies outside the generated code. The design is his.

## Invariant Framework

Aiki's implementation is governed by an invariant framework that keeps the executable system, the documented language, and the distributed source tree in agreement. The governing principle is simple: implementation code is replaceable; observable language behavior and declared structure are authoritative.

The framework operates at multiple levels. Behavioral tests establish expected language semantics. Gold files preserve exact observable results and detect unintended drift. Executable documentation runs documented examples against the language itself so that examples cannot silently diverge from implementation. Grammar coverage requires that declared syntax have corresponding implementation behavior. Structural checks enforce repository and architectural boundaries. Library documentation and help are checked against exported interfaces. Engine smoke tests exercise the language pipeline independently of ordinary unit tests.

These are reinforced by explicit executable couplings between parts of the system that could otherwise drift independently, including:

- grammar-to-handler coverage
- prelude-to-help consistency
- formatter-to-AST coverage
- modules-to-exports consistency
- behavior-to-gold agreement
- library help and documentation-to-export agreement
- graphics-boundary confinement
- canvas transcript golds
- documentation examples-to-stated-values agreement
- documentation-entry disposition
- module-documentation presence

Source-tree structure is further constrained by `treecheck`: every repository file must have a recognized structural relationship or an explicit standalone disposition. The binary user distribution is checked separately by `distcheck`.

The purpose is not merely to accumulate tests. Each invariant expresses a relationship the project intends to remain true. A change is acceptable only when the affected relationships continue to hold or are deliberately revised together. This makes drift visible: syntax cannot quietly outrun the evaluator, documentation cannot quietly cease to describe executable behavior, exports cannot silently escape their help surface, and implementation reorganization cannot arbitrarily alter the distribution.

The framework is reinforced by a gated working method. Changes are made in small serial cuts and validated at the appropriate level. Live rationale, limitations, and restart state belong to the active proposal or transient handoff; completed work leaves only durable outcomes and independently useful evidence. Claims of completion are therefore tied to evidence rather than to the production of code.

This framework is especially important given the project's extensive use of generative AI. Generated implementation is not trusted because it is plausible, idiomatic, or internally consistent. It is constrained from outside itself. The tests, gold files, executable documents, structural checks, and other invariants form the boundary between generated implementation and language authority. They are the means by which an implementation produced with substantial AI assistance remains subordinate to a human-defined language rather than gradually redefining that language through implementation accident.

## License

Aiki is licensed under the BSD 3-Clause License.

Copyright (c) 2026, William D. Senn. See `LICENSE`.

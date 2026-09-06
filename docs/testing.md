# Testing, Blessing, and Validation

Aiki separates three different acts: checking correctness, blessing a known-good reference state, and validating later work against that state.

## `make check`

`make check` runs correctness checks that do not depend on existing behavior or engine gold snapshots. It builds and formats the tree, runs the linter, verifies the distribution tree with `aiki treecheck`, runs Go tests and Aiki tests, and verifies that the structural engine specimens exercise every production declared by `grammar.ebnfx`.

`make check` never writes gold files.

Use it while developing an intentional behavior or structural change, before updating reference snapshots.

## `make bless`

`make bless` first runs `make check`. Only if those independent checks succeed does it replace the blessed reference snapshots:

```text
behavior smoke transcripts
lexer structural golds
parser structural golds
evaluator structural golds
EBNF grammar gold
```

Blessing is not validation. It records the current implementation as the new reference state after the intended behavior has been established independently.

The underlying commands are:

```bash
./aiki smoke --gold test/behavior/
./aiki enginesmoke --stage all --gold test/structure/engine
```

`smoke --gold` preserves authored `IN:` and `DISPLAY:` directives from an existing transcript, then regenerates observed `OUT:`, `ERR:`, `EXIT:`, and `CANVAS:` records using the smoke framework's canonical transcript encoding. A new no-input smoke can be blessed without an existing gold; input-bearing smokes should establish their `IN:` directives before blessing.

### Negative behavior fixtures

A smoke specimen whose intended behavior is failure declares that intent in a
leading source comment. The declaration is textual so it remains readable even
when the specimen intentionally does not parse:

```text
# @negative parse
```

The marker is a general negative-fixture declaration; `parse` is currently the
only supported kind. Unknown kinds are errors. Negative declarations are valid
only on `*_smoke.ai` specimens; placing one in ordinary `.ai` source is an
error, so application or library source cannot exempt itself from formatter or
linter coverage. A parse-negative fixture is excluded from formatter and linter
source traversal because successful parsing is not part of that specimen's
contract.

Intent and evidence are checked separately. `# @negative parse` is the intent;
the smoke transcript is the observation. Smoke validation requires both
directions: every declared parse-negative must have a nonzero parser-failure
transcript, and every parser-failure transcript must belong to a specimen that
declares `# @negative parse`. Blessing applies the same check to the observed
run before writing a gold, so a newly broken ordinary specimen cannot authorize
itself by being reblessed.

`enginesmoke --gold` refuses to bless an incomplete structural suite: every EBNF production must first be exercised by at least one `*_engine.ai` specimen.

## Validation levels

Repository selection policy belongs to the Aiki CLI resolver, not to Makefile
path lists. The top-level validation targets select a level; they do not know
which repository trees are normally ignored.

`make validate-lang` is the routine language/runtime gate. It uses ordinary CLI
target policy, so commands such as `aiki test ./...` honor `.aikiignore`. It runs
the independent correctness checks and compares current behavior and engine
structure with the already-blessed golds. Select the required validation level explicitly; there is no generic validation alias.

`make validate-all` is the exhaustive applicable repository gate. It runs the
ordinary Go package suite with `go test ./...` and invokes recursive Aiki
validation surfaces with `--all`, which bypasses `.aikiignore` while preserving
each command's own source filtering and directory semantics. This includes
exhaustive Aiki test, lint, fmt, and distfmt selection.

`make rigorous` starts from `validate-all` and adds the expensive/deep checks:
tagged self-host validation and lexer/parser fuzzing.

All three validation levels never bless gold files. Formatting commands retain
their normal formatting behavior.

A successful `validate-lang` means the normal language/runtime population and
blessed behavior/engine structure are sound. A successful `validate-all` extends
that evidence to normally ignored applicable repository targets. A successful
`rigorous` additionally exercises the expensive/deep surfaces.

## Normal workflow

For a change that should not alter blessed behavior or structure:

```bash
make validate-lang
```

Use the exhaustive gate when the change affects repository-wide discovery or
when full-tree evidence is wanted:

```bash
make validate-all
```

For an intentional behavior or structural change:

```bash
make check
# inspect and establish that the new behavior/structure is correct
make bless
make validate-lang
```

Then commit the implementation and updated golds together.

## Why golds come after checks

A gold is an oracle for future regression detection; generating one does not prove that the captured behavior is correct. Unit, semantic, invariant, and human review establish correctness first. Blessing then freezes that known-good state so later drift is visible.

## Distribution tree invariant

`aiki treecheck` checks that every file in the current source tree has a recognized distribution relationship or an explicit standalone disposition. It is part of `make check` and therefore `make validate-lang` (and the exhaustive validation levels built above it).

The checker infers ordinary relationships such as standard-library `.ai`/`.help`/`.doc` sets, Aiki-native tests, smoke specimens and golds, engine specimens and stage golds, grammar/prelude artifacts, samples, profiling drivers, native/FFI conformance specimens, Go implementation files, and direct references from already-justified text files. It also detects structural contradictions such as a gold without its source specimen or a module companion without an owning `.ai` file.

Intentional standalone artifacts are listed in the small root file `treecheck.allow`. That file is an exception list, not a manifest of the repository. Directory entries end in `/`; other entries use `filepath.Match`-style patterns. Add an exception only when a file is intentionally standalone and no stronger structural relationship describes it.

When Git metadata is available, `treecheck` examines tracked files plus untracked non-ignored files that actually exist in the working tree. This makes overlay additions visible before staging while also allowing a removed tracked path to disappear normally. Without Git metadata, it falls back to walking the source tree.

## Runtime coverage model

`aiki test -cover` uses one dynamic execution observation path: the profiling
network records what actually executed, while `engine/coverage` derives the
static source anatomy that could have executed. Coverage is the comparison of
those two authorities:

```text
static parsed Aiki anatomy     what could be covered
profiling network              what actually executed
coverage analysis              comparison of the two
```

A static coverage site is a source-backed non-`TERMINAL` AST node. Line
coverage rolls those sites up to the distinct source lines that contain them,
so blank lines, comments, and punctuation-only lines are not part of the
denominator. Static sites carry file, lexical-function, module/library, and
semantic-layer coordinates without creating runtime profiling nodes for code
that never executes.

Dynamic coverage observes entry to `Evaluator.Eval`. The evaluator exposes an
optional `EvalProbe` nerve, and `engine/observe` provides a network-backed
`EvalCapture`. Each AST object caches only its compact `(registry token, NodeID)`
binding after first encounter; later Eval entries record dense network-owned
node hits. `aiki test -cover` consumes those network observations directly; it
does not maintain a second runtime coverage counter.

Coverage reporting keeps portable coordinates separate from repository policy.
`engine/coverage` knows files, lexical functions, modules/libraries, layers, and
static sites. The `aiki test` command classifies this repository's paths into
report populations such as supported libraries, examples, and auxiliary
material. Those population labels are not profiling-network identity and are
not assumed to exist in other codebases. External consumers, including the
separate Aiki experiments repository, may apply their own coverage population
policy. Examples in this repository remain reportable evidence without defining
the supported-library headline.

The supported coverage views are `summary`, `file`, `function`, `module`,
`layer`, and `scope`. `-coverprofile` writes the observed line-hit profile and
implies `-cover`.

`aiki enginesmoke --coverage` asks a different structural question: whether the
engine specimens exercise the grammar productions. It is not runtime Aiki
source coverage and is not folded into the profiling network.

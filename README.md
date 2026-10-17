# Nezha

**Nezha is Aiki’s smaller, tougher sibling.**

The name recalls Nezha (哪吒), the youthful warrior of Chinese mythology: small in stature, formidable in combat, and unwilling to accept inherited constraints. Nezha the language takes a similar approach to its Aiki ancestry—not by growing larger, but by stripping away what it does not need and confronting the harder question of what makes evaluation work.

Nezha is an experimental programming language descended from Aiki, intended to test whether language evaluation can be described declaratively.

Its central hypothesis is that the evaluation mechanics of a small, well-structured language can be expressed using a **finite, small set of composable evaluation characters**, separately from syntax and implementation strategy.

The intended separation is:

```text
syntax
 evaluation
 realization
```

The proposed long-term architecture is:

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
  v        v        v
interp     VM      native
```

**This is a design target, not the current implementation.**

## Current development status

The repository contains a working `.nez` language implementation and a handwritten evaluator. The first development cut separated and renamed the Aiki descendant; the project builds and exercises lexer, parser, formatter, and evaluation paths.

The ongoing work is to reduce inherited Aiki machinery and inventory the evaluation acts needed by the surviving language. The evaluation-character inventory remains provisional. The declarative `eval.spec` engine, binary evaluation ABI, and independent interpreter/VM/native realizations proposed above have **not** been implemented in this baseline.

This is an early development baseline (`dev-v0.0.1`), not a claim of a completed semantic architecture.

## Build and run

Build the `nezha` command:

```sh
make build
```

Run a source file:

```sh
./nezha path/to/file.nez
```

Evaluate an expression:

```sh
./nezha -e '1 + 2 * 3'
```

Inspect language processing stages:

```sh
./nezha debug -stage lex path/to/file.nez
./nezha debug -stage parse path/to/file.nez
./nezha debug -stage fmt path/to/file.nez
./nezha debug -stage all path/to/file.nez
```

Run reference specimens and validation:

```sh
make reference
make validate-lang
make cleanroom
```

The commented `.nez` specimens under `reference/` document executable examples of the current language.

## An experimental block-based interface

Oh, and there is a block-based user interface thrown in as an experiment. The `blockly/` directory contains a Blockly editor wrapped in a Wails desktop application, allowing programs to be assembled visually and translated into Nezha source. It is an exploratory alternative to writing `.nez` files, not a second language or a central part of the evaluation architecture.

With Wails and the required desktop dependencies installed, try:

```sh
make blockly
```

## Design and contributor guidance

- [Evaluation architecture proposal](engineering/proposals/active/nezha-evaluation-architecture-proposal.md): hypothesis, semantic boundaries, and development cuts.
- [Evaluation-character inventory](engineering/proposals/active/evaluation-character-inventory.md): active analysis, not a finalized character set.
- [AI contribution instructions](engineering/ai/README.md): project rules for bounded, reviewed and validated AI-assisted changes.

## License

BSD 3-Clause. See [LICENSE](LICENSE).

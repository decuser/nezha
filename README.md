# Nezha

Nezha is a deliberately contained descendant of Aiki.

The current purpose of the repository is to test whether the evaluation mechanics of a small, well-structured language can be expressed using a finite and small set of declarative evaluation characters.

Nezha separates:

```text
syntax
evaluation
realization
```

The intended long-term shape is:

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

## Current State

Nezha has been split from the Aiki Alpha 3 baseline and renamed as its own working repository.

The source extension is `.nez`.

The command binary is `nezha`.

The first milestone established that the renamed tree builds and that lexer, parser, formatter, and evaluator paths work over `.nez` source.

## Primary Design Notes

See:

- `engineering/proposals/active/nezha-evaluation-architecture-proposal.md`
- `engineering/ai/README.md`

The proposal describes the evaluation architecture and cut sequence.

The AI contract describes how AI-generated changes should be bounded, reviewed, and gated.

## Basic Commands

Build:

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

Inspect lexer/parser/formatter/evaluator stages:

```sh
./nezha debug -stage lex path/to/file.nez
./nezha debug -stage parse path/to/file.nez
./nezha debug -stage fmt path/to/file.nez
./nezha debug -stage all path/to/file.nez
```

Validate:

```sh
make validate-lang
make validate-all
make rigorous
```

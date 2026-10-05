# Nezha Blockly

This is a deliberately small local Blockly front end for Nezha.

Blockly emits ordinary Nezha source. A minimal Wails host binds that source
straight to the existing Nezha runner, with runtime-owned input and output
endpoints. There is no HTTP service, no remote runtime dependency, and no
Node/npm frontend build. Blockly itself remains vendored browser JavaScript.

## One-time Blockly vendor step

From the Nezha repository root:

    ./blockly/vendor-blockly.sh

This downloads the pinned official Blockly 13.3.0 release, verifies its SHA-256,
and copies only the browser files needed by this prototype into
`blockly/web/vendor/blockly/`.

## Run

The Wails host needs the normal platform WebKitGTK development package on Linux.
From the Nezha repository root:

    make blockly

On Debian/Devuan systems that provide WebKitGTK 4.1, the make target passes the
Wails `webkit2_41` build tag.

## Editor views

The application has three views:

- **Editor** — Blockly workspace, Input, generated Nezha source, and Output.
- **Samples** — bundled example programs. Opening one loads it into the ordinary
  editor with the full toolbox.
- **Help** — currently a small placeholder; it is intended to project Nezha's
  own help/reference metadata rather than grow a separate documentation system.

The default working program is the Input / Output sample. It reads `Max` from the
Input pane and prints `hello, Max`.

## Files

The native File menu provides New, Open, Save, Save As, Export Nezha Source, and
Quit. Save writes a versioned `.nezha-blocks` JSON document containing Blockly's
serialized workspace and the Input buffer. Export writes ordinary `.nezha`
source.

The document envelope is intentionally small:

    {
      "format": "nezha-blockly",
      "version": 1,
      "input": "...",
      "workspace": { ... }
    }

## Current language boundary

The Blockly side currently exposes Nezha literals, `read`, binding/reference,
assignment, `print`/`println`, and binary expressions. `let` introduces a name;
`assign` and `name` choose only bindings introduced by an earlier `let` in the
same statement chain. Moving a use above its binding leaves it visibly unbound
rather than generating an invalid Nezha assignment/reference.

The input pane supplies the text consumed by `read()`. It is a preloaded input
buffer, not a live terminal; EOF follows the last character. The source pane
shows the generated Nezha program before execution. The Nezha parser/evaluator
remains authoritative.

Binary blocks emit explicit parentheses so Blockly nesting preserves grouping
despite Nezha's flat left-to-right infix grammar.

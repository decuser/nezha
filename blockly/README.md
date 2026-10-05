# Nezha Blockly — first cut

This is a deliberately small local Blockly front end for Nezha.

It currently has one statement block, `println`, and Nezha's literal value
forms: number, string, rune, symbol, and boolean. The workspace emits ordinary
Nezha source and POSTs that source to a tiny Go host which executes it through
the existing Nezha runner.

There is no Node/npm dependency and no remote runtime dependency.

## One-time Blockly vendor step

From the Nezha repository root:

    ./blockly/vendor-blockly.sh

This downloads the pinned official Blockly 13.3.0 release, verifies its SHA-256,
and copies only the browser files needed by this prototype into
`blockly/web/vendor/blockly/`.

## Run

From the Nezha repository root:

    go run ./blockly

Then open:

    http://localhost:8080

Set `NEZHA_BLOCKLY_ADDR` to change the listen address, for example:

    NEZHA_BLOCKLY_ADDR=:8090 go run ./blockly

## First-cut boundary

The Blockly side knows only how to generate:

    println(<literal>)

The Nezha parser/evaluator remains authoritative. The source pane deliberately
shows the generated Nezha program before it is executed.

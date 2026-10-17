.PHONY: build clean baseline run blockly reference test fmt fmt-all lint lint-all treecheck cleanroom check check-all bless validate-lang validate-all smoke smoke-all smokegold enginesmoke enginesmoke-all enginesmokegold enginecoverage enginecoverage-all rigorous fuzz

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X nezha/cmd/cli.Version=$(VERSION)"

DIST_PARENT ?= $(abspath ..)


SOURCE_DIR_NAME := $(notdir $(CURDIR))
BASELINE_NAME := nezha-baseline-$(VERSION)
BASELINE_ARCHIVE := $(DIST_PARENT)/$(BASELINE_NAME).tar.gz

build:
	go build $(LDFLAGS) -o nezha ./cmd/nezha

clean:
	rm -f nezha
	rm -f "$(BASELINE_ARCHIVE)"


# Capture a portable development baseline beside the source tree. Unlike dist,
# this is a repository snapshot: it intentionally includes .git so branch,
# history, refs, and engineering work state travel with the source. Only the
# built top-level release executable is omitted.
baseline:
	@set -eu; \
	rm -f "$(BASELINE_ARCHIVE)"; \
	tar -C "$(DIST_PARENT)" \
		--exclude="$(SOURCE_DIR_NAME)/nezha" \
		--exclude="$(SOURCE_DIR_NAME)/nezha.exe" \
		-czf "$(BASELINE_ARCHIVE)" "$(SOURCE_DIR_NAME)"; \
	tar -tzf "$(BASELINE_ARCHIVE)" | grep -q "^$(SOURCE_DIR_NAME)/\.git/" || { \
		echo "baseline: archive is missing .git" >&2; exit 1; \
	}; \
	if tar -tzf "$(BASELINE_ARCHIVE)" | grep -Eq "^$(SOURCE_DIR_NAME)/nezha(\.exe)?$$"; then \
		echo "baseline: archive unexpectedly contains built executable" >&2; exit 1; \
	fi; \
	echo "$(BASELINE_ARCHIVE)"

run: build
	./nezha

# Launch the Blockly/Wails structural editor. The Wails app is its own Go module
# so desktop UI dependencies do not enter the Nezha language/runtime module.
blockly:
	cd blockly && WEBKIT_DISABLE_COMPOSITING_MODE=1 wails dev -tags webkit2_41

# Execute the canonical, commented language specimens through the ordinary CLI.
# The corpus is intentionally human-readable reference material, not smoke gold.
reference: build
	@set -eu; \
	for file in reference/*.nez; do \
		echo "=== $$file ==="; \
		./nezha "$$file"; \
	done

# Behavioral and implementation tests. Architectural invariants are run
# separately by `make invariant` and composed back into `make check`.
test:
	@set -e; \
	pkgs="$$(go list ./... | grep -Ev '/test/(invariant|boundary|conformance)$$')"; \
	go test $$pkgs

# Fast architectural contract checks. Keep this target free of fuzzing, gold
# blessing, and other long-running behavioral validation.
invariant:
	go test ./test/invariant ./test/boundary

fmt:
	go fmt ./...
	./nezha fmt ./...
	./nezha gate distfmt ./...

fmt-all:
	go fmt ./...
	./nezha fmt --all ./...
	./nezha gate distfmt --all ./...

lint:
	./nezha lint ./...

lint-all:
	./nezha lint --all ./...

treecheck: build
	./nezha gate treecheck


smoke:
	./nezha gate smoke test/behavior/

smoke-all:
	./nezha gate smoke --all test/behavior/

smokegold: build
	./nezha gate smoke --gold test/behavior/



enginecoverage: build fmt
	./nezha gate enginesmoke --stage all --coverage test/structure/engine

enginecoverage-all: build fmt-all
	./nezha gate enginesmoke --all --stage all --coverage test/structure/engine

enginesmoke: build fmt
	./nezha gate enginesmoke --stage all --check test/structure/engine

enginesmoke-all: build fmt-all
	./nezha gate enginesmoke --all --stage all --check test/structure/engine

enginesmokegold: build fmt
	./nezha gate enginesmoke --stage all --gold test/structure/engine


# Verify that the language core is self-contained. engine/ + runner/ remain the
# execution boundary; reference/ is copied only as the canonical witness corpus.
# The temporary tree tests the core packages and visibly executes every reference
# specimen through a runner-only client. It is left in place for inspection.
CLEANROOM_DIR ?= /tmp/nezha-cleanroom

cleanroom:
	@set -eu; \
	rm -rf "$(CLEANROOM_DIR)"; \
	mkdir -p "$(CLEANROOM_DIR)"; \
	cp -a go.mod go.sum engine runner reference "$(CLEANROOM_DIR)/"; \
	mkdir -p "$(CLEANROOM_DIR)/cmd/cleanrun"; \
	printf '%s\n' \
		'package main' \
		'' \
		'import (' \
		'    "fmt"' \
		'    "os"' \
		'' \
		'    "nezha/runner"' \
		')' \
		'' \
		'func main() {' \
		'    if len(os.Args) != 2 {' \
		'        fmt.Fprintln(os.Stderr, "usage: cleanrun FILE.nez")' \
		'        os.Exit(2)' \
		'    }' \
		'    if err := runner.Run(os.Args[1]); err != nil {' \
		'        fmt.Fprintln(os.Stderr, err)' \
		'        os.Exit(1)' \
		'    }' \
		'}' \
		> "$(CLEANROOM_DIR)/cmd/cleanrun/main.go"; \
	gofmt -w "$(CLEANROOM_DIR)/cmd/cleanrun/main.go"; \
	cd "$(CLEANROOM_DIR)"; \
	echo "=== cleanroom core packages ==="; \
	go test -buildvcs=false ./engine/... ./runner/...; \
	echo "=== cleanroom reference corpus ==="; \
	for file in reference/*.nez; do \
		echo "=== $$file ==="; \
		go run -buildvcs=false ./cmd/cleanrun "$$file"; \
	done; \
	echo "cleanroom ok: $(CLEANROOM_DIR)"

# Snapshot-independent language/runtime correctness checks. This target never
# writes gold files and uses ordinary CLI target policy.
check: build fmt lint treecheck test invariant enginecoverage

# Exhaustive snapshot-independent repository checks. Repository selection remains
# owned by the CLI: --all bypasses .nezhaignore without teaching Make which trees
# are normally excluded. go test ./... supplies the exhaustive ordinary Go suite.
check-all: build fmt-all lint-all treecheck
	go test ./...
	./nezha gate enginesmoke --all --stage all --coverage test/structure/engine

# Bless the current, independently checked behavior and engine structure.
# Blessing records a reference state; it is not validation.
bless: check
	./nezha gate smoke --gold test/behavior/
	./nezha gate enginesmoke --stage all --gold test/structure/engine

# Routine language/runtime repository gate. Ordinary CLI policy applies, so
# repository test exclusions are honored without Make knowing their paths.
validate-lang: check
	./nezha gate smoke test/behavior/
	./nezha gate enginesmoke --stage all --check test/structure/engine

# Exhaustive applicable repository gate. --all changes target-policy selection
# only; each CLI command retains its own source/fixture semantics.
validate-all: check-all
	./nezha gate smoke --all test/behavior/
	./nezha gate enginesmoke --all --stage all --check test/structure/engine


fuzz:
	go test -fuzz=FuzzLexer ./test/fuzz/ -fuzztime=30s
	go test -fuzz=FuzzParser ./test/fuzz/ -fuzztime=30s

rigorous: validate-all fuzz



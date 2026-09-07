.PHONY: build clean baseline run test fmt fmt-all lint lint-all treecheck check check-all bless validate-lang validate-all smoke smoke-all smokegold nezhatest nezhatest-all enginesmoke enginesmoke-all enginesmokegold enginecoverage enginecoverage-all rigorous fuzz intrinsicgate

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
	./nezha distfmt ./...

fmt-all:
	go fmt ./...
	./nezha fmt --all ./...
	./nezha distfmt --all ./...

lint:
	./nezha lint ./...

lint-all:
	./nezha lint --all ./...

treecheck: build
	./nezha treecheck


smoke:
	./nezha smoke test/behavior/

smoke-all:
	./nezha smoke --all test/behavior/

smokegold: build
	./nezha smoke --gold test/behavior/


nezhatest:
	@if find . -name '*_test.ai' -print -quit | grep -q .; then \
		./nezha test ./...; \
	else \
		echo "nezhatest: no *_test.ai files found; skipping"; \
	fi

nezhatest-all:
	@if find . -name '*_test.ai' -print -quit | grep -q .; then \
		./nezha test --all ./...; \
	else \
		echo "nezhatest-all: no *_test.ai files found; skipping"; \
	fi

enginecoverage: build fmt
	./nezha enginesmoke --stage all --coverage test/structure/engine

enginecoverage-all: build fmt-all
	./nezha enginesmoke --all --stage all --coverage test/structure/engine

enginesmoke: build fmt
	./nezha enginesmoke --stage all --check test/structure/engine

enginesmoke-all: build fmt-all
	./nezha enginesmoke --all --stage all --check test/structure/engine

enginesmokegold: build fmt
	./nezha enginesmoke --stage all --gold test/structure/engine

# Snapshot-independent language/runtime correctness checks. This target never
# writes gold files and uses ordinary CLI target policy.
check: build fmt lint treecheck test invariant nezhatest enginecoverage

# Exhaustive snapshot-independent repository checks. Repository selection remains
# owned by the CLI: --all bypasses .nezhaignore without teaching Make which trees
# are normally excluded. go test ./... supplies the exhaustive ordinary Go suite.
check-all: build fmt-all lint-all treecheck
	go test ./...
	$(MAKE) nezhatest-all
	./nezha enginesmoke --all --stage all --coverage test/structure/engine

# Bless the current, independently checked behavior and engine structure.
# Blessing records a reference state; it is not validation.
bless: check
	./nezha smoke --gold test/behavior/
	./nezha enginesmoke --stage all --gold test/structure/engine

# Routine language/runtime repository gate. Ordinary CLI policy applies, so
# repository test exclusions are honored without Make knowing their paths.
validate-lang: check
	./nezha smoke test/behavior/
	./nezha enginesmoke --stage all --check test/structure/engine

# Exhaustive applicable repository gate. --all changes target-policy selection
# only; each CLI command retains its own source/fixture semantics.
validate-all: check-all
	./nezha smoke --all test/behavior/
	./nezha enginesmoke --all --stage all --check test/structure/engine


fuzz:
	go test -fuzz=FuzzLexer ./test/fuzz/ -fuzztime=30s
	go test -fuzz=FuzzParser ./test/fuzz/ -fuzztime=30s

rigorous: validate-all fuzz



# Reproducible semantic/substrate profiling sweep. Not part of validation.
intrinsicgate: build
	./engineering/profiling/state-machine-intrinsic-gate.sh

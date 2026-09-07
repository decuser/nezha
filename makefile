.PHONY: build clean baseline run test fmt fmt-all lint lint-all treecheck check check-all bless validate-lang validate-all smoke smoke-all smokegold aikitest aikitest-all enginesmoke enginesmoke-all enginesmokegold enginecoverage enginecoverage-all rigorous fuzz intrinsicgate

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X aiki/cmd/aikiapp.Version=$(VERSION)"

DIST_PARENT ?= $(abspath ..)

LINUX_DIST_NAME := aiki-$(VERSION)-linux-amd64
LINUX_DIST_DIR := $(DIST_PARENT)/$(LINUX_DIST_NAME)
LINUX_DIST_ARCHIVE := $(DIST_PARENT)/$(LINUX_DIST_NAME).tar.gz

WINDOWS_DIST_NAME := aiki-$(VERSION)-windows-amd64
WINDOWS_DIST_DIR := $(DIST_PARENT)/$(WINDOWS_DIST_NAME)
WINDOWS_DIST_ARCHIVE := $(DIST_PARENT)/$(WINDOWS_DIST_NAME).tar.gz

SOURCE_DIR_NAME := $(notdir $(CURDIR))
BASELINE_NAME := aiki-baseline-$(VERSION)
BASELINE_ARCHIVE := $(DIST_PARENT)/$(BASELINE_NAME).tar.gz

build:
	go build $(LDFLAGS) -o aiki ./cmd/aiki

clean:
	rm -f aiki
	rm -rf "$(LINUX_DIST_DIR)" "$(WINDOWS_DIST_DIR)"
	rm -f "$(LINUX_DIST_ARCHIVE)" "$(WINDOWS_DIST_ARCHIVE)"
	rm -f "$(BASELINE_ARCHIVE)"


# Capture a portable development baseline beside the source tree. Unlike dist,
# this is a repository snapshot: it intentionally includes .git so branch,
# history, refs, and engineering work state travel with the source. Only the
# built top-level release executable is omitted.
baseline:
	@set -eu; \
	rm -f "$(BASELINE_ARCHIVE)"; \
	tar -C "$(DIST_PARENT)" \
		--exclude="$(SOURCE_DIR_NAME)/aiki" \
		--exclude="$(SOURCE_DIR_NAME)/aiki.exe" \
		-czf "$(BASELINE_ARCHIVE)" "$(SOURCE_DIR_NAME)"; \
	tar -tzf "$(BASELINE_ARCHIVE)" | grep -q "^$(SOURCE_DIR_NAME)/\.git/" || { \
		echo "baseline: archive is missing .git" >&2; exit 1; \
	}; \
	if tar -tzf "$(BASELINE_ARCHIVE)" | grep -Eq "^$(SOURCE_DIR_NAME)/aiki(\.exe)?$$"; then \
		echo "baseline: archive unexpectedly contains built executable" >&2; exit 1; \
	fi; \
	echo "$(BASELINE_ARCHIVE)"

run: build
	./aiki

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
	./aiki fmt ./...
	./aiki distfmt ./...

fmt-all:
	go fmt ./...
	./aiki fmt --all ./...
	./aiki distfmt --all ./...

lint:
	./aiki lint ./...

lint-all:
	./aiki lint --all ./...

treecheck: build
	./aiki treecheck


smoke:
	./aiki smoke test/behavior/

smoke-all:
	./aiki smoke --all test/behavior/

smokegold: build
	./aiki smoke --gold test/behavior/


aikitest:
	@if find . -name '*_test.ai' -print -quit | grep -q .; then \
		./aiki test ./...; \
	else \
		echo "aikitest: no *_test.ai files found; skipping"; \
	fi

aikitest-all:
	@if find . -name '*_test.ai' -print -quit | grep -q .; then \
		./aiki test --all ./...; \
	else \
		echo "aikitest-all: no *_test.ai files found; skipping"; \
	fi

enginecoverage: build fmt
	./aiki enginesmoke --stage all --coverage test/structure/engine

enginecoverage-all: build fmt-all
	./aiki enginesmoke --all --stage all --coverage test/structure/engine

enginesmoke: build fmt
	./aiki enginesmoke --stage all --check test/structure/engine

enginesmoke-all: build fmt-all
	./aiki enginesmoke --all --stage all --check test/structure/engine

enginesmokegold: build fmt
	./aiki enginesmoke --stage all --gold test/structure/engine

# Snapshot-independent language/runtime correctness checks. This target never
# writes gold files and uses ordinary CLI target policy.
check: build fmt lint treecheck test invariant aikitest enginecoverage

# Exhaustive snapshot-independent repository checks. Repository selection remains
# owned by the CLI: --all bypasses .aikiignore without teaching Make which trees
# are normally excluded. go test ./... supplies the exhaustive ordinary Go suite.
check-all: build fmt-all lint-all treecheck
	go test ./...
	$(MAKE) aikitest-all
	./aiki enginesmoke --all --stage all --coverage test/structure/engine

# Bless the current, independently checked behavior and engine structure.
# Blessing records a reference state; it is not validation.
bless: check
	./aiki smoke --gold test/behavior/
	./aiki enginesmoke --stage all --gold test/structure/engine

# Routine language/runtime repository gate. Ordinary CLI policy applies, so
# repository test exclusions are honored without Make knowing their paths.
validate-lang: check
	./aiki smoke test/behavior/
	./aiki enginesmoke --stage all --check test/structure/engine

# Exhaustive applicable repository gate. --all changes target-policy selection
# only; each CLI command retains its own source/fixture semantics.
validate-all: check-all
	./aiki smoke --all test/behavior/
	./aiki enginesmoke --all --stage all --check test/structure/engine


fuzz:
	go test -fuzz=FuzzLexer ./test/fuzz/ -fuzztime=30s
	go test -fuzz=FuzzParser ./test/fuzz/ -fuzztime=30s

rigorous: validate-all fuzz



# Reproducible semantic/substrate profiling sweep. Not part of validation.
intrinsicgate: build
	./engineering/profiling/state-machine-intrinsic-gate.sh

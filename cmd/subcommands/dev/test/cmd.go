// Package test implements the `aiki test` subcommand.
//
// Usage:
//
//	aiki test ./...
//	aiki test lib/math
//	aiki test path/to/foo_test.ai
//	aiki test -cover ./...
//	aiki test -cover -coverview=file ./...
//	aiki test -coverprofile=coverage.out ./...
package test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aiki/cmd/internal/aitest"
	"aiki/cmd/internal/targetresolve"
	"aiki/engine/coverage"
	"aiki/engine/observe"
	"aiki/engine/runner"
	"aiki/engine/runtime/hal/substrate"
	"aiki/engine/syntax"
	"aiki/engine/syntax/grammar"
)

// Run is the entry point for the `aiki test` subcommand.
func Run(args []string) int {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	cover := fs.Bool("cover", false, "enable coverage analysis")
	coverprofile := fs.String("coverprofile", "", "write coverage profile to file")
	coverview := fs.String("coverview", "summary", "coverage view: summary, file, function, module, layer, scope")
	allTargets := fs.Bool("all", false, "include targets excluded by .aikiignore")
	fs.Parse(args)

	if *coverprofile != "" {
		*cover = true
	}
	if *cover && !validCoverageView(*coverview) {
		fmt.Fprintf(os.Stderr, "aiki test: invalid -coverview %q (want summary, file, function, module, layer, or scope)\n", *coverview)
		return 1
	}

	targets := fs.Args()
	if len(targets) == 0 {
		targets = []string{"."}
	}

	files, err := discoverTestsWithPolicy(targets, *allTargets)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aiki test:", err)
		return 1
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "aiki test: no *_test.ai files found")
		return 1
	}

	// Coverage spans many independent test-program executions. Each execution gets
	// its own bounded nervous-system capture; immutable observations are merged
	// only after that execution completes. This keeps network capacity and
	// identity lifetime aligned with the program being observed rather than
	// accumulating unrelated AST identities across the entire repository run.
	var coverageHits []observe.EvalSiteCount

	totalPassed := 0
	totalFailed := 0
	anyFailed := false

	for _, f := range files {
		rt := substrate.NewGoRuntime()
		rt.ResetTestState()
		rt.SetTestFile(f)

		sourceBytes, readErr := os.ReadFile(f)
		if readErr != nil {
			rt.CloseAllResources()
			fmt.Fprintf(os.Stderr, "FAIL %s\n    reading file: %v\n", f, readErr)
			anyFailed = true
			totalFailed++
			continue
		}
		testSource := aitest.Source + "\n" + string(sourceBytes)

		var runErr error
		if *cover {
			schema := observe.NewObservationSchema()
			network := observe.NewNetwork(observe.NewNodeRegistry(), schema, observe.NetworkConfig{
				Depth: observe.DepthFlat, NodeHitCapacity: 1 << 18,
			})
			evalCapture := observe.NewEvalCapture(network)
			runErr = runner.RunSourceWithEvalProbeRuntime(f, testSource, evalCapture, rt)
			if runErr == nil {
				coverageHits = append(coverageHits, evalCapture.Hits()...)
			}
		} else {
			runErr = runner.RunSourceWithRuntime(f, testSource, rt)
		}

		if runErr != nil {
			rt.CloseAllResources()
			fmt.Fprintf(os.Stderr, "FAIL %s\n    %v\n", f, runErr)
			anyFailed = true
			totalFailed++
			continue
		}

		passed, failed, failures := rt.TestResults()
		rt.CloseAllResources()
		totalPassed += passed
		totalFailed += failed

		if failed > 0 {
			fmt.Fprintf(os.Stdout, "FAIL %s\n", f)
			for _, msg := range failures {
				fmt.Fprintln(os.Stdout, msg)
			}
			anyFailed = true
		} else {
			fmt.Fprintf(os.Stdout, "PASS %s\n", f)
		}
	}

	fmt.Fprintf(os.Stdout, "\n%d tests, %d passed, %d failed\n", totalPassed+totalFailed, totalPassed, totalFailed)

	if *cover {
		hits := coverageHits
		model, modelErr := buildCoverageModelWithPolicy(targets, hits, *allTargets)
		if modelErr != nil {
			fmt.Fprintf(os.Stderr, "aiki test: coverage model: %v\n", modelErr)
			return 1
		}
		report := coverage.Analyze(model, hits)
		printCoverage(report, *coverview)
		if *coverprofile != "" {
			if err := writeCoverageProfile(*coverprofile, coverage.HitCounts(hits)); err != nil {
				fmt.Fprintf(os.Stderr, "aiki test: writing coverage profile: %v\n", err)
			}
		}
	}

	if anyFailed {
		return 1
	}
	return 0
}

func validCoverageView(v string) bool {
	switch v {
	case "summary", "file", "function", "module", "layer", "scope":
		return true
	default:
		return false
	}
}

func printCoverage(report coverage.Report, view string) {
	fmt.Fprintln(os.Stdout, "")
	if view == "summary" {
		fmt.Fprintln(os.Stdout, "Coverage summary")
		printRollups("", coverageByScope(report))
		t := report.Total()
		printRollup("all executable Aiki", t)

		fmt.Fprintln(os.Stdout, "\nSupported libraries")
		printRollups("", supportedLibraryCoverage(report))
		return
	}

	var title string
	var rows []coverage.Rollup
	switch view {
	case "file":
		title, rows = "Coverage by file", report.ByFile()
	case "function":
		title, rows = "Coverage by function", report.ByFunction()
	case "module":
		title, rows = "Coverage by supported library", supportedLibraryCoverage(report)
	case "layer":
		title, rows = "Coverage by layer", report.ByLayer()
	case "scope":
		title, rows = "Coverage by scope", coverageByScope(report)
	}
	fmt.Fprintln(os.Stdout, title)
	printRollups("", rows)
}

// coverageScope is repository policy, not part of the portable coverage model or
// profiling-network identity. It maps this repository's top-level source roles
// onto report populations; other codebases can apply different path policy.
func coverageScope(file string) string {
	p := filepath.ToSlash(filepath.Clean(file))
	parts := strings.Split(p, "/")
	for _, part := range parts {
		switch part {
		case "lib":
			return "supported"
		case "samples", "examples":
			return "examples"
		}
	}
	return "auxiliary"
}

func coverageByScope(report coverage.Report) []coverage.Rollup {
	return report.Rollup(func(line coverage.LineRecord) string { return coverageScope(line.File) }, nil)
}

func supportedLibraryCoverage(report coverage.Report) []coverage.Rollup {
	return report.Rollup(
		func(line coverage.LineRecord) string { return line.Module },
		func(line coverage.LineRecord) bool { return coverageScope(line.File) == "supported" },
	)
}

func printRollups(prefix string, rows []coverage.Rollup) {
	for _, row := range rows {
		printRollup(prefix+row.Name, row)
	}
}

func printRollup(name string, row coverage.Rollup) {
	if row.Total == 0 {
		return
	}
	pct := float64(row.Hit) / float64(row.Total) * 100
	fmt.Fprintf(os.Stdout, "  %-40s %5.1f%% (%d/%d lines)\n", name, pct, row.Hit, row.Total)
}

// buildCoverageModel constructs static anatomy from source selected by the test
// targets, then adds any dynamically observed dependency source outside those
// targets. The static side therefore includes wholly unexecuted source in the
// requested scope without losing imported dependencies.
func buildCoverageModel(targets []string, hits []observe.EvalSiteCount) (coverage.Model, error) {
	return buildCoverageModelWithPolicy(targets, hits, false)
}

func buildCoverageModelWithPolicy(targets []string, hits []observe.EvalSiteCount, all bool) (coverage.Model, error) {
	sources, err := discoverCoverageSourcesWithPolicy(targets, all)
	if err != nil {
		return coverage.Model{}, err
	}
	for _, hit := range hits {
		f := hit.Site.File
		if !coverableSourcePath(f) {
			continue
		}
		if _, err := os.Stat(f); err == nil {
			sources[f] = true
		}
	}

	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		return coverage.Model{}, fmt.Errorf("loading grammar: %w", err)
	}
	paths := make([]string, 0, len(sources))
	for path := range sources {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var model coverage.Model
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return coverage.Model{}, fmt.Errorf("reading %s: %w", path, err)
		}
		lexer := syntax.NewLexer(g, path, string(data), nil)
		tokens, err := lexer.Tokenize()
		if err != nil {
			return coverage.Model{}, fmt.Errorf("lexer %s: %w", path, err)
		}
		parser := syntax.NewParser(g, tokens, string(data), nil)
		ast, err := parser.Parse()
		if err != nil {
			return coverage.Model{}, fmt.Errorf("parser %s: %w", path, err)
		}
		part := coverage.FromAST(ast)
		model.Sites = append(model.Sites, part.Sites...)
	}
	return model, nil
}

func discoverCoverageSources(targets []string) (map[string]bool, error) {
	return discoverCoverageSourcesWithPolicy(targets, false)
}

func discoverCoverageSourcesWithPolicy(targets []string, all bool) (map[string]bool, error) {
	coverageTargets := make([]string, 0, len(targets))
	for _, target := range targets {
		if strings.HasSuffix(target, "/...") || strings.HasSuffix(target, string(filepath.Separator)+"...") || target == "..." {
			coverageTargets = append(coverageTargets, target)
			continue
		}
		info, err := os.Stat(target)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			coverageTargets = append(coverageTargets, target)
		} else {
			coverageTargets = append(coverageTargets, filepath.Join(filepath.Dir(target), "..."))
		}
	}
	var paths []string
	var err error
	if all {
		paths, err = targetresolve.ResolveAll("test", coverageTargets, true)
	} else {
		paths, err = targetresolve.Resolve("test", coverageTargets, true)
	}
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool)
	for _, path := range paths {
		if coverableSourcePath(path) {
			out[path] = true
		}
	}
	return out, nil
}

func coverableSourcePath(path string) bool {
	if path == "" || strings.HasPrefix(path, "<") || !strings.HasSuffix(path, ".ai") || strings.HasSuffix(path, "_test.ai") {
		return false
	}
	p := filepath.ToSlash(filepath.Clean(path))
	parts := strings.Split(p, "/")
	for _, part := range parts {
		if part == ".git" || part == "ai" || part == "test" {
			return false
		}
	}
	if strings.HasSuffix(p, "engine/runtime/prelude/prelude.ai") {
		return false
	}
	return true
}

func writeCoverageProfile(path string, hits map[string]map[int]int64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "mode: count")
	var files []string
	for file := range hits {
		files = append(files, file)
	}
	sort.Strings(files)
	for _, file := range files {
		var lines []int
		for line := range hits[file] {
			lines = append(lines, line)
		}
		sort.Ints(lines)
		for _, line := range lines {
			fmt.Fprintf(f, "%s:%d.1,%d.1 1 %d\n", file, line, line, hits[file][line])
		}
	}
	return nil
}

func discoverTests(targets []string) ([]string, error) {
	return discoverTestsWithPolicy(targets, false)
}

func discoverTestsWithPolicy(targets []string, all bool) ([]string, error) {
	var paths []string
	var err error
	if all {
		paths, err = targetresolve.ResolveAll("test", targets, true)
	} else {
		paths, err = targetresolve.Resolve("test", targets, true)
	}
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.ai") {
			files = append(files, path)
		}
	}
	return files, nil
}

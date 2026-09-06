package substrate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aiki/engine/runtime/hal"
	"aiki/engine/runtime/modules"
	"aiki/engine/syntax"
	"aiki/engine/syntax/grammar"
)

func TestStripDocMarkers(t *testing.T) {
	in := "@preamble use(\"turtle\")\nsqrt\n@unchecked\nComputes a square root.\n\nsqrt(:red)"
	want := "sqrt\nComputes a square root.\n\nsqrt(:red)"
	if got := stripDocMarkers(in); got != want {
		t.Fatalf("stripDocMarkers() = %q, want %q", got, want)
	}
}

func TestStripDocMarkersLeavesOrdinaryAtLines(t *testing.T) {
	in := "example\n  @unchecked\n@other marker\ntext"
	want := in
	if got := stripDocMarkers(in); got != want {
		t.Fatalf("stripDocMarkers() = %q, want %q", got, want)
	}
}

func TestModuleHelpCarriesPreamble(t *testing.T) {
	mh := &modules.ModuleHelp{Preamble: `use("math/native")`}
	if got, want := mh.Preamble, `use("math/native")`; got != want {
		t.Fatalf("ModuleHelp.Preamble = %q, want %q", got, want)
	}
}

func TestOutputHelpUsesPagerHook(t *testing.T) {
	var out bytes.Buffer
	rt := NewGoRuntime()
	rt.SetIO(nil, &out)
	var got string
	rt.SetPageOutput(func(text string) bool {
		got = text
		return true
	})

	rt.outputHelp("long help\n")
	if got != "long help\n" {
		t.Fatalf("pager got %q", got)
	}
	if out.Len() != 0 {
		t.Fatalf("Stdout received %q despite handled pager output", out.String())
	}
}

func TestOutputHelpFallsBackToStdout(t *testing.T) {
	var out bytes.Buffer
	rt := NewGoRuntime()
	rt.SetIO(nil, &out)
	rt.SetPageOutput(func(string) bool { return false })

	rt.outputHelp("short help\n")
	if got, want := out.String(), "short help\n"; got != want {
		t.Fatalf("Stdout = %q, want %q", got, want)
	}
}

func TestGeneralHelpListsAllBasicLanguageTypes(t *testing.T) {
	var out bytes.Buffer
	rt := NewGoRuntime()
	rt.SetIO(nil, &out)
	rt.SetPageOutput(func(string) bool { return false })

	rt.showHelpIndex(nil)

	want := "  Values: number, boolean, rune, string, symbol, list, function\n"
	if got := out.String(); !strings.Contains(got, want) {
		t.Fatalf("general help missing basic language types: want %q in\n%s", want, got)
	}
}

func TestGrammarNewlineHelpUsesDeclaredMetadata(t *testing.T) {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatal(err)
	}
	if g.Newline.Meta.Help == "" {
		t.Fatal("grammar newline policy has no @help metadata")
	}

	var out bytes.Buffer
	rt := NewGoRuntime()
	rt.SetIO(nil, &out)
	rt.SetPageOutput(func(string) bool { return false })

	ctx := &hal.EvalContext{Grammar: g}
	rt.showHelp("newline", ctx)

	want := "newline\n  " + g.Newline.Meta.Help + "\n"
	if got := out.String(); got != want {
		t.Fatalf("newline help = %q, want %q", got, want)
	}
}

func TestModuleHelpShowsCanonicalTargetForBareNativeAlias(t *testing.T) {
	root := t.TempDir()
	moduleDir := filepath.Join(root, "math")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "native.ai"), []byte("package \"math/native\"\n\nlet floor = (x) {\n\treturn x\n}\nexport(:floor)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "native.help"), []byte("@summary \"math/native test summary\"\n\n@func floor\n@template \"floor(x)\"\n@help \"test floor\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "native.doc"), []byte("floor\nTest floor documentation.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatal(err)
	}
	registry := modules.NewModuleRegistry([]string{root})
	if err := registry.Scan(g); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	rt := NewGoRuntime()
	rt.SetIO(nil, &out)
	rt.SetPageOutput(func(string) bool { return false })
	rt.SetModuleRegistry(registry)

	rt.showModuleHelp("math")
	got := out.String()
	if !strings.Contains(got, "Module: math\nResolves to: math/native\n") {
		t.Fatalf("bare module help did not expose canonical resolution:\n%s", got)
	}

	out.Reset()
	rt.showModuleHelp("math/native")
	got = out.String()
	if strings.Contains(got, "Resolves to:") {
		t.Fatalf("canonical module help unexpectedly reported alias resolution:\n%s", got)
	}

	out.Reset()
	rt.showModuleFuncHelp("math.floor")
	got = out.String()
	if !strings.Contains(got, "math.floor\nResolves to: math/native.floor\n") {
		t.Fatalf("bare module function help did not expose canonical resolution:\n%s", got)
	}

	out.Reset()
	rt.showModuleFuncHelp("math/native.floor")
	got = out.String()
	if strings.Contains(got, "Resolves to:") {
		t.Fatalf("canonical module function help unexpectedly reported alias resolution:\n%s", got)
	}
}

func TestModuleDocShowsCanonicalTargetForBareNativeAlias(t *testing.T) {
	root := t.TempDir()
	moduleDir := filepath.Join(root, "math")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "native.ai"), []byte("package \"math/native\"\n\nlet floor = (x) {\n\treturn x\n}\nexport(:floor)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "native.help"), []byte("@summary \"math/native test summary\"\n\n@func floor\n@template \"floor(x)\"\n@help \"test floor\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "native.doc"), []byte("floor\nTest floor documentation.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatal(err)
	}
	registry := modules.NewModuleRegistry([]string{root})
	if err := registry.Scan(g); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	rt := NewGoRuntime()
	rt.SetIO(nil, &out)
	rt.SetPageOutput(func(string) bool { return false })
	rt.SetModuleRegistry(registry)

	rt.showModuleDoc("math")
	got := out.String()
	if !strings.Contains(got, "Module: math\nResolves to: math/native\n") {
		t.Fatalf("bare module doc did not expose canonical resolution:\n%s", got)
	}
	if !strings.Contains(got, "math/native test summary\n") {
		t.Fatalf("bare module doc did not expose module summary:\n%s", got)
	}

	out.Reset()
	rt.showModuleDoc("math/native")
	got = out.String()
	if !strings.Contains(got, "Module: math/native\n\nmath/native test summary\n") {
		t.Fatalf("canonical module doc did not expose module summary:\n%s", got)
	}

	out.Reset()
	rt.showModuleFuncDoc("math.floor")
	got = out.String()
	if !strings.Contains(got, "math.floor\nResolves to: math/native.floor\n") {
		t.Fatalf("bare module function doc did not expose canonical resolution:\n%s", got)
	}

	out.Reset()
	rt.showModuleFuncDoc("math/native.floor")
	got = out.String()
	if strings.Contains(got, "Resolves to:") {
		t.Fatalf("canonical module function doc unexpectedly reported alias resolution:\n%s", got)
	}
}

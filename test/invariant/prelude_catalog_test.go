package invariant

import (
	"strings"
	"testing"

	"nezha/engine/runtime/prelude"
	"nezha/engine/syntax"
	"nezha/engine/syntax/grammar"
)

func invariantGrammar(t *testing.T) *grammar.Grammar {
	t.Helper()
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatalf("loading grammar: %v", err)
	}
	return g
}

func TestPreludeCatalogJoinsSourceHelpAndDocs(t *testing.T) {
	catalog, err := prelude.LoadCatalog(invariantGrammar(t))
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	for _, name := range catalog.Names {
		if catalog.Registry.GetHelp(name) == nil {
			t.Errorf("%s missing help", name)
		}
		if catalog.Registry.GetDoc(name) == nil {
			t.Errorf("%s missing doc", name)
		}
	}
}

func TestPreludeInvariantRejectsMissingHelp(t *testing.T) {
	helpSource := strings.Replace(prelude.HelpSource,
		"@func abs\n@template \"abs(n)\"\n@help \"Returns absolute value of number.\"\n", "", 1)
	if helpSource == prelude.HelpSource {
		t.Fatal("fixture did not remove abs help")
	}
	_, err := prelude.ValidateSources(invariantGrammar(t), prelude.Source, helpSource, prelude.DocSource)
	if err == nil || !strings.Contains(err.Error(), "missing help for 'abs'") {
		t.Fatalf("expected missing help failure, got %v", err)
	}
}

func TestPreludeInvariantRejectsMissingDoc(t *testing.T) {
	docSource := strings.Replace(prelude.DocSource,
		"abs\nReturns the absolute value of a number.\n\nabs(n)\n\nabs(-5)    # 5\nabs(3)     # 3\nabs(0)     # 0\n===\n", "", 1)
	if docSource == prelude.DocSource {
		t.Fatal("fixture did not remove abs doc")
	}
	_, err := prelude.ValidateSources(invariantGrammar(t), prelude.Source, prelude.HelpSource, docSource)
	if err == nil || !strings.Contains(err.Error(), "missing doc for 'abs'") {
		t.Fatalf("expected missing doc failure, got %v", err)
	}
}

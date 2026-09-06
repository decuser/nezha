package coverage

import (
	"path/filepath"
	"testing"

	"aiki/engine/observe"
)

func TestAnalyzeCombinesStaticLinesWithNetworkEvalHits(t *testing.T) {
	file := filepath.Join(t.TempDir(), "lib", "demo", "demo.ai")
	model := Model{Sites: []Site{
		{File: file, Line: 1, Col: 1, Node: "let_stmt", Function: "<main>", Module: "demo", Layer: "semantic"},
		{File: file, Line: 2, Col: 2, Node: "expr", Function: "f", Module: "demo", Layer: "semantic"},
		{File: file, Line: 2, Col: 4, Node: "NAME", Function: "f", Module: "demo", Layer: "semantic"},
		{File: file, Line: 3, Col: 1, Node: "return_stmt", Function: "f", Module: "demo", Layer: "semantic"},
	}}
	hits := []observe.EvalSiteCount{{Site: observe.SemanticSite{File: file, Line: 2}, Count: 7}}
	report := Analyze(model, hits)
	if len(report.Lines) != 3 {
		t.Fatalf("lines=%d want 3", len(report.Lines))
	}
	var hit, total int
	for _, line := range report.Lines {
		total++
		if line.Covered {
			hit++
		}
	}
	if hit != 1 || total != 3 {
		t.Fatalf("coverage=%d/%d want 1/3", hit, total)
	}
	modules := report.ByModule()
	if len(modules) != 1 || modules[0].Name != "demo" || modules[0].Hit != 1 || modules[0].Total != 3 {
		t.Fatalf("module rollup=%+v", modules)
	}
}

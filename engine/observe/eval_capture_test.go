package observe

import "testing"

func TestEvalCaptureCachesBindingAndUsesNetworkNodeHits(t *testing.T) {
	schema := NewObservationSchema()
	registry := NewNodeRegistry()
	network := NewNetwork(registry, schema, NetworkConfig{NodeHitCapacity: 64})
	capture := NewEvalCapture(network)

	var expr, name NodeBinding
	site := SemanticSite{File: "lib/demo/demo.nez", Line: 7, Col: 3, Function: "f", Source: "x + 1"}
	if capture.Hit(&expr) {
		t.Fatal("unbound AST node unexpectedly hit")
	}
	capture.BindAndHit("expr", site, &expr)
	if !capture.Hit(&expr) || !capture.Hit(&expr) {
		t.Fatal("cached AST binding did not take steady-state path")
	}
	capture.BindAndHit("NAME", SemanticSite{File: site.File, Line: 7, Col: 3, Function: "f", Source: site.Source}, &name)

	hits := capture.Hits()
	if len(hits) != 2 {
		t.Fatalf("hits = %#v, want two distinct AST evaluation sites", hits)
	}
	var exprCount, nameCount int64
	for _, hit := range hits {
		switch hit.Node {
		case "expr":
			exprCount = hit.Count
		case "NAME":
			nameCount = hit.Count
		}
		if hit.Site.File != site.File {
			t.Fatalf("site identity lost: %#v", hit)
		}
		if hit.Site.Function != "" {
			t.Fatalf("dynamic frame leaked into static eval-site identity: %#v", hit)
		}
	}
	if exprCount != 3 || nameCount != 1 {
		t.Fatalf("eval counts expr=%d name=%d, want 3 and 1", exprCount, nameCount)
	}
}

func TestNodeHitSteadyStateAllocatesZero(t *testing.T) {
	schema := NewObservationSchema()
	network := NewNetwork(NewNodeRegistry(), schema, NetworkConfig{NodeHitCapacity: 8})
	capture := NewEvalCapture(network)
	var binding NodeBinding
	capture.BindAndHit("expr", SemanticSite{File: "x.nez", Line: 1, Col: 1}, &binding)
	allocs := testing.AllocsPerRun(1000, func() {
		if !capture.Hit(&binding) {
			t.Fatal("binding unexpectedly missed")
		}
	})
	if allocs != 0 {
		t.Fatalf("steady-state Eval hit allocations = %v, want 0", allocs)
	}
}

package substrate

import (
	"testing"

	"nezha/engine"
	"nezha/engine/observe"
	"nezha/engine/runtime/hal"
	"nezha/engine/semantics/value"
)

func TestProfileTimedReturnsCountsElapsedAndResolvedTimingRows(t *testing.T) {
	caller := observe.Node{ID: 1, Identity: observe.NodeIdentity{Kind: observe.NodeFunction, Name: "caller", File: "a.ai"}, Layer: "semantic", Boundary: "function"}
	callee := observe.Node{ID: 2, Identity: observe.NodeIdentity{Kind: observe.NodePrimitive, Name: "_x", File: "b.ai"}, Module: "m", Layer: "runtime", Boundary: "provider"}
	ctx := &hal.EvalContext{
		MeasureTimed: func(fn value.Value, args []value.Value, attributed bool) (value.Value, engine.SemanticMeasurement) {
			return value.EMPTY, engine.SemanticMeasurement{
				Counts:        engine.SemanticCounts{Call: 1},
				HostElapsedNS: 200,
				Graph: observe.GraphMeasurement{
					Nodes:   []observe.Node{caller, callee},
					Edges:   []observe.EdgeCount{{Caller: 1, Callee: 2, Count: 1}},
					Timings: []observe.EdgeTiming{{Caller: 1, Callee: 2, InclusiveNS: 150, SelfNS: 100}},
				},
			}
		},
	}
	result := halProfileTimed([]value.Value{value.EMPTY}, ctx)
	list, ok := result.(*value.List)
	if !ok || len(list.Elements) != 3 {
		t.Fatalf("result=%s", result.Inspect())
	}
	if got := list.Elements[1].Inspect(); got != "200" {
		t.Fatalf("elapsed=%s want 200", got)
	}
	rows, ok := list.Elements[2].(*value.List)
	if !ok || len(rows.Elements) != 1 {
		t.Fatalf("timing rows=%s", list.Elements[2].Inspect())
	}
	row := rows.Elements[0].(*value.List)
	if len(row.Elements) != 10 {
		t.Fatalf("timing row length=%d", len(row.Elements))
	}
	if row.Elements[2].Inspect() != "1" || row.Elements[3].Inspect() != "150" || row.Elements[4].Inspect() != "100" || row.Elements[5].Inspect() != "150" {
		t.Fatalf("timing row=%s", row.Inspect())
	}
}

package runner

import (
	"os"
	"path/filepath"
	"testing"

	"nezha/engine/observe"
)

func writeProfileTimingProgram(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "profile-timing.nez")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("write program: %v", err)
	}
	return path
}

func TestRunProfileTimingAttributesCallsWithoutChangingCallAuthority(t *testing.T) {
	path := writeProfileTimingProgram(t, `
let child = () {
	1 + 2
}
let parent = () {
	child()
	child()
}
parent()
`)
	run, err := RunProfileDetailed(path, ProfileOptions{Attributed: false, Depth: 1, Timing: true})
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if run.Semantic.HostElapsedNS <= 0 {
		t.Fatalf("elapsed=%d", run.Semantic.HostElapsedNS)
	}
	if len(run.Semantic.Graph.Timings) == 0 {
		t.Fatal("no timing rows")
	}
	rows := observe.NodeTimingRows(run.Semantic.Graph)
	var child observe.NodeTimingRow
	for _, row := range rows {
		if row.Node.Identity.Name == "child" {
			child = row
		}
	}
	if child.Count != 2 {
		t.Fatalf("child calls=%d want 2", child.Count)
	}
	if child.InclusiveNS <= 0 || child.SelfNS <= 0 {
		t.Fatalf("child timing inclusive=%d self=%d", child.InclusiveNS, child.SelfNS)
	}
}

func TestRunProfileWithoutTimingKeepsTimingStorageDisabled(t *testing.T) {
	path := writeProfileTimingProgram(t, `
let work = () { 1 + 2 }
work()
`)
	run, err := RunProfileDetailed(path, ProfileOptions{Attributed: false, Depth: 1})
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if len(run.Semantic.Graph.Timings) != 0 {
		t.Fatalf("untimed run captured %d timing rows", len(run.Semantic.Graph.Timings))
	}
}

func TestRunProfileTimingTreatsTailCallsAsSuccessiveSegments(t *testing.T) {
	path := writeProfileTimingProgram(t, `
let loop = (n, acc) {
	if n <= 0 {
		return acc
	}
	return loop(n - 1, acc + n)
}
loop(20, 0)
`)
	run, err := RunProfileDetailed(path, ProfileOptions{Attributed: false, Depth: 1, Timing: true})
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	rows := observe.NodeTimingRows(run.Semantic.Graph)
	for _, row := range rows {
		if row.Node.Identity.Name == "loop" {
			if row.Count != 21 {
				t.Fatalf("loop timed calls=%d want 21", row.Count)
			}
			if row.SelfNS <= 0 || row.InclusiveNS <= 0 {
				t.Fatalf("loop timing self=%d inclusive=%d", row.SelfNS, row.InclusiveNS)
			}
			return
		}
	}
	t.Fatal("no loop timing row")
}

func TestRunProfileTimingApplyPreservesNestedSelfAttribution(t *testing.T) {
	path := writeProfileTimingProgram(t, `
let child = () {
	let x = 0
	let i = 0
	while i < 200 {
		x = x + i
		i = i + 1
	}
	x
}
let parent = () {
	child()
}
apply(parent, [])
`)
	run, err := RunProfileDetailed(path, ProfileOptions{Attributed: false, Depth: 1, Timing: true})
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	rows := observe.NodeTimingRows(run.Semantic.Graph)
	var parent, child observe.NodeTimingRow
	for _, row := range rows {
		switch row.Node.Identity.Name {
		case "parent":
			parent = row
		case "child":
			child = row
		}
	}
	if parent.Count != 1 || child.Count != 1 {
		t.Fatalf("calls parent=%d child=%d want 1/1", parent.Count, child.Count)
	}
	if child.InclusiveNS <= 0 || child.SelfNS <= 0 {
		t.Fatalf("child timing inclusive=%d self=%d", child.InclusiveNS, child.SelfNS)
	}
	if parent.InclusiveNS <= child.InclusiveNS {
		t.Fatalf("parent inclusive=%d child inclusive=%d", parent.InclusiveNS, child.InclusiveNS)
	}
	if parent.SelfNS >= parent.InclusiveNS {
		t.Fatalf("apply nested child was not subtracted: parent self=%d inclusive=%d", parent.SelfNS, parent.InclusiveNS)
	}
}

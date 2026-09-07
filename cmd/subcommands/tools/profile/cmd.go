package profile

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"nezha/engine"
	"nezha/engine/observe"
	"nezha/engine/runner"
)

func Run(args []string) int {
	fs := flag.NewFlagSet("profile", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	countsOnly := fs.Bool("counts", false, "show semantic totals without source-site attribution")
	timing := fs.Bool("time", false, "attribute inclusive/self host time to profiling-network calls")
	cpuPath := fs.String("cpu", "", "write a correlated Go CPU profile")
	allocsPath := fs.String("allocs", "", "write a Go allocation profile (pprof labels do not apply)")
	allocTarget := fs.String("alloc-target", "", "attribute allocator deltas to calls of one primitive (requires -depth 1+)")
	tracePath := fs.String("trace", "", "write a Go runtime trace for the measured evaluation")
	depthText := fs.String("depth", "0", "profiling graph depth: 0, N, or full")
	callers := fs.String("callers", "", "show direct callers of a function or primitive")
	from := fs.String("from", "", "show graph reachable from a function or primitive")
	hops := fs.Int("hops", -1, "maximum traversal hops for -from (default: -depth; 0 means unrestricted)")
	top := fs.Int("top", 20, "maximum rows in graph summary sections")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: nezha profile [-counts] [-time] [-depth 0|N|full] [-callers name] [-from name] [-hops N] [-alloc-target primitive] [-cpu file] [-allocs file] [-trace file] file.nez")
		return 2
	}
	depth, full, err := parseDepth(*depthText)
	if err != nil {
		fmt.Fprintln(os.Stderr, "profile:", err)
		return 2
	}
	effectiveDepth := depth
	if *timing && effectiveDepth == 0 {
		effectiveDepth = 1
	}
	if (*callers != "" || *from != "") && effectiveDepth == 0 {
		fmt.Fprintln(os.Stderr, "profile: -callers and -from require -depth 1 or greater")
		return 2
	}
	if *allocTarget != "" && effectiveDepth == 0 {
		fmt.Fprintln(os.Stderr, "profile: -alloc-target requires -depth 1 or greater")
		return 2
	}
	if *top < 1 {
		fmt.Fprintln(os.Stderr, "profile: -top must be at least 1")
		return 2
	}
	run, err := runner.RunProfileDetailed(fs.Arg(0), runner.ProfileOptions{
		Attributed:       !*countsOnly,
		CPUProfile:       *cpuPath,
		AllocsProfile:    *allocsPath,
		TraceFile:        *tracePath,
		Depth:            effectiveDepth,
		Timing:           *timing,
		AllocationTarget: *allocTarget,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "profile:", err)
		return 1
	}
	printMeasurement(run.Semantic, !*countsOnly)
	printSubstrate(run.Substrate)
	if effectiveDepth > 0 {
		printGraph(run.Semantic.Graph, *top)
		if *timing {
			printTiming(run.Semantic.Graph, run.Semantic.HostElapsedNS, *top)
		}
		if *allocTarget != "" {
			printAllocations(run.Semantic.Graph, *allocTarget, *top)
		}
		if *callers != "" {
			printCallers(run.Semantic.Graph, *callers)
		}
		if *from != "" {
			limit := *hops
			if limit < 0 {
				if full {
					limit = 0
				} else {
					limit = depth
				}
			}
			printTraversal(run.Semantic.Graph, *from, limit)
		}
	}
	if *cpuPath != "" {
		fmt.Printf("  cpu profile  %s (labels: nezha_layer, nezha_function, nezha_file, nezha_line, nezha_primitive)\n", *cpuPath)
	}
	if *allocsPath != "" {
		fmt.Printf("  alloc profile %s (Go allocation sites; not Nezha-label correlated)\n", *allocsPath)
	}
	if *tracePath != "" {
		fmt.Printf("  runtime trace %s\n", *tracePath)
	}
	return 0
}

func printMeasurement(m engine.SemanticMeasurement, showSites bool) {
	c := m.Counts
	fmt.Println("Nezha semantic work")
	rows := []struct {
		name string
		n    int64
	}{
		{"arithmetic", c.Arithmetic}, {"comparison", c.Comparison},
		{"call", c.Call}, {"iteration", c.Iteration}, {"index", c.Index},
		{"send", c.Send}, {"receive", c.Receive},
		{"store_read", c.StoreRead}, {"store_write", c.StoreWrite},
	}
	for _, row := range rows {
		fmt.Printf("  %-12s %d\n", row.name, row.n)
	}
	printNumberRealization("Number arithmetic realization", m.Numbers, true)
	printNumberRealization("Number call-return realization", m.CallNumbers, false)
	printCallRealization(m.Calls)
	printListRealization(m.Lists)
	printEnvRealization(m.Envs)
	if !showSites || len(m.Sites) == 0 {
		return
	}
	fmt.Println("\nNezha source attribution")
	for _, sc := range m.Sites {
		where := fmt.Sprintf("%s:%d:%d", sc.Site.File, sc.Site.Line, sc.Site.Col)
		context := ""
		if sc.Site.Function != "" {
			context = " in " + sc.Site.Function
		}
		if sc.Site.Detail != "" {
			context += " -> " + sc.Site.Detail
		}
		fmt.Printf("  %8d  %-12s %s%s\n", sc.Count, sc.Kind, where, context)
		if text := strings.TrimSpace(sc.Site.Source); text != "" {
			fmt.Printf("            %s\n", text)
		}
	}
}

func printCallRealization(c engine.CallRealizationCounts) {
	total := c.Function + c.Primitive + c.TailReuse + c.TailEnvReuse
	if total == 0 {
		return
	}
	fmt.Println("\nCall realization")
	fmt.Printf("  %-20s %d\n", "function", c.Function)
	fmt.Printf("  %-20s %d\n", "primitive", c.Primitive)
	fmt.Printf("  %-20s %d\n", "tail_reuse", c.TailReuse)
	fmt.Printf("  %-20s %d\n", "tail_env_reuse", c.TailEnvReuse)
	fmt.Printf("  %-20s %d\n", "arg_arity_0", c.ArgArity0)
	fmt.Printf("  %-20s %d\n", "arg_arity_1", c.ArgArity1)
	fmt.Printf("  %-20s %d\n", "arg_arity_2", c.ArgArity2)
	fmt.Printf("  %-20s %d\n", "arg_arity_3", c.ArgArity3)
	fmt.Printf("  %-20s %d\n", "arg_arity_4", c.ArgArity4)
	fmt.Printf("  %-20s %d\n", "arg_arity_5_plus", c.ArgArity5Plus)
	fmt.Printf("  %-20s %d\n", "args_evaluated", c.ArgsEvaluated)
	fmt.Printf("  %-20s %d\n", "arg_frame_new", c.ArgFrameNew)
	fmt.Printf("  %-20s %d\n", "arg_frame_reused", c.ArgFrameReused)
	fmt.Printf("  %-20s %d\n", "arg_frame_promoted", c.ArgFramePromoted)
	fmt.Printf("  %-20s %d\n", "arg_durable", c.ArgDurable)
	fmt.Printf("  %-20s %d\n", "arg_tail_transfer", c.ArgTailTransfer)
}

func printListRealization(c engine.ListRealizationCounts) {
	total := c.FrontierPromoted + c.FrontierExtended + c.FrontierForked
	if total == 0 {
		return
	}
	fmt.Println("\nList realization")
	fmt.Printf("  %-24s %d\n", "frontier_promoted", c.FrontierPromoted)
	fmt.Printf("  %-24s %d\n", "frontier_extended", c.FrontierExtended)
	fmt.Printf("  %-24s %d\n", "frontier_grown", c.FrontierGrown)
	fmt.Printf("  %-24s %d\n", "frontier_forked", c.FrontierForked)
	fmt.Printf("  %-24s %d\n", "elements_copied", c.ElementsCopied)
	fmt.Printf("  %-24s %d\n", "backing_slots_allocated", c.BackingSlotsAllocated)
}

func printEnvRealization(c engine.EnvRealizationCounts) {
	total := c.PhysicalCall + c.PhysicalEnclosed + c.PhysicalIsolated + c.LogicalCall
	if total == 0 {
		return
	}

	call0 := c.LogicalCall - c.CallReached1
	call1 := c.CallReached1 - c.CallReached2
	call2 := c.CallReached2 - c.CallReached3
	call34 := c.CallReached3 - c.CallReached5

	enclosed0 := c.PhysicalEnclosed - c.EnclosedReached1
	if enclosed0 < 0 {
		enclosed0 = 0
	}
	enclosed1 := c.EnclosedReached1 - c.EnclosedReached2
	enclosed2 := c.EnclosedReached2 - c.EnclosedReached3
	enclosed34 := c.EnclosedReached3 - c.EnclosedReached5

	fmt.Println("\nEnvironment realization")
	fmt.Printf("  %-28s %d\n", "physical_call", c.PhysicalCall)
	fmt.Printf("  %-28s %d\n", "logical_call", c.LogicalCall)
	fmt.Printf("  %-28s %d\n", "physical_enclosed", c.PhysicalEnclosed)
	fmt.Printf("  %-28s %d\n", "physical_isolated", c.PhysicalIsolated)

	fmt.Printf("  %-28s %d\n", "call_local_max_0", call0)
	fmt.Printf("  %-28s %d\n", "call_local_max_1", call1)
	fmt.Printf("  %-28s %d\n", "call_local_max_2", call2)
	fmt.Printf("  %-28s %d\n", "call_local_max_3_4", call34)
	fmt.Printf("  %-28s %d\n", "call_local_max_5_plus", c.CallReached5)

	fmt.Printf("  %-28s %d\n", "enclosed_local_max_0", enclosed0)
	fmt.Printf("  %-28s %d\n", "enclosed_local_max_1", enclosed1)
	fmt.Printf("  %-28s %d\n", "enclosed_local_max_2", enclosed2)
	fmt.Printf("  %-28s %d\n", "enclosed_local_max_3_4", enclosed34)
	fmt.Printf("  %-28s %d\n", "enclosed_local_max_5_plus", c.EnclosedReached5)

	fmt.Printf("  %-28s %d\n", "call_compact_allocations", c.CallCompactAllocations)
	fmt.Printf("  %-28s %d\n", "enclosed_compact_allocations", c.EnclosedCompactAllocations)
	fmt.Printf("  %-28s %d\n", "call_map_promotions", c.CallMapPromotions)
	fmt.Printf("  %-28s %d\n", "enclosed_map_promotions", c.EnclosedMapPromotions)
	fmt.Printf("  %-28s %d\n", "call_local_new", c.CallLocalNew)
	fmt.Printf("  %-28s %d\n", "call_local_update", c.CallLocalUpdate)
	fmt.Printf("  %-28s %d\n", "enclosed_local_new", c.EnclosedLocalNew)
	fmt.Printf("  %-28s %d\n", "enclosed_local_update", c.EnclosedLocalUpdate)
}

func printNumberRealization(title string, n engine.NumberRealizationCounts, arithmetic bool) {
	total := n.ResultSmallInteger + n.ResultCompactRational + n.ResultBinaryCarrier + n.ResultBigRational
	if total == 0 {
		return
	}
	fmt.Printf("\n%s\n", title)
	fmt.Printf("  %-20s %d\n", "small_integer", n.ResultSmallInteger)
	fmt.Printf("  %-20s %d\n", "compact_rational", n.ResultCompactRational)
	fmt.Printf("  %-20s %d\n", "binary_carrier", n.ResultBinaryCarrier)
	fmt.Printf("  %-20s %d\n", "big_rational", n.ResultBigRational)
	if arithmetic {
		fmt.Printf("  %-20s %d\n", "binary_certified", n.BinaryCertified)
		fmt.Printf("  %-20s %d\n", "binary_fallback", n.BinaryFallback)
		fmt.Printf("  %-20s %d\n", "promoted_big", n.PromotedBigRational)
	}
}

func printSubstrate(s runner.SubstrateStats) {
	fmt.Println("\nGo substrate realization")
	fmt.Printf("  %-12s %s\n", "elapsed", s.Elapsed)
	fmt.Printf("  %-12s %d\n", "alloc_bytes", s.AllocBytes)
	fmt.Printf("  %-12s %d\n", "mallocs", s.Mallocs)
	fmt.Printf("  %-12s %d\n", "gc_cycles", s.GCs)
}

func parseDepth(text string) (depth int, full bool, err error) {
	if text == "full" {
		return 1, true, nil
	}
	n, e := strconv.Atoi(text)
	if e != nil || n < 0 {
		return 0, false, fmt.Errorf("invalid -depth %q (want 0, positive integer, or full)", text)
	}
	return n, false, nil
}

func printGraph(g observe.GraphMeasurement, top int) {
	fmt.Println("\nProfiling network")
	fmt.Printf("  %-16s %d\n", "nodes", len(g.Nodes))
	fmt.Printf("  %-16s %d\n", "direct_edges", len(g.Edges))
	fmt.Printf("  %-16s %d\n", "edge_overflow", g.EdgeOverflow)

	printBoundaryRollup(g)
	printTopNodes("Top primitives", observe.TopPrimitives(g), top)
	printTopEdges(g, top)
	printRollup("File rollup", observe.FileRollup(g), top)
	printRollup("Module/library rollup", observe.ModuleRollup(g), top)
	printRollup("Layer rollup", observe.LayerRollup(g), top)
}

func printBoundaryRollup(g observe.GraphMeasurement) {
	rows := observe.BoundaryRollup(g)
	if len(rows) == 0 {
		return
	}
	fmt.Println("\nBoundary crossings")
	for _, row := range rows {
		fmt.Printf("  %-14s -> %-14s %d\n", row.From, row.To, row.Count)
	}
}

func printTopNodes(title string, rows []observe.NodeCallCount, top int) {
	if len(rows) == 0 {
		return
	}
	fmt.Printf("\n%s\n", title)
	if len(rows) > top {
		rows = rows[:top]
	}
	for _, row := range rows {
		fmt.Printf("  %-32s %d%s\n", observe.NodeDisplayName(row.Node), row.Count, nodeSuffix(row.Node))
	}
}

func printTopEdges(g observe.GraphMeasurement, top int) {
	rows := observe.TopEdges(g)
	if len(rows) == 0 {
		return
	}
	fmt.Println("\nCaller -> callee (top edges)")
	if len(rows) > top {
		rows = rows[:top]
	}
	for _, row := range rows {
		fmt.Printf("  %-24s -> %-24s %d\n", observe.NodeDisplayName(row.Caller), observe.NodeDisplayName(row.Callee), row.Count)
	}
}

func printRollup(title string, rows []observe.RollupCount, top int) {
	if len(rows) == 0 {
		return
	}
	fmt.Printf("\n%s\n", title)
	if len(rows) > top {
		rows = rows[:top]
	}
	for _, row := range rows {
		fmt.Printf("  %-44s %d calls (%d nodes)\n", row.Name, row.Count, row.Nodes)
	}
}

func printCallers(g observe.GraphMeasurement, query string) {
	rows := observe.CallersOf(g, query)
	fmt.Printf("\nCallers of %s\n", query)
	if len(rows) == 0 {
		fmt.Println("  <none>")
		return
	}
	for _, row := range rows {
		fmt.Printf("  %-32s %d%s\n", observe.NodeDisplayName(row.Caller), row.Count, nodeSuffix(row.Caller))
	}
}

func printTraversal(g observe.GraphMeasurement, query string, hops int) {
	rows := observe.TraverseFrom(g, query, hops)
	if hops > 0 {
		fmt.Printf("\nSubgraph from %s (depth %d)\n", query, hops)
	} else {
		fmt.Printf("\nSubgraph from %s (full)\n", query)
	}
	if len(rows) == 0 {
		fmt.Println("  <none>")
		return
	}
	for _, row := range rows {
		fmt.Printf("  %-24s -> %-24s %d\n", observe.NodeDisplayName(row.Caller), observe.NodeDisplayName(row.Callee), row.Count)
	}
}

func printTiming(g observe.GraphMeasurement, elapsedNS int64, top int) {
	rows := observe.TimingRows(g)
	fmt.Println("\nHost-time attribution")
	fmt.Printf("  %-16s %s\n", "elapsed", time.Duration(elapsedNS))
	fmt.Printf("  %-16s %d\n", "timed_edges", len(rows))
	fmt.Printf("  %-16s %d\n", "timing_overflow", g.TimingOverflow)
	if len(rows) == 0 {
		fmt.Println("  <no timed calls>")
		return
	}
	nodes := observe.NodeTimingRows(g)
	fmt.Println("\nTop timed nodes (by self time)")
	if len(nodes) > top {
		nodes = nodes[:top]
	}
	for _, row := range nodes {
		avg := time.Duration(0)
		if row.Count > 0 {
			avg = time.Duration(row.InclusiveNS / row.Count)
		}
		fmt.Printf("  %-32s calls %-9d self %-11s incl %-11s avg %s%s\n",
			observe.NodeDisplayName(row.Node), row.Count, time.Duration(row.SelfNS), time.Duration(row.InclusiveNS), avg, nodeSuffix(row.Node))
	}

	fmt.Println("\nTop timed caller -> callee (by self time)")
	if len(rows) > top {
		rows = rows[:top]
	}
	for _, row := range rows {
		caller := "<measurement>"
		if row.CallerID != 0 {
			caller = observe.NodeDisplayName(row.Caller)
		}
		callee := observe.NodeDisplayName(row.Callee)
		avg := time.Duration(0)
		if row.Count > 0 {
			avg = time.Duration(row.InclusiveNS / row.Count)
		}
		fmt.Printf("  %-22s -> %-22s calls %-9d self %-11s incl %-11s avg %s\n",
			caller, callee, row.Count, time.Duration(row.SelfNS), time.Duration(row.InclusiveNS), avg)
	}
	if crossings := observe.BoundaryCrossingTimings(g); len(crossings) > 0 {
		fmt.Println("\nTimed boundary crossings")
		if len(crossings) > top {
			crossings = crossings[:top]
		}
		for _, row := range crossings {
			fmt.Printf("  %-14s -> %-14s calls %-9d self %-11s incl %s\n",
				row.From, row.To, row.Calls, time.Duration(row.SelfNS), time.Duration(row.InclusiveNS))
		}
	}
	printTimingRollup("File timing rollup", observe.FileTimingRollup(g), top)
	printTimingRollup("Module/library timing rollup", observe.ModuleTimingRollup(g), top)
	printTimingRollup("Layer timing rollup", observe.LayerTimingRollup(g), top)
	printTimingRollup("Boundary timing rollup", observe.BoundaryTimingRollup(g), top)
}

func printTimingRollup(title string, rows []observe.TimingRollup, top int) {
	if len(rows) == 0 {
		return
	}
	fmt.Printf("\n%s\n", title)
	if len(rows) > top {
		rows = rows[:top]
	}
	for _, row := range rows {
		fmt.Printf("  %-40s self %-11s incl %-11s calls %-9d (%d nodes)\n",
			row.Name, time.Duration(row.SelfNS), time.Duration(row.InclusiveNS), row.Calls, row.Nodes)
	}
}

func printAllocations(g observe.GraphMeasurement, target string, top int) {
	rows := observe.AllocationEdges(g)
	fmt.Printf("\nTargeted allocation attribution for %s\n", target)
	if len(rows) == 0 {
		fmt.Println("  <none>")
	} else {
		if len(rows) > top {
			rows = rows[:top]
		}
		for _, row := range rows {
			fmt.Printf("  %-24s -> %-24s alloc_bytes=%d mallocs=%d\n", observe.NodeDisplayName(row.Caller), observe.NodeDisplayName(row.Callee), row.AllocBytes, row.Mallocs)
		}
	}
	if g.AllocationOverflow != 0 {
		fmt.Printf("  allocation_overflow %d\n", g.AllocationOverflow)
	}
}

func nodeSuffix(n observe.Node) string {
	loc := observe.NodeLocation(n)
	if loc == "" {
		return ""
	}
	return "  [" + loc + "]"
}

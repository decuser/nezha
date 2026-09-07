package coverage

import (
	"path/filepath"
	"sort"
	"strings"

	"nezha/engine/observe"
)

// LineRecord is one static executable source line and whether execution.eval
// was observed on it. Function is populated when all static sites on the line
// belong to one lexical function, otherwise "<mixed>".
type LineRecord struct {
	File     string
	Line     int
	Function string
	Module   string
	Layer    string
	Covered  bool
	Count    int64
}

// Report combines static anatomy with dynamic network observations.
type Report struct {
	Lines []LineRecord
}

// Analyze compares static sites with network-backed evaluation hits. Matching
// is line-based for compatibility with Nezha's existing coverage surface; the
// dynamic source remains the execution.eval event, not a second hit accountant.
func Analyze(model Model, hits []observe.EvalSiteCount) Report {
	hitLines := make(map[string]map[int]int64)
	for _, hit := range hits {
		file := canonicalFile(hit.Site.File)
		if file == "" || file == "<prelude>" || hit.Site.Line <= 0 {
			continue
		}
		if hitLines[file] == nil {
			hitLines[file] = make(map[int]int64)
		}
		hitLines[file][hit.Site.Line] += hit.Count
	}

	type lineKey struct {
		file string
		line int
	}
	type lineMeta struct {
		functions     map[string]bool
		module, layer string
	}
	metas := make(map[lineKey]*lineMeta)
	for _, site := range model.Sites {
		file := canonicalFile(site.File)
		if file == "" || file == "<prelude>" || site.Line <= 0 || strings.HasSuffix(file, "_test.nez") {
			continue
		}
		key := lineKey{file, site.Line}
		meta := metas[key]
		if meta == nil {
			meta = &lineMeta{functions: make(map[string]bool), module: site.Module, layer: site.Layer}
			metas[key] = meta
		}
		if site.Function != "" {
			meta.functions[site.Function] = true
		}
	}

	out := make([]LineRecord, 0, len(metas))
	for key, meta := range metas {
		fn := ""
		if len(meta.functions) == 1 {
			for name := range meta.functions {
				fn = name
			}
		} else if len(meta.functions) > 1 {
			fn = "<mixed>"
		}
		count := int64(0)
		if byLine := hitLines[key.file]; byLine != nil {
			count = byLine[key.line]
		}
		out = append(out, LineRecord{
			File: key.file, Line: key.line, Function: fn, Module: meta.module, Layer: meta.layer,
			Covered: count > 0, Count: count,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return Report{Lines: out}
}

func canonicalFile(file string) string {
	if file == "" || strings.HasPrefix(file, "<") {
		return file
	}
	abs, err := filepath.Abs(file)
	if err != nil {
		return filepath.Clean(file)
	}
	return filepath.Clean(abs)
}

// Rollup is one coverage population grouped by a static coordinate.
type Rollup struct {
	Name       string
	Hit, Total int
}

func (r Report) Rollup(key func(LineRecord) string, accept func(LineRecord) bool) []Rollup {
	m := map[string]*Rollup{}
	for _, line := range r.Lines {
		if accept != nil && !accept(line) {
			continue
		}
		name := key(line)
		if name == "" {
			continue
		}
		x := m[name]
		if x == nil {
			x = &Rollup{Name: name}
			m[name] = x
		}
		x.Total++
		if line.Covered {
			x.Hit++
		}
	}
	out := make([]Rollup, 0, len(m))
	for _, x := range m {
		out = append(out, *x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (r Report) ByFile() []Rollup { return r.Rollup(func(x LineRecord) string { return x.File }, nil) }
func (r Report) ByFunction() []Rollup {
	return r.Rollup(func(x LineRecord) string {
		if x.Function == "" {
			return ""
		}
		return x.File + ":" + x.Function
	}, nil)
}
func (r Report) ByModule() []Rollup {
	return r.Rollup(func(x LineRecord) string { return x.Module }, nil)
}
func (r Report) ByLayer() []Rollup {
	return r.Rollup(func(x LineRecord) string { return x.Layer }, nil)
}

func (r Report) Total() Rollup {
	out := Rollup{Name: "all"}
	for _, line := range r.Lines {
		out.Total++
		if line.Covered {
			out.Hit++
		}
	}
	return out
}

// HitCounts returns network-derived dynamic line counts for coverprofile output.
func HitCounts(hits []observe.EvalSiteCount) map[string]map[int]int64 {
	out := make(map[string]map[int]int64)
	for _, hit := range hits {
		file := canonicalFile(hit.Site.File)
		if file == "" || file == "<prelude>" || strings.HasSuffix(file, "_test.nez") || hit.Site.Line <= 0 {
			continue
		}
		if out[file] == nil {
			out[file] = make(map[int]int64)
		}
		out[file][hit.Site.Line] += hit.Count
	}
	return out
}

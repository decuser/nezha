package tags

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"aiki/cmd/internal/targetresolve"
	"aiki/engine/language"
	"aiki/engine/syntax"
	"aiki/engine/syntax/grammar"
)

func Run(args []string) int {
	fs := flag.NewFlagSet("tags", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	output := fs.String("o", "tags", "output tags file ('-' for stdout)")
	allTargets := fs.Bool("all", false, "include targets excluded by .aikiignore")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: aiki tags [-o file] <file.ai|dir|path/...> [more targets]")
		return 2
	}
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tags:", err)
		return 1
	}
	svc := language.NewService(g)
	files, err := expandWithPolicy(fs.Args(), *allTargets)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tags:", err)
		return 1
	}
	var lines []string
	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "tags:", err)
			return 1
		}
		syms, err := svc.Symbols(language.Document{Path: path, Source: string(b)})
		if err != nil {
			fmt.Fprintln(os.Stderr, "tags:", err)
			return 1
		}
		for _, sym := range syms {
			if !sym.TopLevel {
				continue
			}
			kind := "v"
			if sym.Kind == "shape" {
				kind = "s"
			}
			lines = append(lines, fmt.Sprintf("%s\t%s\t%d;\"\t%s", sym.Name, path, sym.Pos.Line, kind))
		}
	}
	sort.Strings(lines)
	text := strings.Join(lines, "\n")
	if len(lines) > 0 {
		text += "\n"
	}
	if *output == "-" {
		fmt.Print(text)
		return 0
	}
	if err := os.WriteFile(*output, []byte(text), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "tags:", err)
		return 1
	}
	return 0
}
func expand(args []string) ([]string, error) {
	return expandWithPolicy(args, false)
}

func expandWithPolicy(args []string, all bool) ([]string, error) {
	var paths []string
	var err error
	if all {
		paths, err = targetresolve.ResolveAll("tags", args, true)
	} else {
		paths, err = targetresolve.Resolve("tags", args, true)
	}
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.HasSuffix(path, ".ai") {
			out = append(out, path)
		}
	}
	return out, nil
}

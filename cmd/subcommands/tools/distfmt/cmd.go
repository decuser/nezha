package distfmt

import (
	"flag"
	"fmt"
	"os"
)

// Run applies Nezha's distribution/source presentation style.
//
// It intentionally mirrors `nezha fmt` command behavior:
//
//	-n     do not write; just list files that would change
//	-p     print restyled source to stdout (single file only; no write)
//	-b     create a .bak file before overwriting (only when changes occur)
func Run(args []string) int {
	fs := flag.NewFlagSet("gate distfmt", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	noWrite := fs.Bool("n", false, "do not write; list files that would change")
	printOut := fs.Bool("p", false, "print restyled output to stdout (single file)")
	backup := fs.Bool("b", false, "create .bak before overwriting")
	allTargets := fs.Bool("all", false, "include targets excluded by .nezhaignore")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: nezha gate distfmt <file.nez|file.go|path/...> [more targets]")
		return 2
	}
	if *printOut && fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "distfmt: -p requires exactly one file")
		return 2
	}

	cfg := Config{
		Write:         !*noWrite && !*printOut,
		ListOnly:      *noWrite,
		PrintToStdout: *printOut,
		Backup:        *backup,
	}

	changedAny := false
	for _, p := range fs.Args() {
		changed, err := formatTarget(p, cfg, *allTargets)
		if err != nil {
			fmt.Fprintln(os.Stderr, "distfmt:", err)
			return 1
		}
		if changed {
			changedAny = true
		}
	}
	if cfg.ListOnly && changedAny {
		return 1
	}
	return 0
}

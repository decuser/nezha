package smoke

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"nezha/cmd/internal/targetresolve"
	"nezha/cmd/internal/testfixture"
)

func Run(args []string) int {
	fs := flag.NewFlagSet("gate smoke", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	gold := fs.Bool("gold", false, "write blessed .gold transcripts")
	allTargets := fs.Bool("all", false, "include targets excluded by .nezhaignore")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	targets, err := collectSmokeFilesWithPolicy(fs.Args(), *allTargets)
	if err != nil {
		fmt.Fprintln(os.Stderr, "smoke:", err)
		return 1
	}
	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "smoke: no *_smoke.nez files found")
		return 1
	}

	for _, aiPath := range targets {
		if *gold {
			if err := blessPair(aiPath); err != nil {
				fmt.Fprintln(os.Stderr, "smoke BLESS FAIL:", aiPath)
				fmt.Fprintln(os.Stderr, "  ", err)
				return 1
			}
			continue
		}
		if err := runPair(aiPath); err != nil {
			fmt.Fprintln(os.Stderr, "smoke FAIL:", aiPath)
			fmt.Fprintln(os.Stderr, "  ", err)
			return 1
		}
	}

	if *gold {
		fmt.Fprintf(os.Stdout, "smoke bless ok (%d tests)\n", len(targets))
	} else {
		fmt.Fprintf(os.Stdout, "smoke ok (%d tests)\n", len(targets))
	}
	return 0
}

func blessPair(aiPath string) error {
	goldPath := strings.TrimSuffix(aiPath, ".nez") + ".gold"
	negativeKind, err := testfixture.NegativeKindOf(aiPath)
	if err != nil {
		return err
	}

	// Preserve authored stimulus/presentation directives from an existing gold.
	var stdin []byte
	var displays []string
	if _, err := os.Stat(goldPath); err == nil {
		in, _, _, _, ds, err := loadTranscript(goldPath)
		if err != nil {
			return fmt.Errorf("load existing transcript: %w", err)
		}
		stdin = in
		displays = ds
	} else if !os.IsNotExist(err) {
		return err
	}

	stdout, stderr, exitCode, err := runNezhaFile(aiPath, stdin)
	if err != nil {
		return fmt.Errorf("run error: %w", err)
	}
	if err := validateNegativeObservation(aiPath, negativeKind, stderr, exitCode); err != nil {
		return err
	}

	data := encodeTranscript(stdin, stdout, stderr, exitCode, displays)
	if err := os.WriteFile(goldPath, data, 0o644); err != nil {
		return fmt.Errorf("write gold: %w", err)
	}
	fmt.Fprintf(os.Stdout, "wrote gold: %s\n", goldPath)
	return nil
}

func escapeTranscript(s string) string {
	q := strconv.Quote(s)
	return q[1 : len(q)-1]
}

func appendTranscriptRecords(buf *bytes.Buffer, prefix string, data []byte) {
	if len(data) == 0 {
		return
	}
	// One record per logical line keeps golds readable while preserving bytes.
	start := 0
	for i, b := range data {
		if b == '\n' {
			fmt.Fprintf(buf, "%s%s\n", prefix, escapeTranscript(string(data[start:i+1])))
			start = i + 1
		}
	}
	if start < len(data) {
		fmt.Fprintf(buf, "%s%s\n", prefix, escapeTranscript(string(data[start:])))
	}
}

func encodeTranscript(stdin, stdout, stderr []byte, exitCode int, displays []string) []byte {
	var buf bytes.Buffer
	appendTranscriptRecords(&buf, "IN:", stdin)
	appendTranscriptRecords(&buf, "OUT:", stdout)
	appendTranscriptRecords(&buf, "ERR:", stderr)
	if exitCode != 0 {
		fmt.Fprintf(&buf, "EXIT:%d\n", exitCode)
	}
	for _, msg := range displays {
		fmt.Fprintf(&buf, "DISPLAY:%s\n", msg)
	}
	return buf.Bytes()
}

func collectSmokeFiles(args []string) ([]string, error) {
	return collectSmokeFilesWithPolicy(args, false)
}

func collectSmokeFilesWithPolicy(args []string, all bool) ([]string, error) {
	if len(args) == 0 {
		args = []string{"."}
	}
	var paths []string
	var err error
	if all {
		paths, err = targetresolve.ResolveAll("smoke", args, true)
	} else {
		paths, err = targetresolve.Resolve("smoke", args, true)
	}
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.HasSuffix(path, "_smoke.nez") {
			out = append(out, path)
		}
	}
	return out, nil
}

func runPair(aiPath string) error {
	goldPath := strings.TrimSuffix(aiPath, ".nez") + ".gold"
	negativeKind, err := testfixture.NegativeKindOf(aiPath)
	if err != nil {
		return err
	}

	stdin, expOut, expErr, expExit, displays, err := loadTranscript(goldPath)
	if err != nil {
		return fmt.Errorf("load transcript: %w", err)
	}
	if err := validateNegativeObservation(aiPath+" gold", negativeKind, expErr, expExit); err != nil {
		return err
	}

	stdout, stderr, exitCode, err := runNezhaFile(aiPath, stdin)
	if err != nil {
		return fmt.Errorf("run error: %w", err)
	}

	// exit code
	if exitCode != expExit {
		return fmt.Errorf("exit mismatch: expected %d, got %d\nstderr:\n%s",
			expExit, exitCode, string(stderr))
	}

	// stderr
	stdErrTrim := bytes.TrimRight(stderr, "\n")
	expErrTrim := bytes.TrimRight(expErr, "\n")
	if !bytes.Equal(stdErrTrim, expErrTrim) {
		return fmt.Errorf("stderr mismatch\n--- expected ---\n%q\n--- got ---\n%q\n",
			string(expErr), string(stderr))
	}

	// stdout
	stdTrim := bytes.TrimRight(stdout, "\n")
	expTrim := bytes.TrimRight(expOut, "\n")
	if !bytes.Equal(stdTrim, expTrim) {
		return fmt.Errorf("output mismatch\n--- expected ---\n%q\n--- got ---\n%q\n",
			string(expOut), string(stdout))
	}

	// DISPLAY prompts (interactive, for make visual only)
	for _, msg := range displays {
		fmt.Printf("DISPLAY: %s [y/N]: ", strings.TrimSpace(msg))
		var resp string
		fmt.Scanln(&resp)
		if strings.ToLower(strings.TrimSpace(resp)) != "y" {
			return fmt.Errorf("display check failed")
		}
	}

	return nil
}

func validateNegativeObservation(label string, kind testfixture.NegativeKind, stderr []byte, exitCode int) error {
	parseFailure := exitCode != 0 && bytes.HasPrefix(stderr, []byte("parser: "))
	switch kind {
	case testfixture.NegativeParse:
		if !parseFailure {
			return fmt.Errorf("%s: declared @negative parse but observation is not a parser failure", label)
		}
	case testfixture.NegativeNone:
		if parseFailure {
			return fmt.Errorf("%s: parser failure is not declared with # @negative parse", label)
		}
	default:
		return fmt.Errorf("%s: unsupported negative kind %q", label, kind)
	}
	return nil
}

func linesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func unescape(s string) string {
	u, err := strconv.Unquote(`"` + s + `"`)
	if err != nil {
		// fall back to raw if something is weird
		return s
	}
	return u
}

func loadTranscript(path string) (stdin, stdout, stderr []byte, expExit int, displays []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, 0, nil, err
	}
	var inBuf, outBuf, errBuf bytes.Buffer
	expExit = 0

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "IN:"):
			inBuf.WriteString(unescape(strings.TrimPrefix(line, "IN:")))
		case strings.HasPrefix(line, "OUT:"):
			outBuf.WriteString(unescape(strings.TrimPrefix(line, "OUT:")))
		case strings.HasPrefix(line, "ERR:"):
			errBuf.WriteString(unescape(strings.TrimPrefix(line, "ERR:")))
		case strings.HasPrefix(line, "EXIT:"):
			n, e := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "EXIT:")))
			if e != nil {
				return nil, nil, nil, 0, nil, fmt.Errorf("bad EXIT line %q: %w", line, e)
			}
			expExit = n
		case strings.HasPrefix(line, "DISPLAY:"):
			displays = append(displays, strings.TrimPrefix(line, "DISPLAY:"))
		case strings.TrimSpace(line) == "":
			continue
		default:
			return nil, nil, nil, 0, nil, fmt.Errorf("invalid transcript line: %q", line)
		}
	}

	return inBuf.Bytes(), outBuf.Bytes(), errBuf.Bytes(), expExit, displays, nil
}

func runNezhaFile(path string, stdin []byte) (stdout []byte, stderr []byte, exitCode int, err error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, nil, 1, fmt.Errorf("locate executable: %w", err)
	}

	cmd := exec.Command(exe, path)

	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()

	stdout = outBuf.Bytes()
	stderr = errBuf.Bytes()

	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
			return stdout, stderr, exitCode, nil
		}
		return stdout, stderr, 1, runErr
	}

	return stdout, stderr, 0, nil
}

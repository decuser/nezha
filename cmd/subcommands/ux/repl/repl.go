package repl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"nezha/engine/program"
	"nezha/engine/runtime/hal/substrate"
	"nezha/engine/semantics/value"
)

const (
	promptMain = "> "
	promptCont = "  "
)

// Run starts the interactive Nezha REPL.
func Run(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "usage: nezha repl")
		return 2
	}

	programOut := newTrackingWriter(os.Stdout)
	input := newSharedInput(os.Stdin)
	rt := substrate.NewGoRuntime()
	rt.SetInput(input)
	rt.SetOutput(programOut)

	session, err := program.NewSession(rt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "repl:", err)
		return 1
	}

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)

	return run(&terminalReader{input: input}, os.Stdout, os.Stderr, programOut, session, interrupts)
}

var errInputInterrupt = errors.New("input interrupted")

type lineReader interface {
	Prompt() (string, error)
}

// sharedInput is the single terminal input stream shared by REPL acquisition
// and the Nezha runtime. Read never crosses a line boundary, preventing either
// consumer's buffering from stealing input intended for the other. The REPL
// and evaluator use it serially, never concurrently.
type sharedInput struct {
	reader *bufio.Reader
}

func newSharedInput(in io.Reader) *sharedInput {
	return &sharedInput{reader: bufio.NewReader(in)}
}

func (s *sharedInput) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	n := 0
	for n < len(p) {
		b, err := s.reader.ReadByte()
		if err != nil {
			if n > 0 {
				return n, nil
			}
			return 0, err
		}
		p[n] = b
		n++
		if b == '\n' {
			return n, nil
		}
	}
	return n, nil
}

func (s *sharedInput) readLine() (string, error) {
	line, err := s.reader.ReadString('\n')
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	if err == nil {
		return line, nil
	}
	if errors.Is(err, syscall.EINTR) {
		return "", errInputInterrupt
	}
	if errors.Is(err, io.EOF) && line != "" {
		return line, nil
	}
	return "", err
}

type terminalReader struct {
	input *sharedInput
}

func (r *terminalReader) Prompt() (string, error) {
	return r.input.readLine()
}

type trackingWriter struct {
	out              io.Writer
	endedWithNewline bool
}

func newTrackingWriter(out io.Writer) *trackingWriter {
	return &trackingWriter{out: out, endedWithNewline: true}
}

func (w *trackingWriter) Write(p []byte) (int, error) {
	n, err := w.out.Write(p)
	if n > 0 {
		w.endedWithNewline = p[n-1] == '\n'
	}
	return n, err
}

type promptResult struct {
	line string
	err  error
}

func run(reader lineReader, stdout, stderr io.Writer, programOut *trackingWriter, session *program.Session, interrupts <-chan os.Signal) int {
	var buffer string
	prompt := promptMain

replLoop:
	for {
		fmt.Fprint(stdout, prompt)
		lineCh := make(chan promptResult, 1)
		go func() {
			line, err := reader.Prompt()
			lineCh <- promptResult{line: line, err: err}
		}()

		var line string
		for {
			select {
			case <-interrupts:
				// Keep the one pending terminal read. The terminal driver has
				// discarded the interrupted line; the next completed line belongs
				// to this fresh prompt. Do not start a competing stdin reader.
				buffer = ""
				prompt = promptMain
				fmt.Fprintln(stdout)
				fmt.Fprint(stdout, prompt)
			case item := <-lineCh:
				if errors.Is(item.err, errInputInterrupt) {
					buffer = ""
					prompt = promptMain
					drainInterrupts(interrupts)
					fmt.Fprintln(stdout)
					line = ""
					continue replLoop
				}
				if errors.Is(item.err, io.EOF) {
					fmt.Fprintln(stdout)
					return 0
				}
				if item.err != nil {
					fmt.Fprintln(stderr, "repl:", item.err)
					return 1
				}
				line = item.line
				goto acquired
			}
		}
		continue

	acquired:
		if buffer == "" {
			buffer = line
		} else {
			buffer += "\n" + line
		}

		if !isComplete(buffer) {
			prompt = promptCont
			continue
		}
		if isBlank(buffer) {
			buffer = ""
			prompt = promptMain
			continue
		}

		source := buffer
		buffer = ""
		prompt = promptMain
		programOut.endedWithNewline = true

		result, err := evaluate(session, source, interrupts)
		if errors.Is(err, program.ErrInterrupted) {
			if !programOut.endedWithNewline {
				fmt.Fprintln(stdout)
			}
			continue
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
		} else if result != nil && result != value.EMPTY {
			if !programOut.endedWithNewline {
				fmt.Fprintln(stdout)
			}
			fmt.Fprintln(stdout, result.Inspect())
		} else if !programOut.endedWithNewline {
			fmt.Fprintln(stdout)
		}
	}
}

func evaluate(session *program.Session, source string, interrupts <-chan os.Signal) (value.Value, error) {
	// An interrupt received while editing must not poison the next submitted
	// evaluation.
	drainInterrupts(interrupts)

	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-interrupts:
				// The signal may win the race with Eval entering its running
				// state. Keep the request pending until either the evaluation
				// becomes interruptible or it has already completed.
				for !session.Interrupt() {
					select {
					case <-done:
						return
					default:
						runtime.Gosched()
					}
				}
			}
		}
	}()

	return session.Eval("<repl>", source)
}

func drainInterrupts(interrupts <-chan os.Signal) {
	for {
		select {
		case <-interrupts:
			continue
		default:
			return
		}
	}
}

// isComplete recognizes whether another input line is structurally required.
// This is acquisition logic only; accepted submissions are still parsed by the
// ordinary Nezha parser through program.Session.Eval.
func isComplete(input string) bool {
	braces, brackets, parens := 0, 0, 0
	inString := false
	inComment := false
	escaped := false

	for _, r := range input {
		if inComment {
			if r == '\n' {
				inComment = false
			}
			continue
		}
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				inString = false
			}
			continue
		}

		switch r {
		case '#':
			inComment = true
		case '"':
			inString = true
		case '{':
			braces++
		case '}':
			braces--
		case '[':
			brackets++
		case ']':
			brackets--
		case '(':
			parens++
		case ')':
			parens--
		}
	}

	return !inString && braces <= 0 && brackets <= 0 && parens <= 0
}

func isBlank(input string) bool {
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return false
	}
	return true
}

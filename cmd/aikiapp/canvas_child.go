package aikiapp

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"aiki/engine/runtime/hal/substrate"
)

func runCanvasChild(opts Options) {
	// Headless recording remains inside the text-only aiki executable so smoke
	// and conformance paths never initialize the graphics library.
	if canvasRecordRequested() {
		runCanvasRecordChild(opts)
		return
	}

	if opts.CanvasW <= 0 || opts.CanvasH <= 0 {
		fmt.Fprintln(os.Stderr, "canvas: width and height must be positive")
		os.Exit(2)
	}

	helper, err := canvasRendererHelper()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	cmd := exec.Command(helper,
		fmt.Sprintf("-canvasw=%d", opts.CanvasW),
		fmt.Sprintf("-canvash=%d", opts.CanvasH),
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			os.Exit(ee.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func canvasRendererHelper() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	name := "aiki-canvas"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(filepath.Dir(exe), name)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("canvas: renderer helper not found beside executable: %s", path)
	}
	return path, nil
}

func canvasStdinLoop(r io.Reader, cvs *substrate.CanvasResource) {
	dec := substrate.NewCanvasDecoder(r)
	for {
		cmd, err := dec.Read()
		if err != nil {
			break
		}
		handleCanvasWire(cmd, cvs)
	}
	select {
	case <-cvs.Done:
	default:
		close(cvs.Done)
	}
}

func handleCanvasWire(cmd any, cvs *substrate.CanvasResource) {
	switch m := cmd.(type) {
	case substrate.CanvasWireBatch:
		for _, one := range m.Cmds {
			handleCanvasWire(one, cvs)
		}
		return
	case substrate.CanvasWireClose:
		select {
		case <-cvs.Done:
		default:
			close(cvs.Done)
		}
		return
	case substrate.CanvasWireSetBG:
		cvs.BG = m.RGBA
		cvs.Commands <- substrate.CanvasCmd{Op: "clear"}
		return
	case substrate.CanvasWireSetFG:
		cvs.FG = m.RGBA
		return
	case substrate.CanvasWireTurtle:
		cvs.SetTurtle(float64(m.X), float64(m.Y), float64(m.Heading), m.Visible, m.RGBA)
		return
	case substrate.CanvasWireCmd:
		clr := cvs.FG
		if m.HasRGBA {
			clr = m.RGBA
		}
		pen := m.Pen
		if pen <= 0 {
			pen = cvs.PenSize
		}
		args := make([]int, len(m.Args))
		for i, a := range m.Args {
			args[i] = int(a)
		}
		cvs.Commands <- substrate.CanvasCmd{Op: m.Op, Args: args, Color: clr, PenSize: pen}
		return
	default:
		return
	}
}

package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"nezha/engine/program"
	"nezha/engine/runtime/hal/substrate"
)

const (
	maxSourceBytes   = 1 << 20
	maxDocumentBytes = 8 << 20
)

//go:embed all:web
var webAssets embed.FS

type App struct {
	ctx context.Context

	runMu          sync.Mutex
	runSession     *program.Session
	outputSink     func(string)
	completionSink func(runResponse)
}

type runResponse struct {
	Error   string `json:"error,omitempty"`
	Stopped bool   `json:"stopped,omitempty"`
}

type fileResponse struct {
	Path    string `json:"path,omitempty"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

func main() {
	assets, err := fs.Sub(webAssets, "web")
	if err != nil {
		panic(err)
	}

	app := &App{}
	appMenu := buildMenu(app)

	if err := wails.Run(&options.App{
		Title:            "Nezha Blockly",
		Width:            1200,
		Height:           800,
		WindowStartState: options.Maximised,
		AssetServer:      &assetserver.Options{Assets: assets},
		Menu:             appMenu,
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
	}); err != nil {
		panic(err)
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	wruntime.EventsOn(ctx, "run:stop", func(_ ...interface{}) {
		a.Stop()
	})
}

func buildMenu(app *App) *menu.Menu {
	appMenu := menu.NewMenu()
	if goruntime.GOOS == "darwin" {
		appMenu.Append(menu.AppMenu())
	}

	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("New", keys.CmdOrCtrl("n"), func(_ *menu.CallbackData) { app.emit("file:new") })
	fileMenu.AddText("Open...", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) { app.emit("file:open") })
	fileMenu.AddSeparator()
	fileMenu.AddText("Save", keys.CmdOrCtrl("s"), func(_ *menu.CallbackData) { app.emit("file:save") })
	fileMenu.AddText("Save As...", nil, func(_ *menu.CallbackData) { app.emit("file:save-as") })
	fileMenu.AddSeparator()
	fileMenu.AddText("Export Nezha Source...", nil, func(_ *menu.CallbackData) { app.emit("file:export") })
	fileMenu.AddSeparator()
	fileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) { app.emit("file:quit") })

	editMenu := appMenu.AddSubmenu("Edit")
	editMenu.AddText("Undo", keys.CmdOrCtrl("z"), func(_ *menu.CallbackData) { app.emit("edit:undo") })
	editMenu.AddText("Redo", nil, func(_ *menu.CallbackData) { app.emit("edit:redo") })

	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.AddText("Editor", nil, func(_ *menu.CallbackData) { app.emit("view:editor") })
	viewMenu.AddText("Samples", nil, func(_ *menu.CallbackData) { app.emit("view:samples") })
	viewMenu.AddText("Help", nil, func(_ *menu.CallbackData) { app.emit("view:help") })

	if goruntime.GOOS == "darwin" {
		appMenu.Append(menu.EditMenu())
	}
	return appMenu
}

func (a *App) emit(name string) {
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, name)
	}
}

// Start launches generated Nezha source in a fresh canonical program session
// and returns immediately. The active session is retained only while the run
// is executing so Stop can interrupt it. Runtime output and completion are
// delivered independently, which keeps the Wails bridge free to service Stop.
func (a *App) Start(source, input string) runResponse {
	if len(source) > maxSourceBytes {
		return runResponse{Error: fmt.Sprintf("source is too large: %d bytes", len(source))}
	}

	rt := substrate.NewGoRuntime()
	rt.SetInput(strings.NewReader(input))

	stream := newOutputStream(a.emitRunOutput)
	rt.SetOutput(stream)

	session, err := program.NewSession(rt)
	if err != nil {
		return runResponse{Error: err.Error()}
	}
	if !a.beginRun(session) {
		return runResponse{Error: "a program is already running"}
	}

	go a.executeRun(session, source, stream)
	return runResponse{}
}

func (a *App) executeRun(session *program.Session, source string, stream *outputStream) {
	result := runResponse{}
	_, err := session.Eval("blockly.nez", source)
	if errors.Is(err, program.ErrInterrupted) {
		result.Stopped = true
	} else if err != nil {
		result.Error = err.Error()
	}

	stream.Close()
	a.endRun(session)
	a.emitRunFinished(result)
}

func (a *App) Stop() bool {
	a.runMu.Lock()
	session := a.runSession
	a.runMu.Unlock()
	if session == nil {
		return false
	}
	return session.Interrupt()
}

func (a *App) beginRun(session *program.Session) bool {
	a.runMu.Lock()
	defer a.runMu.Unlock()
	if a.runSession != nil {
		return false
	}
	a.runSession = session
	return true
}

func (a *App) endRun(session *program.Session) {
	a.runMu.Lock()
	if a.runSession == session {
		a.runSession = nil
	}
	a.runMu.Unlock()
}

func (a *App) emitRunOutput(chunk string) {
	if chunk == "" {
		return
	}
	if a.outputSink != nil {
		a.outputSink(chunk)
		return
	}
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, "run:output", chunk)
	}
}

func (a *App) emitRunFinished(result runResponse) {
	if a.completionSink != nil {
		a.completionSink(result)
		return
	}
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, "run:finished", result)
	}
}

const (
	outputFlushInterval = 50 * time.Millisecond
	outputBufferBytes   = 8 * 1024
)

type outputStream struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    strings.Builder
	emit   func(string)
	done   chan struct{}
	closed bool
}

func newOutputStream(emit func(string)) *outputStream {
	s := &outputStream{
		emit: emit,
		done: make(chan struct{}),
	}
	s.cond = sync.NewCond(&s.mu)
	go s.run()
	return s
}

func (s *outputStream) Write(p []byte) (int, error) {
	written := 0
	for written < len(p) {
		s.mu.Lock()
		for !s.closed && s.buf.Len() >= outputBufferBytes {
			s.cond.Wait()
		}
		if s.closed {
			s.mu.Unlock()
			return len(p), nil
		}

		room := outputBufferBytes - s.buf.Len()
		if room <= 0 {
			s.mu.Unlock()
			continue
		}
		n := len(p) - written
		if n > room {
			n = room
		}
		_, _ = s.buf.Write(p[written : written+n])
		written += n
		s.mu.Unlock()
	}
	return len(p), nil
}

func (s *outputStream) run() {
	ticker := time.NewTicker(outputFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.flush()
		case <-s.done:
			s.flush()
			return
		}
	}
}

func (s *outputStream) flush() {
	s.mu.Lock()
	if s.buf.Len() == 0 {
		s.mu.Unlock()
		return
	}
	chunk := s.buf.String()
	s.buf.Reset()
	s.cond.Broadcast()
	s.mu.Unlock()

	if s.emit != nil {
		s.emit(chunk)
	}
}

func (s *outputStream) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.cond.Broadcast()
	close(s.done)
	s.mu.Unlock()
}

type writerFunc func([]byte) (int, error)

func (fn writerFunc) Write(p []byte) (int, error) { return fn(p) }

var _ io.Writer = writerFunc(nil)

func (a *App) OpenDocument() fileResponse {
	path, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Open Nezha Blockly Program",
		Filters: []wruntime.FileFilter{{
			DisplayName: "Nezha Blockly programs (*.nezha-blocks)",
			Pattern:     "*.nezha-blocks",
		}},
	})
	if err != nil {
		return fileResponse{Error: err.Error()}
	}
	if path == "" {
		return fileResponse{}
	}
	content, err := readTextFile(path)
	if err != nil {
		return fileResponse{Error: err.Error()}
	}
	return fileResponse{Path: path, Content: content}
}

func (a *App) SaveDocument(path, content string, saveAs bool) fileResponse {
	if len(content) > maxDocumentBytes {
		return fileResponse{Error: fmt.Sprintf("document is too large: %d bytes", len(content))}
	}
	if path == "" || saveAs {
		selected, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
			Title:           "Save Nezha Blockly Program",
			DefaultFilename: "untitled.nezha-blocks",
			Filters: []wruntime.FileFilter{{
				DisplayName: "Nezha Blockly programs (*.nezha-blocks)",
				Pattern:     "*.nezha-blocks",
			}},
		})
		if err != nil {
			return fileResponse{Error: err.Error()}
		}
		if selected == "" {
			return fileResponse{}
		}
		path = selected
	}
	if err := writeTextFile(path, content); err != nil {
		return fileResponse{Error: err.Error()}
	}
	return fileResponse{Path: path}
}

func (a *App) ExportSource(source string) fileResponse {
	if len(source) > maxSourceBytes {
		return fileResponse{Error: fmt.Sprintf("source is too large: %d bytes", len(source))}
	}
	path, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		Title:           "Export Nezha Source",
		DefaultFilename: "program.nezha",
		Filters: []wruntime.FileFilter{{
			DisplayName: "Nezha source (*.nezha)",
			Pattern:     "*.nezha",
		}},
	})
	if err != nil {
		return fileResponse{Error: err.Error()}
	}
	if path == "" {
		return fileResponse{}
	}
	if err := writeTextFile(path, source); err != nil {
		return fileResponse{Error: err.Error()}
	}
	return fileResponse{Path: path}
}

func (a *App) ConfirmDiscard() bool {
	result, err := wruntime.MessageDialog(a.ctx, wruntime.MessageDialogOptions{
		Type:          wruntime.QuestionDialog,
		Title:         "Unsaved Changes",
		Message:       "Discard unsaved changes?",
		DefaultButton: "No",
	})
	return err == nil && result == "Yes"
}

func (a *App) Quit() {
	wruntime.Quit(a.ctx)
}

func readTextFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.Size() > maxDocumentBytes {
		return "", fmt.Errorf("document is too large: %d bytes", info.Size())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeTextFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

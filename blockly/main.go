package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	goruntime "runtime"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"nezha/engine/runtime/hal/substrate"
	"nezha/runner"
)

const (
	maxSourceBytes   = 1 << 20
	maxDocumentBytes = 8 << 20
)

//go:embed all:web
var webAssets embed.FS

type App struct {
	ctx context.Context
}

type runResponse struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
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

// Run executes generated Nezha source with the supplied text as the runtime
// input endpoint. Blockly remains only a source producer; the ordinary Nezha
// runner and runtime remain authoritative for execution.
func (a *App) Run(source, input string) runResponse {
	if len(source) > maxSourceBytes {
		return runResponse{Error: fmt.Sprintf("source is too large: %d bytes", len(source))}
	}
	output, err := runSource(source, input)
	response := runResponse{Output: output}
	if err != nil {
		response.Error = err.Error()
	}
	return response
}

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

func runSource(source, input string) (string, error) {
	var output bytes.Buffer
	rt := substrate.NewGoRuntime()
	rt.SetInput(strings.NewReader(input))
	rt.SetOutput(&output)
	err := runner.RunSourceWithRuntime("blockly.nez", source, rt)
	return output.String(), err
}

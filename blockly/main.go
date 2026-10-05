package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"

	"nezha/engine/runtime/hal/substrate"
	"nezha/runner"
)

const maxSourceBytes = 1 << 20

type runResponse struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

func main() {
	addr := ":8080"
	if v := os.Getenv("NEZHA_BLOCKLY_ADDR"); v != "" {
		addr = v
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /run", runHandler)
	mux.Handle("/", http.FileServer(http.Dir("blockly/web")))

	log.Printf("Nezha Blockly: http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func runSource(source string) (string, error) {
	var output bytes.Buffer
	rt := substrate.NewGoRuntime()
	rt.SetOutput(&output)
	err := runner.RunSourceWithRuntime("blockly.nez", source, rt)
	return output.String(), err
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	source, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxSourceBytes))
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "source is too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "could not read source", http.StatusBadRequest)
		return
	}

	output, runErr := runSource(string(source))
	response := runResponse{Output: output}
	if runErr != nil {
		response.Error = runErr.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("encoding /run response: %v", err)
	}
}

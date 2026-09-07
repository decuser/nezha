package substrate

import (
	"fmt"
	"sort"

	"aiki/engine/runtime/hal"
)

// goHostProvenance records how this substrate realizes each architectural HAL
// operation. Architectural metadata lives in engine/runtime/hal; this file is
// intentionally Go-substrate-specific.
var goHostProvenance = map[string]string{
	"_print":            "go:fmt.Fprint/io.Writer",
	"_read":             "go:bufio.Reader.ReadString/io.Reader",
	"_file_open":        "go:os.Open/os.Create/os.OpenFile",
	"_file_read_text":   "go:io.ReadAll(*os.File)",
	"_file_read_bytes":  "go:io.ReadAll(*os.File)",
	"_file_read_line":   "go:bufio.Reader.ReadString(*os.File)",
	"_file_write_text":  "go:os.File.WriteString",
	"_file_write_bytes": "go:os.File.Write",
	"_file_close":       "go:os.File.Close",
	"_file_exists":      "go:os.Stat",
	"_file_delete":      "go:os.Remove",
	"_file_list":        "go:os.ReadDir",
	"_file_entries":     "go:os.ReadDir/DirEntry.IsDir",
	"_file_read_at":     "go:os.File.ReadAt",
	"_file_write_at":    "go:os.File.WriteAt",
	"_file_stat":        "go:os.Stat",
	"_file_rename":      "go:os.Rename",
	"_file_mkdir":       "go:os.Mkdir",
	"_file_mkdir_all":   "go:os.MkdirAll",
	"_file_remove_all":  "go:os.RemoveAll",
	"_file_temp":        "go:os.CreateTemp",
	"_file_temp_dir":    "go:os.MkdirTemp",
	"_file_copy":        "go:os.Open/os.Create/io.Copy",
	"_file_size":        "go:os.Stat",
	"_file_walk":        "go:path/filepath.WalkDir",
	"_file_symlink":     "go:os.Symlink",
	"_file_read_link":   "go:os.Readlink",
	"_file_lock":        "go:github.com/gofrs/flock.Lock",
	"_file_try_lock":    "go:github.com/gofrs/flock.TryLock",
	"_file_unlock":      "go:github.com/gofrs/flock.Unlock",
	"_file_permissions": "go:os.FileMode.Perm",
	"_file_chmod":       "go:os.Chmod",
}

func goHostOperation(primitive string) hal.HostOperation {
	op, ok := hal.OperationDefinition(primitive)
	if !ok {
		panic(fmt.Sprintf("missing HAL operation definition: %s", primitive))
	}
	provenance, ok := goHostProvenance[primitive]
	if !ok || provenance == "" {
		panic(fmt.Sprintf("missing Go substrate provenance: %s", primitive))
	}
	op.SubstrateProvenance = provenance
	return op
}

// HostOperations returns the canonical host-operation descriptors bound by this
// runtime, sorted by HAL identity. The returned slice is independent of runtime
// execution state and is intended for tooling, validation, and observation.
func (g *GoRuntime) HostOperations() []hal.HostOperation {
	g.mu.RLock()
	out := make([]hal.HostOperation, 0, len(g.hostBindings))
	for _, op := range g.hostBindings {
		copyOp := op
		copyOp.Context = append([]string(nil), op.Context...)
		copyOp.AikiBindings = append([]hal.AikiBinding(nil), op.AikiBindings...)
		out = append(out, copyOp)
	}
	g.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool { return out[i].Identity < out[j].Identity })
	return out
}

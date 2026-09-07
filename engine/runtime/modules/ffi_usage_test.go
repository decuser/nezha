package modules

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFFIUsageFollowsNativeAliasesAndTransitiveFFI(t *testing.T) {
	root := t.TempDir()
	lib := filepath.Join(root, "lib")
	mustWriteUsageFile(t, filepath.Join(lib, "thing", "native.nez"), `package "thing/native"
export()
`)
	mustWriteUsageFile(t, filepath.Join(lib, "thing", "native.help"), "@summary \"test thing native module\"\n")
	mustWriteUsageFile(t, filepath.Join(lib, "thing", "native.doc"), ``)
	mustWriteUsageFile(t, filepath.Join(lib, "fast", "ffi.nez"), `package "fast/ffi"
export()
`)
	mustWriteUsageFile(t, filepath.Join(lib, "fast", "ffi.help"), "@summary \"test fast ffi module\"\n")
	mustWriteUsageFile(t, filepath.Join(lib, "fast", "ffi.doc"), ``)
	mustWriteUsageFile(t, filepath.Join(lib, "mixed", "mixed.nez"), `package "mixed"
let fast = import("fast/ffi")
export()
`)
	mustWriteUsageFile(t, filepath.Join(lib, "mixed", "mixed.help"), "@summary \"test mixed module\"\n")
	mustWriteUsageFile(t, filepath.Join(lib, "mixed", "mixed.doc"), ``)
	entry := filepath.Join(root, "main.nez")
	mustWriteUsageFile(t, entry, `let native = import("thing")
let mixed = import("mixed")
`)

	got, err := FFIUsage(testGrammar(t), entry, []string{lib})
	if err != nil {
		t.Fatalf("FFIUsage: %v", err)
	}
	if want := []string{"fast/ffi"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("FFIUsage = %v, want %v", got, want)
	}
}

func TestFFIUsageFollowsRelativeImports(t *testing.T) {
	root := t.TempDir()
	lib := filepath.Join(root, "lib")
	mustWriteUsageFile(t, filepath.Join(lib, "fast", "ffi.nez"), `package "fast/ffi"
export()
`)
	mustWriteUsageFile(t, filepath.Join(lib, "fast", "ffi.help"), "@summary \"test fast ffi module\"\n")
	mustWriteUsageFile(t, filepath.Join(lib, "fast", "ffi.doc"), ``)
	entry := filepath.Join(root, "main.nez")
	mustWriteUsageFile(t, entry, `let local = import("./local")
`)
	mustWriteUsageFile(t, filepath.Join(root, "local.nez"), `let fast = import("fast/ffi")
`)

	got, err := FFIUsage(testGrammar(t), entry, []string{lib})
	if err != nil {
		t.Fatalf("FFIUsage: %v", err)
	}
	if want := []string{"fast/ffi"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("FFIUsage = %v, want %v", got, want)
	}
}

func mustWriteUsageFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

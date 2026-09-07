package substrate

import (
	"os"
	"path/filepath"
	"testing"

	"nezha/engine/semantics/value"
)

func TestM5FileMetadataDirectoryAndCopyAffordances(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(src, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	stat := halFileStat([]value.Value{&value.String{Val: src}}, nil)
	st, ok := stat.(*value.List)
	if !ok || st.Shape != "stat" || len(st.Elements) != 3 {
		t.Fatalf("stat = %#v, want [@stat, size, modified, is_dir]", stat)
	}
	if got := st.Elements[0].Inspect(); got != "4" {
		t.Fatalf("stat size = %s, want 4", got)
	}
	if st.Elements[2] != value.FALSE {
		t.Fatalf("stat is_dir = %s, want false", st.Elements[2].Inspect())
	}

	size := halFileSize([]value.Value{&value.String{Val: src}}, nil)
	if got := size.Inspect(); got != "4" {
		t.Fatalf("size = %s, want 4", got)
	}

	dst := filepath.Join(dir, "copy.txt")
	if got := halFileCopy([]value.Value{&value.String{Val: src}, &value.String{Val: dst}}, nil); got != value.TRUE {
		t.Fatalf("copy = %s", got.Inspect())
	}
	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "data" {
		t.Fatalf("copied data = %q, %v", data, err)
	}

	renamed := filepath.Join(dir, "renamed.txt")
	if got := halFileRename([]value.Value{&value.String{Val: dst}, &value.String{Val: renamed}}, nil); got != value.TRUE {
		t.Fatalf("rename = %s", got.Inspect())
	}

	nested := filepath.Join(dir, "one", "two")
	if got := halFileMkdirAll([]value.Value{&value.String{Val: nested}}, nil); got != value.TRUE {
		t.Fatalf("mkdir_all = %s", got.Inspect())
	}
	single := filepath.Join(dir, "single")
	if got := halFileMkdir([]value.Value{&value.String{Val: single}}, nil); got != value.TRUE {
		t.Fatalf("mkdir = %s", got.Inspect())
	}
	if got := halFileRemoveAll([]value.Value{&value.String{Val: filepath.Join(dir, "one")}}, nil); got != value.TRUE {
		t.Fatalf("remove_all = %s", got.Inspect())
	}
}

func TestM5TempAffordancesCreateRealPaths(t *testing.T) {
	fileV := halFileTemp(nil, nil)
	filePath, ok := fileV.(*value.String)
	if !ok {
		t.Fatalf("temp = %T, want string", fileV)
	}
	defer os.Remove(filePath.Val)
	if _, err := os.Stat(filePath.Val); err != nil {
		t.Fatalf("temp path does not exist: %v", err)
	}

	dirV := halFileTempDir(nil, nil)
	dirPath, ok := dirV.(*value.String)
	if !ok {
		t.Fatalf("temp_dir = %T, want string", dirV)
	}
	defer os.RemoveAll(dirPath.Val)
	info, err := os.Stat(dirPath.Val)
	if err != nil || !info.IsDir() {
		t.Fatalf("temp_dir is not a directory: %v", err)
	}
}

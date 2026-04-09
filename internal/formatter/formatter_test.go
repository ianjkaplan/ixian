package formatter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iankaplan/ixian/internal/binder"
	"github.com/iankaplan/ixian/internal/emitter"
	"github.com/iankaplan/ixian/internal/parser"
	"github.com/iankaplan/ixian/internal/planner"
)

func testdataPath(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func TestFormatPetstore(t *testing.T) {
	t.Parallel()
	spec, err := parser.Parse(testdataPath("petstore.yaml"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bound, err := binder.Bind(spec)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	plan := planner.Plan(bound)

	files, err := emitter.Emit(plan)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	formatted, err := Format(files)
	if err != nil {
		t.Fatalf("format: %v", err)
	}

	if len(formatted) != len(files) {
		t.Errorf("formatted file count = %d, want %d", len(formatted), len(files))
	}

	for _, f := range formatted {
		if len(f.Content) == 0 {
			t.Errorf("formatted file %s is empty", f.Name)
		}
	}
}

func TestWriteFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	files := []emitter.File{
		{Name: "types/types.go", Content: []byte("package types\n")},
		{Name: "main.go", Content: []byte("package main\n")},
	}

	if err := Write(dir, files); err != nil {
		t.Fatalf("write: %v", err)
	}

	for _, f := range files {
		path := filepath.Join(dir, f.Name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("reading %s: %v", f.Name, err)
			continue
		}
		if string(data) != string(f.Content) {
			t.Errorf("file %s content mismatch", f.Name)
		}
	}
}

func TestFormatInvalidGo(t *testing.T) {
	t.Parallel()
	files := []emitter.File{
		{Name: "bad.go", Content: []byte("this is not valid go code {{{")},
	}
	_, err := Format(files)
	if err == nil {
		t.Error("expected error formatting invalid Go")
	}
}

package e2e

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/iankaplan/ixian/internal/binder"
	"github.com/iankaplan/ixian/internal/emitter"
	"github.com/iankaplan/ixian/internal/formatter"
	"github.com/iankaplan/ixian/internal/parser"
	"github.com/iankaplan/ixian/internal/planner"
)

var update = flag.Bool("update", false, "update golden files")

func testdataPath(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func goldenDir(name string) string {
	return filepath.Join("..", "..", "testdata", "golden", name)
}

// runPipeline runs the full codegen pipeline on a spec file and returns
// the formatted output files.
func runPipeline(t *testing.T, specPath string) []emitter.File {
	t.Helper()

	spec, err := parser.Parse(specPath)
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
	formatted, err := formatter.Format(files)
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	return formatted
}

func TestGoldenPetstore(t *testing.T) {
	files := runPipeline(t, testdataPath("petstore.yaml"))
	dir := goldenDir("petstore")

	if *update {
		updateGoldenFiles(t, dir, files)
		return
	}

	// Load expected golden files from disk.
	expected := loadGoldenFiles(t, dir)

	// Build map of actual output.
	actual := make(map[string]string, len(files))
	for _, f := range files {
		actual[f.Name] = string(f.Content)
	}

	// Check for missing or extra files.
	allKeys := make(map[string]bool)
	for k := range expected {
		allKeys[k] = true
	}
	for k := range actual {
		allKeys[k] = true
	}

	sorted := make([]string, 0, len(allKeys))
	for k := range allKeys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	for _, name := range sorted {
		exp, hasExp := expected[name]
		act, hasAct := actual[name]

		if !hasExp {
			t.Errorf("unexpected file in output: %s (run with -update to accept)", name)
			continue
		}
		if !hasAct {
			t.Errorf("missing file in output: %s (expected by golden)", name)
			continue
		}
		if act != exp {
			t.Errorf("file %s differs from golden.\n%s", name, diff(exp, act))
		}
	}
}

// updateGoldenFiles writes the current pipeline output as the new golden files.
func updateGoldenFiles(t *testing.T, dir string, files []emitter.File) {
	t.Helper()

	// Remove old golden dir to catch deleted files.
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("remove old golden dir: %v", err)
	}

	for _, f := range files {
		path := filepath.Join(dir, f.Name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, f.Content, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	t.Logf("updated golden files in %s", dir)
}

// loadGoldenFiles reads all files under dir into a map keyed by relative path.
func loadGoldenFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	result := make(map[string]string)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("walk golden dir %s: %v", dir, err)
	}
	return result
}

// diff returns a simple line-by-line diff showing first divergence.
func diff(expected, actual string) string {
	expLines := strings.Split(expected, "\n")
	actLines := strings.Split(actual, "\n")

	var b strings.Builder
	maxLines := len(expLines)
	if len(actLines) > maxLines {
		maxLines = len(actLines)
	}

	shown := 0
	for i := 0; i < maxLines && shown < 10; i++ {
		exp, act := "", ""
		if i < len(expLines) {
			exp = expLines[i]
		}
		if i < len(actLines) {
			act = actLines[i]
		}
		if exp != act {
			fmt.Fprintf(&b, "  line %d:\n", i+1)
			fmt.Fprintf(&b, "    want: %s\n", exp)
			fmt.Fprintf(&b, "    got:  %s\n", act)
			shown++
		}
	}

	if shown == 0 {
		b.WriteString("  (no line differences found — possible trailing whitespace)")
	}

	return b.String()
}

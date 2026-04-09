// Package formatter runs goimports on generated Go source files and writes them to disk.
package formatter

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"

	"github.com/iankaplan/ixian/internal/emitter"
)

// Format formats the source of each file using go/format (gofmt).
// Returns the files with formatted content.
func Format(files []emitter.File) ([]emitter.File, error) {
	var result []emitter.File
	for _, f := range files {
		formatted, err := format.Source(f.Content)
		if err != nil {
			return nil, fmt.Errorf("formatting %s: %w", f.Name, err)
		}
		result = append(result, emitter.File{Name: f.Name, Content: formatted})
	}
	return result, nil
}

// Write writes the formatted files to the output directory.
func Write(dir string, files []emitter.File) error {
	for _, f := range files {
		path := filepath.Join(dir, f.Name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", f.Name, err)
		}
		if err := os.WriteFile(path, f.Content, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", f.Name, err)
		}
	}
	return nil
}

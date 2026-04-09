// Package parser reads an OpenAPI 3.x spec file and produces a raw, unresolved AST.
package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/iankaplan/ixian/internal/ast"
	"gopkg.in/yaml.v3"
)

// Parse reads an OpenAPI spec from the given file path and returns the raw AST.
// Supports both YAML and JSON input formats based on file extension.
func Parse(path string) (*ast.Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading spec file: %w", err)
	}
	return ParseBytes(data, filepath.Ext(path))
}

// ParseBytes parses raw bytes into an AST. The ext parameter (".yaml", ".yml", ".json")
// determines the format. If empty, YAML is assumed.
func ParseBytes(data []byte, ext string) (*ast.Spec, error) {
	var spec ast.Spec

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("parsing JSON spec: %w", err)
		}
	default:
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("parsing YAML spec: %w", err)
		}
	}

	if spec.OpenAPI == "" {
		return nil, fmt.Errorf("missing required field: openapi")
	}
	if spec.Info.Title == "" {
		return nil, fmt.Errorf("missing required field: info.title")
	}

	return &spec, nil
}

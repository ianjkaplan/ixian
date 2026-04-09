// Package checker validates the bound AST and produces diagnostics.
package checker

import (
	"fmt"
	"strings"

	"github.com/iankaplan/ixian/internal/ast"
	"github.com/iankaplan/ixian/internal/binder"
)

type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

type Diagnostic struct {
	Severity Severity
	Message  string
}

func (d Diagnostic) String() string {
	prefix := "error"
	if d.Severity == SeverityWarning {
		prefix = "warning"
	}
	return fmt.Sprintf("%s: %s", prefix, d.Message)
}

type Result struct {
	Diagnostics []Diagnostic
}

func (r *Result) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

func (r *Result) Errors() []Diagnostic {
	var errs []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			errs = append(errs, d)
		}
	}
	return errs
}

func (r *Result) Warnings() []Diagnostic {
	var warns []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityWarning {
			warns = append(warns, d)
		}
	}
	return warns
}

// Check validates the bound spec and returns diagnostics.
func Check(bound *binder.BoundSpec) *Result {
	c := &checker{bound: bound, result: &Result{}}
	c.checkOperationIDs()
	c.checkPathParams()
	c.checkSchemas()
	return c.result
}

type checker struct {
	bound  *binder.BoundSpec
	result *Result
}

func (c *checker) errorf(msg string, args ...any) {
	c.result.Diagnostics = append(c.result.Diagnostics, Diagnostic{
		Severity: SeverityError,
		Message:  fmt.Sprintf(msg, args...),
	})
}

func (c *checker) warnf(msg string, args ...any) {
	c.result.Diagnostics = append(c.result.Diagnostics, Diagnostic{
		Severity: SeverityWarning,
		Message:  fmt.Sprintf(msg, args...),
	})
}

// checkOperationIDs ensures all operation IDs are unique and present.
func (c *checker) checkOperationIDs() {
	seen := make(map[string]string) // operationId → path+method
	for path, pi := range c.bound.Spec.Paths {
		for method, op := range map[string]*ast.Operation{
			"GET": pi.Get, "POST": pi.Post, "PUT": pi.Put, "DELETE": pi.Delete, "PATCH": pi.Patch,
		} {
			if op == nil {
				continue
			}
			if op.OperationID == "" {
				c.errorf("%s %s: missing operationId", method, path)
				continue
			}
			loc := fmt.Sprintf("%s %s", method, path)
			if prev, ok := seen[op.OperationID]; ok {
				c.errorf("duplicate operationId %q: %s and %s", op.OperationID, prev, loc)
			}
			seen[op.OperationID] = loc
		}
	}
}

// checkPathParams ensures path parameters in the URL template match declared parameters.
func (c *checker) checkPathParams() {
	for path, pi := range c.bound.Spec.Paths {
		templateParams := extractPathParams(path)
		for method, op := range map[string]*ast.Operation{
			"GET": pi.Get, "POST": pi.Post, "PUT": pi.Put, "DELETE": pi.Delete, "PATCH": pi.Patch,
		} {
			if op == nil {
				continue
			}
			declared := make(map[string]bool)
			for _, p := range op.Parameters {
				if p.In == "path" {
					declared[p.Name] = true
				}
			}
			for _, tp := range templateParams {
				if !declared[tp] {
					c.errorf("%s %s: path parameter {%s} not declared in operation %s", method, path, tp, op.OperationID)
				}
			}
			for name := range declared {
				found := false
				for _, tp := range templateParams {
					if tp == name {
						found = true
						break
					}
				}
				if !found {
					c.errorf("%s %s: declared path parameter %q not in path template", method, path, name)
				}
			}
		}
	}
}

// checkSchemas validates schema definitions.
func (c *checker) checkSchemas() {
	for name, schema := range c.bound.SymbolTable {
		c.checkSchema(name, schema)
	}
}

func (c *checker) checkSchema(name string, s *ast.Schema) {
	if s == nil {
		return
	}

	// Check required fields reference existing properties
	if s.Type == "object" && len(s.Required) > 0 {
		for _, req := range s.Required {
			if _, ok := s.Properties[req]; !ok {
				c.errorf("schema %s: required field %q not in properties", name, req)
			}
		}
	}
}

func extractPathParams(path string) []string {
	var params []string
	for {
		start := strings.Index(path, "{")
		if start == -1 {
			break
		}
		end := strings.Index(path[start:], "}")
		if end == -1 {
			break
		}
		params = append(params, path[start+1:start+end])
		path = path[start+end+1:]
	}
	return params
}

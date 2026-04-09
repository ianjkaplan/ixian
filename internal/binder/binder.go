// Package binder resolves all $ref pointers in the raw AST, builds a symbol table
// and dependency graph. After binding, every reference is a direct pointer.
package binder

import (
	"fmt"
	"strings"

	"github.com/iankaplan/ixian/internal/ast"
)

// BoundSpec is the output of the binder: the original spec with all refs resolved,
// plus a symbol table and dependency graph.
type BoundSpec struct {
	Spec         *ast.Spec
	SymbolTable  map[string]*ast.Schema // schema name → resolved definition
	Dependencies map[string][]string    // schema name → schemas it references
}

// Bind resolves all $ref pointers in the spec and returns the bound result.
func Bind(spec *ast.Spec) (*BoundSpec, error) {
	b := &BoundSpec{
		Spec:         spec,
		SymbolTable:  make(map[string]*ast.Schema),
		Dependencies: make(map[string][]string),
	}

	// Build symbol table from components/schemas
	if spec.Components != nil {
		for name, schema := range spec.Components.Schemas {
			b.SymbolTable[name] = schema
		}
	}

	// Build dependency graph
	if spec.Components != nil {
		for name, schema := range spec.Components.Schemas {
			deps := collectRefs(schema)
			if len(deps) > 0 {
				b.Dependencies[name] = deps
			}
		}
	}

	// Resolve all $ref pointers throughout the spec
	for _, pathItem := range spec.Paths {
		if err := b.resolvePathItem(pathItem); err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (b *BoundSpec) resolvePathItem(pi *ast.PathItem) error {
	for _, op := range []*ast.Operation{pi.Get, pi.Post, pi.Put, pi.Delete, pi.Patch} {
		if op == nil {
			continue
		}
		if err := b.resolveOperation(op); err != nil {
			return err
		}
	}
	return nil
}

func (b *BoundSpec) resolveOperation(op *ast.Operation) error {
	for i := range op.Parameters {
		if op.Parameters[i].Schema != nil {
			resolved, err := b.resolveSchema(op.Parameters[i].Schema)
			if err != nil {
				return fmt.Errorf("operation %s param %s: %w", op.OperationID, op.Parameters[i].Name, err)
			}
			op.Parameters[i].Schema = resolved
		}
	}

	if op.RequestBody != nil {
		for mediaType, mt := range op.RequestBody.Content {
			if mt.Schema != nil {
				resolved, err := b.resolveSchema(mt.Schema)
				if err != nil {
					return fmt.Errorf("operation %s requestBody %s: %w", op.OperationID, mediaType, err)
				}
				mt.Schema = resolved
				op.RequestBody.Content[mediaType] = mt
			}
		}
	}

	for code, resp := range op.Responses {
		for mediaType, mt := range resp.Content {
			if mt.Schema != nil {
				resolved, err := b.resolveSchema(mt.Schema)
				if err != nil {
					return fmt.Errorf("operation %s response %s %s: %w", op.OperationID, code, mediaType, err)
				}
				mt.Schema = resolved
				resp.Content[mediaType] = mt
			}
		}
	}

	return nil
}

func (b *BoundSpec) resolveSchema(s *ast.Schema) (*ast.Schema, error) {
	if s == nil {
		return nil, nil
	}

	if s.IsRef() {
		name := refToName(s.Ref)
		resolved, ok := b.SymbolTable[name]
		if !ok {
			return nil, fmt.Errorf("unresolved $ref: %s", s.Ref)
		}
		return resolved, nil
	}

	// Resolve nested schemas
	if s.Items != nil {
		resolved, err := b.resolveSchema(s.Items)
		if err != nil {
			return nil, err
		}
		s.Items = resolved
	}

	for propName, prop := range s.Properties {
		resolved, err := b.resolveSchema(prop)
		if err != nil {
			return nil, fmt.Errorf("property %s: %w", propName, err)
		}
		s.Properties[propName] = resolved
	}

	for i, sub := range s.AllOf {
		resolved, err := b.resolveSchema(sub)
		if err != nil {
			return nil, err
		}
		s.AllOf[i] = resolved
	}
	for i, sub := range s.OneOf {
		resolved, err := b.resolveSchema(sub)
		if err != nil {
			return nil, err
		}
		s.OneOf[i] = resolved
	}
	for i, sub := range s.AnyOf {
		resolved, err := b.resolveSchema(sub)
		if err != nil {
			return nil, err
		}
		s.AnyOf[i] = resolved
	}

	if s.AdditionalProperties != nil {
		resolved, err := b.resolveSchema(s.AdditionalProperties)
		if err != nil {
			return nil, err
		}
		s.AdditionalProperties = resolved
	}

	return s, nil
}

// refToName extracts the schema name from a $ref string like "#/components/schemas/Pet".
func refToName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

// collectRefs finds all $ref targets in a schema tree.
func collectRefs(s *ast.Schema) []string {
	if s == nil {
		return nil
	}
	var refs []string
	if s.IsRef() {
		refs = append(refs, refToName(s.Ref))
		return refs
	}
	if s.Items != nil {
		refs = append(refs, collectRefs(s.Items)...)
	}
	for _, prop := range s.Properties {
		refs = append(refs, collectRefs(prop)...)
	}
	for _, sub := range s.AllOf {
		refs = append(refs, collectRefs(sub)...)
	}
	for _, sub := range s.OneOf {
		refs = append(refs, collectRefs(sub)...)
	}
	for _, sub := range s.AnyOf {
		refs = append(refs, collectRefs(sub)...)
	}
	if s.AdditionalProperties != nil {
		refs = append(refs, collectRefs(s.AdditionalProperties)...)
	}
	return refs
}

// Package planner transforms the validated bound AST into a Go+CLI codegen IR.
// This is the decision-making stage: type mapping, command grouping, flag strategy.
package planner

import (
	"sort"
	"strings"
	"unicode"

	"github.com/iankaplan/ixian/internal/ast"
	"github.com/iankaplan/ixian/internal/binder"
	"github.com/iankaplan/ixian/internal/ir"
)

// Plan transforms a bound spec into a codegen IR.
func Plan(bound *binder.BoundSpec) *ir.Plan {
	p := &planner{bound: bound}
	return p.plan()
}

type planner struct {
	bound *binder.BoundSpec
}

func (p *planner) plan() *ir.Plan {
	plan := &ir.Plan{
		PackageName:    toKebabCase(p.bound.Spec.Info.Title),
		APITitle:       p.bound.Spec.Info.Title,
		APIDescription: p.bound.Spec.Info.Description,
	}
	if plan.PackageName == "" {
		plan.PackageName = "generated"
	}

	// Map server URL and descriptions
	if len(p.bound.Spec.Servers) > 0 {
		plan.ClientConfig.BaseURL = p.bound.Spec.Servers[0].URL
		for _, srv := range p.bound.Spec.Servers {
			plan.ClientConfig.ServerDescriptions = append(plan.ClientConfig.ServerDescriptions, ir.ServerDescription{
				URL:         srv.URL,
				Description: srv.Description,
			})
		}
	}

	// Map security schemes (sorted for deterministic output)
	if p.bound.Spec.Components != nil {
		schemeNames := make([]string, 0, len(p.bound.Spec.Components.SecuritySchemes))
		for name := range p.bound.Spec.Components.SecuritySchemes {
			schemeNames = append(schemeNames, name)
		}
		sort.Strings(schemeNames)
		for _, name := range schemeNames {
			if auth := p.planAuth(name, p.bound.Spec.Components.SecuritySchemes[name]); auth != nil {
				plan.ClientConfig.AuthSchemes = append(plan.ClientConfig.AuthSchemes, *auth)
			}
		}
	}

	// Generate types from schemas (sorted for deterministic output)
	typeNames := make([]string, 0, len(p.bound.SymbolTable))
	for name := range p.bound.SymbolTable {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)
	for _, name := range typeNames {
		goType := p.planType(name, p.bound.SymbolTable[name])
		plan.Types = append(plan.Types, goType)
	}

	// Generate commands from operations (sorted by path for deterministic output)
	paths := make([]string, 0, len(p.bound.Spec.Paths))
	for path := range p.bound.Spec.Paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		pi := p.bound.Spec.Paths[path]
		for _, pair := range []struct {
			method string
			op     *ast.Operation
		}{
			{"DELETE", pi.Delete},
			{"GET", pi.Get},
			{"PATCH", pi.Patch},
			{"POST", pi.Post},
			{"PUT", pi.Put},
		} {
			if pair.op == nil {
				continue
			}
			cmd := p.planCommand(path, pair.method, pair.op)
			plan.Commands = append(plan.Commands, cmd)
		}
	}

	return plan
}

func (p *planner) planType(name string, s *ast.Schema) ir.GoType {
	if len(s.Enum) > 0 {
		return ir.GoType{
			Name:        name,
			Description: s.Description,
			Kind:        ir.TypeKindEnum,
			EnumValues:  s.Enum,
			AliasOf:     "string",
		}
	}

	gt := ir.GoType{
		Name:        name,
		Description: s.Description,
		Kind:        ir.TypeKindStruct,
	}

	requiredSet := make(map[string]bool)
	for _, r := range s.Required {
		requiredSet[r] = true
	}

	propNames := make([]string, 0, len(s.Properties))
	for name := range s.Properties {
		propNames = append(propNames, name)
	}
	sort.Strings(propNames)

	for _, propName := range propNames {
		prop := s.Properties[propName]
		field := ir.GoField{
			Name:     toPascalCase(propName),
			JSONName: propName,
			Type:     p.mapType(prop),
			Required: requiredSet[propName],
			Comment:  prop.Description,
		}
		gt.Fields = append(gt.Fields, field)
	}

	return gt
}

func (p *planner) planCommand(path, method string, op *ast.Operation) ir.GoCommand {
	group := "default"
	if len(op.Tags) > 0 {
		group = op.Tags[0]
	}

	cmd := ir.GoCommand{
		Name:        toKebabCase(op.OperationID),
		GroupName:   group,
		OperationID: op.OperationID,
		HTTPMethod:  method,
		Path:        path,
		Summary:     op.Summary,
		Description: op.Description,
	}

	// Map parameters to flags
	for _, param := range op.Parameters {
		flag := ir.GoFlag{
			Name:        toKebabCase(param.Name),
			GoName:      toCamelCase(param.Name),
			Type:        p.mapType(param.Schema),
			Description: param.Description,
			Required:    param.Required,
			In:          param.In,
		}
		cmd.Flags = append(cmd.Flags, flag)
	}

	// Map request body
	if op.RequestBody != nil {
		cmd.BodyDescription = op.RequestBody.Description
		if mt, ok := op.RequestBody.Content["application/json"]; ok && mt.Schema != nil {
			cmd.BodyType = p.schemaToTypeName(mt.Schema)
		}
	}

	// Map responses (sorted for deterministic output)
	respCodes := make([]string, 0, len(op.Responses))
	for code := range op.Responses {
		respCodes = append(respCodes, code)
	}
	sort.Strings(respCodes)
	for _, code := range respCodes {
		resp := op.Responses[code]
		cmd.ResponseDescriptions = append(cmd.ResponseDescriptions, ir.ResponseDescription{
			StatusCode:  code,
			Description: resp.Description,
		})
		if strings.HasPrefix(code, "2") && cmd.ResponseType == "" {
			if mt, ok := resp.Content["application/json"]; ok && mt.Schema != nil {
				cmd.ResponseType = p.mapType(mt.Schema)
			}
		}
	}

	return cmd
}

func (p *planner) mapType(s *ast.Schema) string {
	if s == nil {
		return "any"
	}

	switch s.Type {
	case "string":
		switch s.Format {
		case "date-time", "date":
			return "time.Time"
		case "binary":
			return "[]byte"
		default:
			return "string"
		}
	case "integer":
		switch s.Format {
		case "int32":
			return "int32"
		case "int64":
			return "int64"
		default:
			return "int"
		}
	case "number":
		switch s.Format {
		case "float":
			return "float32"
		default:
			return "float64"
		}
	case "boolean":
		return "bool"
	case "array":
		if s.Items != nil {
			return "[]" + p.mapType(s.Items)
		}
		return "[]any"
	case "object":
		if s.AdditionalProperties != nil {
			return "map[string]" + p.mapType(s.AdditionalProperties)
		}
		// Named object type — look it up
		return p.schemaToTypeName(s)
	default:
		return "any"
	}
}

func (p *planner) schemaToTypeName(s *ast.Schema) string {
	// Find the schema in the symbol table by pointer identity
	for name, sym := range p.bound.SymbolTable {
		if sym == s {
			return name
		}
	}
	return "any"
}

func (p *planner) planAuth(name string, scheme *ast.SecurityScheme) *ir.AuthScheme {
	switch scheme.Type {
	case "http":
		switch scheme.Scheme {
		case "bearer":
			return &ir.AuthScheme{
				Name:     name,
				Type:     "bearer",
				FlagName: "auth-token",
				GoName:   "authToken",
			}
		case "basic":
			return &ir.AuthScheme{
				Name:     name,
				Type:     "basic",
				FlagName: "auth-basic",
				GoName:   "authBasic",
			}
		}
	case "apiKey":
		flagName := toKebabCase(scheme.Name)
		return &ir.AuthScheme{
			Name:       name,
			Type:       "apiKey",
			FlagName:   flagName,
			GoName:     toCamelCase(scheme.Name),
			HeaderName: scheme.Name,
			In:         scheme.In,
		}
	}
	return nil
}

func toPascalCase(s string) string {
	parts := splitIdentifier(s)
	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteRune(unicode.ToUpper(rune(part[0])))
			result.WriteString(part[1:])
		}
	}
	return result.String()
}

func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if len(pascal) == 0 {
		return ""
	}
	return string(unicode.ToLower(rune(pascal[0]))) + pascal[1:]
}

func toKebabCase(s string) string {
	parts := splitIdentifier(s)
	for i, part := range parts {
		parts[i] = strings.ToLower(part)
	}
	return strings.Join(parts, "-")
}

// splitIdentifier splits on camelCase boundaries, underscores, and hyphens.
func splitIdentifier(s string) []string {
	var parts []string
	var current strings.Builder

	for i, r := range s {
		if r == '_' || r == '-' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}
		if unicode.IsUpper(r) && i > 0 {
			prev := rune(s[i-1])
			if unicode.IsLower(prev) || prev == '_' || prev == '-' {
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
			}
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

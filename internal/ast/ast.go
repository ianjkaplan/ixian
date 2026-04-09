// Package ast defines the raw OpenAPI AST structures produced by the parser.
// These types mirror the OpenAPI 3.x specification and contain unresolved $ref pointers.
package ast

// Spec is the top-level OpenAPI 3.x document.
type Spec struct {
	OpenAPI    string                `yaml:"openapi" json:"openapi"`
	Info       Info                  `yaml:"info" json:"info"`
	Servers    []Server              `yaml:"servers,omitempty" json:"servers,omitempty"`
	Paths      map[string]*PathItem  `yaml:"paths" json:"paths"`
	Components *Components           `yaml:"components,omitempty" json:"components,omitempty"`
	Security   []SecurityRequirement `yaml:"security,omitempty" json:"security,omitempty"`
}

type Info struct {
	Title       string `yaml:"title" json:"title"`
	Version     string `yaml:"version" json:"version"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

type Server struct {
	URL         string `yaml:"url" json:"url"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

type PathItem struct {
	Get    *Operation `yaml:"get,omitempty" json:"get,omitempty"`
	Post   *Operation `yaml:"post,omitempty" json:"post,omitempty"`
	Put    *Operation `yaml:"put,omitempty" json:"put,omitempty"`
	Delete *Operation `yaml:"delete,omitempty" json:"delete,omitempty"`
	Patch  *Operation `yaml:"patch,omitempty" json:"patch,omitempty"`
}

type Operation struct {
	OperationID string                `yaml:"operationId" json:"operationId"`
	Summary     string                `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description string                `yaml:"description,omitempty" json:"description,omitempty"`
	Tags        []string              `yaml:"tags,omitempty" json:"tags,omitempty"`
	Parameters  []Parameter           `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	RequestBody *RequestBody          `yaml:"requestBody,omitempty" json:"requestBody,omitempty"`
	Responses   map[string]Response   `yaml:"responses,omitempty" json:"responses,omitempty"`
	Security    []SecurityRequirement `yaml:"security,omitempty" json:"security,omitempty"`
}

// SecurityRequirement maps security scheme names to their scopes.
type SecurityRequirement map[string][]string

type Parameter struct {
	Name        string  `yaml:"name" json:"name"`
	In          string  `yaml:"in" json:"in"` // query, path, header, cookie
	Required    bool    `yaml:"required,omitempty" json:"required,omitempty"`
	Description string  `yaml:"description,omitempty" json:"description,omitempty"`
	Schema      *Schema `yaml:"schema,omitempty" json:"schema,omitempty"`
}

type RequestBody struct {
	Required    bool                 `yaml:"required,omitempty" json:"required,omitempty"`
	Description string               `yaml:"description,omitempty" json:"description,omitempty"`
	Content     map[string]MediaType `yaml:"content" json:"content"`
}

type MediaType struct {
	Schema *Schema `yaml:"schema,omitempty" json:"schema,omitempty"`
}

type Response struct {
	Description string               `yaml:"description" json:"description"`
	Content     map[string]MediaType `yaml:"content,omitempty" json:"content,omitempty"`
}

type Schema struct {
	Ref                  string             `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Type                 string             `yaml:"type,omitempty" json:"type,omitempty"`
	Format               string             `yaml:"format,omitempty" json:"format,omitempty"`
	Description          string             `yaml:"description,omitempty" json:"description,omitempty"`
	Properties           map[string]*Schema `yaml:"properties,omitempty" json:"properties,omitempty"`
	Items                *Schema            `yaml:"items,omitempty" json:"items,omitempty"`
	Required             []string           `yaml:"required,omitempty" json:"required,omitempty"`
	Enum                 []string           `yaml:"enum,omitempty" json:"enum,omitempty"`
	Nullable             bool               `yaml:"nullable,omitempty" json:"nullable,omitempty"`
	AllOf                []*Schema          `yaml:"allOf,omitempty" json:"allOf,omitempty"`
	OneOf                []*Schema          `yaml:"oneOf,omitempty" json:"oneOf,omitempty"`
	AnyOf                []*Schema          `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	AdditionalProperties *Schema            `yaml:"additionalProperties,omitempty" json:"additionalProperties,omitempty"`
}

type Components struct {
	Schemas         map[string]*Schema         `yaml:"schemas,omitempty" json:"schemas,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `yaml:"securitySchemes,omitempty" json:"securitySchemes,omitempty"`
}

// SecurityScheme represents an OpenAPI security scheme definition.
type SecurityScheme struct {
	Type         string `yaml:"type" json:"type"`                         // apiKey, http, oauth2, openIdConnect
	Scheme       string `yaml:"scheme,omitempty" json:"scheme,omitempty"` // bearer, basic (for type=http)
	BearerFormat string `yaml:"bearerFormat,omitempty" json:"bearerFormat,omitempty"`
	Name         string `yaml:"name,omitempty" json:"name,omitempty"` // header/query/cookie name (for type=apiKey)
	In           string `yaml:"in,omitempty" json:"in,omitempty"`     // header, query, cookie (for type=apiKey)
	Description  string `yaml:"description,omitempty" json:"description,omitempty"`
}

// IsRef returns true if this schema is a $ref pointer.
func (s *Schema) IsRef() bool {
	return s != nil && s.Ref != ""
}

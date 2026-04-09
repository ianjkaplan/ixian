// Package ir defines the intermediate representation used between the planner and emitter.
// These types are Go-specific and CLI-specific — the emitter prints them without making decisions.
package ir

// Plan is the top-level codegen IR.
type Plan struct {
	PackageName  string
	Types        []GoType
	Commands     []GoCommand
	ClientConfig ClientConfig
}

// GoType represents a generated Go type (struct, enum, or alias).
type GoType struct {
	Name        string
	Description string
	Kind        TypeKind
	Fields      []GoField // for structs
	EnumValues  []string  // for enums
	AliasOf     string    // for aliases
}

type TypeKind int

const (
	TypeKindStruct TypeKind = iota
	TypeKindEnum
	TypeKindAlias
)

// GoField is a single field in a generated struct.
type GoField struct {
	Name     string // Go name (PascalCase)
	JSONName string // JSON tag name
	Type     string // Go type expression (e.g., "string", "*Owner", "[]Pet")
	Required bool
	Comment  string
}

// GoCommand represents a single CLI command in the generated Cobra tree.
type GoCommand struct {
	Name         string // command name (e.g., "list")
	GroupName    string // parent group (e.g., "pets")
	OperationID  string
	HTTPMethod   string
	Path         string
	Summary      string
	Description  string
	Flags        []GoFlag
	BodyType     string // Go type for request body, empty if none
	ResponseType string // Go type for primary success response
}

// GoFlag represents a single CLI flag.
type GoFlag struct {
	Name         string // flag name (e.g., "limit")
	GoName       string // Go variable name
	Type         string // Go type
	Description  string
	Required     bool
	DefaultValue string
	In           string // query, path, header
}

// ClientConfig holds config for the generated HTTP client.
type ClientConfig struct {
	BaseURL     string
	AuthSchemes []AuthScheme
}

// AuthScheme represents a resolved auth mechanism for the generated CLI.
type AuthScheme struct {
	Name     string // scheme name from the spec (e.g., "bearerAuth")
	Type     string // "bearer", "basic", "apiKey"
	FlagName string // CLI flag name (e.g., "auth-token", "api-key")
	GoName   string // Go variable name (e.g., "authToken", "apiKey")
	// apiKey-specific
	HeaderName string // header name to send (e.g., "X-API-Key")
	In         string // "header" or "query"
}

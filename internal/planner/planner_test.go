package planner

import (
	"path/filepath"
	"testing"

	"github.com/iankaplan/ixian/internal/binder"
	"github.com/iankaplan/ixian/internal/ir"
	"github.com/iankaplan/ixian/internal/parser"
)

func testdataPath(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func TestPlanPetstore(t *testing.T) {
	spec, err := parser.Parse(testdataPath("petstore.yaml"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bound, err := binder.Bind(spec)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}

	plan := Plan(bound)

	// Check base URL
	if plan.ClientConfig.BaseURL != "https://api.petstore.example.com/v1" {
		t.Errorf("baseURL = %q, want petstore URL", plan.ClientConfig.BaseURL)
	}

	// Check types were generated
	if len(plan.Types) != 4 {
		t.Errorf("types count = %d, want 4", len(plan.Types))
	}

	typeByName := make(map[string]ir.GoType)
	for _, gt := range plan.Types {
		typeByName[gt.Name] = gt
	}

	pet, ok := typeByName["Pet"]
	if !ok {
		t.Fatal("missing Pet type")
	}
	if pet.Kind != ir.TypeKindStruct {
		t.Errorf("Pet kind = %v, want struct", pet.Kind)
	}
	if len(pet.Fields) != 5 {
		t.Errorf("Pet fields = %d, want 5", len(pet.Fields))
	}

	errType, ok := typeByName["Error"]
	if !ok {
		t.Fatal("missing Error type")
	}
	if errType.Kind != ir.TypeKindStruct {
		t.Errorf("Error kind = %v, want struct", errType.Kind)
	}

	// Check commands were generated
	if len(plan.Commands) != 5 {
		t.Errorf("commands count = %d, want 5", len(plan.Commands))
	}

	cmdByOpID := make(map[string]ir.GoCommand)
	for _, cmd := range plan.Commands {
		cmdByOpID[cmd.OperationID] = cmd
	}

	listPets, ok := cmdByOpID["listPets"]
	if !ok {
		t.Fatal("missing listPets command")
	}
	if listPets.GroupName != "pets" {
		t.Errorf("listPets group = %q, want %q", listPets.GroupName, "pets")
	}
	if listPets.HTTPMethod != "GET" {
		t.Errorf("listPets method = %q, want GET", listPets.HTTPMethod)
	}
	if len(listPets.Flags) != 2 {
		t.Errorf("listPets flags = %d, want 2", len(listPets.Flags))
	}

	createPet, ok := cmdByOpID["createPet"]
	if !ok {
		t.Fatal("missing createPet command")
	}
	if createPet.BodyType != "CreatePetRequest" {
		t.Errorf("createPet bodyType = %q, want %q", createPet.BodyType, "CreatePetRequest")
	}

	getPet, ok := cmdByOpID["getPet"]
	if !ok {
		t.Fatal("missing getPet command")
	}
	if len(getPet.Flags) != 1 {
		t.Errorf("getPet flags = %d, want 1", len(getPet.Flags))
	}
	if getPet.Flags[0].In != "path" {
		t.Errorf("getPet petId flag in = %q, want %q", getPet.Flags[0].In, "path")
	}
	if !getPet.Flags[0].Required {
		t.Error("getPet petId flag should be required")
	}

	listOwners, ok := cmdByOpID["listOwners"]
	if !ok {
		t.Fatal("missing listOwners command")
	}
	if listOwners.GroupName != "owners" {
		t.Errorf("listOwners group = %q, want %q", listOwners.GroupName, "owners")
	}
}

func TestTypeMapping(t *testing.T) {
	spec, err := parser.ParseBytes([]byte(`
openapi: "3.0.3"
info:
  title: Test
  version: "1.0.0"
paths: {}
components:
  schemas:
    Typed:
      type: object
      properties:
        str:
          type: string
        int32:
          type: integer
          format: int32
        int64:
          type: integer
          format: int64
        float:
          type: number
          format: float
        double:
          type: number
        flag:
          type: boolean
        date:
          type: string
          format: date-time
        bin:
          type: string
          format: binary
        arr:
          type: array
          items:
            type: string
        extra:
          type: object
          additionalProperties:
            type: string
`), ".yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bound, err := binder.Bind(spec)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}

	plan := Plan(bound)
	if len(plan.Types) != 1 {
		t.Fatalf("types = %d, want 1", len(plan.Types))
	}

	fields := make(map[string]string)
	for _, f := range plan.Types[0].Fields {
		fields[f.JSONName] = f.Type
	}

	expected := map[string]string{
		"str":    "string",
		"int32":  "int32",
		"int64":  "int64",
		"float":  "float32",
		"double": "float64",
		"flag":   "bool",
		"date":   "time.Time",
		"bin":    "[]byte",
		"arr":    "[]string",
		"extra":  "map[string]string",
	}

	for name, wantType := range expected {
		if got := fields[name]; got != wantType {
			t.Errorf("field %s type = %q, want %q", name, got, wantType)
		}
	}
}

func TestCaseConversions(t *testing.T) {
	tests := []struct {
		input  string
		pascal string
		camel  string
		kebab  string
	}{
		{"listPets", "ListPets", "listPets", "list-pets"},
		{"pet_id", "PetId", "petId", "pet-id"},
		{"HTTPMethod", "HTTPMethod", "hTTPMethod", "httpmethod"},
	}

	for _, tt := range tests {
		if got := toPascalCase(tt.input); got != tt.pascal {
			t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.pascal)
		}
		if got := toCamelCase(tt.input); got != tt.camel {
			t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, got, tt.camel)
		}
		if got := toKebabCase(tt.input); got != tt.kebab {
			t.Errorf("toKebabCase(%q) = %q, want %q", tt.input, got, tt.kebab)
		}
	}
}

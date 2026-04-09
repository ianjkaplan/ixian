package parser

import (
	"path/filepath"
	"testing"
)

func testdataPath(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func TestParsePetstoreYAML(t *testing.T) {
	t.Parallel()
	spec, err := Parse(testdataPath("petstore.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if spec.OpenAPI != "3.0.3" {
		t.Errorf("openapi = %q, want %q", spec.OpenAPI, "3.0.3")
	}
	if spec.Info.Title != "Petstore" {
		t.Errorf("title = %q, want %q", spec.Info.Title, "Petstore")
	}
	if spec.Info.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", spec.Info.Version, "1.0.0")
	}

	// Check paths
	if len(spec.Paths) != 3 {
		t.Errorf("paths count = %d, want 3", len(spec.Paths))
	}

	petsPath, ok := spec.Paths["/pets"]
	if !ok {
		t.Fatal("missing /pets path")
	}
	if petsPath.Get == nil {
		t.Fatal("/pets GET is nil")
	}
	if petsPath.Get.OperationID != "listPets" {
		t.Errorf("operationId = %q, want %q", petsPath.Get.OperationID, "listPets")
	}
	if len(petsPath.Get.Parameters) != 2 {
		t.Errorf("listPets params count = %d, want 2", len(petsPath.Get.Parameters))
	}
	if petsPath.Post == nil {
		t.Fatal("/pets POST is nil")
	}
	if petsPath.Post.RequestBody == nil {
		t.Fatal("createPet requestBody is nil")
	}

	petIdPath, ok := spec.Paths["/pets/{petId}"]
	if !ok {
		t.Fatal("missing /pets/{petId} path")
	}
	if petIdPath.Get == nil {
		t.Fatal("/pets/{petId} GET is nil")
	}
	if petIdPath.Delete == nil {
		t.Fatal("/pets/{petId} DELETE is nil")
	}

	// Check components
	if spec.Components == nil {
		t.Fatal("components is nil")
	}
	if len(spec.Components.Schemas) != 4 {
		t.Errorf("schemas count = %d, want 4", len(spec.Components.Schemas))
	}

	petSchema, ok := spec.Components.Schemas["Pet"]
	if !ok {
		t.Fatal("missing Pet schema")
	}
	if len(petSchema.Required) != 2 {
		t.Errorf("Pet required count = %d, want 2", len(petSchema.Required))
	}
	if len(petSchema.Properties) != 5 {
		t.Errorf("Pet properties count = %d, want 5", len(petSchema.Properties))
	}

	// Check $ref is preserved (not resolved)
	ownerProp := petSchema.Properties["owner"]
	if ownerProp == nil {
		t.Fatal("Pet.owner property is nil")
	}
	if !ownerProp.IsRef() {
		t.Error("Pet.owner should be a $ref")
	}
	if ownerProp.Ref != "#/components/schemas/Owner" {
		t.Errorf("Pet.owner $ref = %q, want %q", ownerProp.Ref, "#/components/schemas/Owner")
	}

	// Check enum
	statusProp := petSchema.Properties["status"]
	if statusProp == nil {
		t.Fatal("Pet.status property is nil")
	}
	if len(statusProp.Enum) != 3 {
		t.Errorf("Pet.status enum count = %d, want 3", len(statusProp.Enum))
	}
}

func TestParseSecuritySchemes(t *testing.T) {
	t.Parallel()
	spec, err := Parse(testdataPath("petstore.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check global security requirement
	if len(spec.Security) != 1 {
		t.Fatalf("security requirements count = %d, want 1", len(spec.Security))
	}
	if _, ok := spec.Security[0]["bearerAuth"]; !ok {
		t.Error("missing bearerAuth in global security")
	}

	// Check security schemes
	if spec.Components == nil {
		t.Fatal("components is nil")
	}
	if len(spec.Components.SecuritySchemes) != 2 {
		t.Fatalf("securitySchemes count = %d, want 2", len(spec.Components.SecuritySchemes))
	}

	bearer, ok := spec.Components.SecuritySchemes["bearerAuth"]
	if !ok {
		t.Fatal("missing bearerAuth scheme")
	}
	if bearer.Type != "http" {
		t.Errorf("bearerAuth type = %q, want %q", bearer.Type, "http")
	}
	if bearer.Scheme != "bearer" {
		t.Errorf("bearerAuth scheme = %q, want %q", bearer.Scheme, "bearer")
	}
	if bearer.BearerFormat != "JWT" {
		t.Errorf("bearerAuth bearerFormat = %q, want %q", bearer.BearerFormat, "JWT")
	}

	apiKey, ok := spec.Components.SecuritySchemes["apiKeyAuth"]
	if !ok {
		t.Fatal("missing apiKeyAuth scheme")
	}
	if apiKey.Type != "apiKey" {
		t.Errorf("apiKeyAuth type = %q, want %q", apiKey.Type, "apiKey")
	}
	if apiKey.Name != "X-API-Key" {
		t.Errorf("apiKeyAuth name = %q, want %q", apiKey.Name, "X-API-Key")
	}
	if apiKey.In != "header" {
		t.Errorf("apiKeyAuth in = %q, want %q", apiKey.In, "header")
	}
}

func TestParseBytesJSON(t *testing.T) {
	t.Parallel()
	jsonSpec := []byte(`{
		"openapi": "3.0.3",
		"info": {"title": "Test", "version": "1.0.0"},
		"paths": {}
	}`)

	spec, err := ParseBytes(jsonSpec, ".json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.Info.Title != "Test" {
		t.Errorf("title = %q, want %q", spec.Info.Title, "Test")
	}
}

func TestParseMissingFields(t *testing.T) {
	t.Parallel()
	_, err := ParseBytes([]byte(`{}`), ".json")
	if err == nil {
		t.Error("expected error for missing openapi field")
	}

	_, err = ParseBytes([]byte(`{"openapi": "3.0.3"}`), ".json")
	if err == nil {
		t.Error("expected error for missing info.title")
	}
}

func TestParseInvalidYAML(t *testing.T) {
	t.Parallel()
	_, err := ParseBytes([]byte(`{invalid`), ".yaml")
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParseFileNotFound(t *testing.T) {
	t.Parallel()
	_, err := Parse("/nonexistent/file.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

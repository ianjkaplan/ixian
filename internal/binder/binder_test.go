package binder

import (
	"path/filepath"
	"testing"

	"github.com/iankaplan/ixian/internal/parser"
)

func testdataPath(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func TestBindPetstore(t *testing.T) {
	t.Parallel()
	spec, err := parser.Parse(testdataPath("petstore.yaml"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	bound, err := Bind(spec)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}

	// Symbol table should have all 4 schemas
	if len(bound.SymbolTable) != 4 {
		t.Errorf("symbol table size = %d, want 4", len(bound.SymbolTable))
	}
	for _, name := range []string{"Pet", "CreatePetRequest", "Owner", "Error"} {
		if _, ok := bound.SymbolTable[name]; !ok {
			t.Errorf("missing symbol: %s", name)
		}
	}

	// Pet depends on Owner
	petDeps, ok := bound.Dependencies["Pet"]
	if !ok {
		t.Fatal("missing Pet dependencies")
	}
	found := false
	for _, d := range petDeps {
		if d == "Owner" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Pet deps = %v, expected Owner", petDeps)
	}

	// After binding, the listPets response items should point directly to the Pet schema
	petsPath := bound.Spec.Paths["/pets"]
	if petsPath == nil || petsPath.Get == nil {
		t.Fatal("/pets GET is nil")
	}
	resp200 := petsPath.Get.Responses["200"]
	mt := resp200.Content["application/json"]
	if mt.Schema == nil || mt.Schema.Items == nil {
		t.Fatal("listPets response schema.items is nil")
	}
	// Items should now be the resolved Pet schema (not a $ref)
	if mt.Schema.Items.IsRef() {
		t.Error("listPets response items should be resolved, not a $ref")
	}
	if mt.Schema.Items.Type != "object" {
		t.Errorf("listPets items type = %q, want %q", mt.Schema.Items.Type, "object")
	}
	if len(mt.Schema.Items.Properties) != 5 {
		t.Errorf("resolved Pet properties = %d, want 5", len(mt.Schema.Items.Properties))
	}

	// createPet request body should be resolved
	if petsPath.Post == nil || petsPath.Post.RequestBody == nil {
		t.Fatal("createPet requestBody is nil")
	}
	bodySchema := petsPath.Post.RequestBody.Content["application/json"].Schema
	if bodySchema.IsRef() {
		t.Error("createPet body should be resolved")
	}
	if bodySchema.Type != "object" {
		t.Errorf("createPet body type = %q, want %q", bodySchema.Type, "object")
	}
}

func TestBindUnresolvedRef(t *testing.T) {
	t.Parallel()
	spec, err := parser.ParseBytes([]byte(`
openapi: "3.0.3"
info:
  title: Test
  version: "1.0.0"
paths:
  /foo:
    get:
      operationId: getFoo
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/DoesNotExist"
`), ".yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, err = Bind(spec)
	if err == nil {
		t.Error("expected error for unresolved $ref")
	}
}

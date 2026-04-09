package checker

import (
	"path/filepath"
	"testing"

	"github.com/iankaplan/ixian/internal/binder"
	"github.com/iankaplan/ixian/internal/parser"
)

func testdataPath(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func TestCheckPetstore(t *testing.T) {
	spec, err := parser.Parse(testdataPath("petstore.yaml"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bound, err := binder.Bind(spec)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}

	result := Check(bound)

	if result.HasErrors() {
		for _, d := range result.Errors() {
			t.Errorf("unexpected error: %s", d)
		}
	}
	// Warnings are acceptable
	for _, d := range result.Warnings() {
		t.Logf("warning: %s", d)
	}
}

func TestCheckDuplicateOperationID(t *testing.T) {
	spec, err := parser.ParseBytes([]byte(`
openapi: "3.0.3"
info:
  title: Test
  version: "1.0.0"
paths:
  /foo:
    get:
      operationId: myOp
      responses:
        "200":
          description: ok
  /bar:
    get:
      operationId: myOp
      responses:
        "200":
          description: ok
`), ".yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bound, err := binder.Bind(spec)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}

	result := Check(bound)
	if !result.HasErrors() {
		t.Error("expected error for duplicate operationId")
	}

	found := false
	for _, d := range result.Errors() {
		if d.Message != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected duplicate operationId diagnostic")
	}
}

func TestCheckMissingOperationID(t *testing.T) {
	spec, err := parser.ParseBytes([]byte(`
openapi: "3.0.3"
info:
  title: Test
  version: "1.0.0"
paths:
  /foo:
    get:
      responses:
        "200":
          description: ok
`), ".yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bound, err := binder.Bind(spec)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}

	result := Check(bound)
	if !result.HasErrors() {
		t.Error("expected error for missing operationId")
	}
}

func TestCheckMissingPathParam(t *testing.T) {
	spec, err := parser.ParseBytes([]byte(`
openapi: "3.0.3"
info:
  title: Test
  version: "1.0.0"
paths:
  /foo/{id}:
    get:
      operationId: getFoo
      responses:
        "200":
          description: ok
`), ".yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bound, err := binder.Bind(spec)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}

	result := Check(bound)
	if !result.HasErrors() {
		t.Error("expected error for undeclared path parameter")
	}
}

func TestCheckRequiredFieldNotInProperties(t *testing.T) {
	spec, err := parser.ParseBytes([]byte(`
openapi: "3.0.3"
info:
  title: Test
  version: "1.0.0"
paths: {}
components:
  schemas:
    Broken:
      type: object
      required:
        - nonexistent
      properties:
        name:
          type: string
`), ".yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bound, err := binder.Bind(spec)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}

	result := Check(bound)
	if !result.HasErrors() {
		t.Error("expected error for required field not in properties")
	}
}

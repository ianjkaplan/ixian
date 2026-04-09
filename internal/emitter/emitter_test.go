package emitter

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/iankaplan/ixian/internal/binder"
	"github.com/iankaplan/ixian/internal/parser"
	"github.com/iankaplan/ixian/internal/planner"
)

func testdataPath(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func TestEmitPetstore(t *testing.T) {
	spec, err := parser.Parse(testdataPath("petstore.yaml"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bound, err := binder.Bind(spec)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	plan := planner.Plan(bound)

	files, err := Emit(plan)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	fileMap := make(map[string]string)
	for _, f := range files {
		fileMap[f.Name] = string(f.Content)
	}

	// Check expected files exist
	expectedFiles := []string{
		"types/types.go",
		"client/client.go",
		"cmd/root.go",
		"cmd/pets.go",
		"cmd/owners.go",
		"main.go",
	}
	for _, name := range expectedFiles {
		if _, ok := fileMap[name]; !ok {
			t.Errorf("missing file: %s", name)
		}
	}

	// Check types.go content
	typesContent := fileMap["types/types.go"]
	if !strings.Contains(typesContent, "type Pet struct") {
		t.Error("types.go missing Pet struct")
	}
	if !strings.Contains(typesContent, "type Owner struct") {
		t.Error("types.go missing Owner struct")
	}
	if !strings.Contains(typesContent, "type Error struct") {
		t.Error("types.go missing Error struct")
	}
	if !strings.Contains(typesContent, `json:"id"`) {
		t.Error("types.go missing json tags")
	}

	// Check client.go content
	clientContent := fileMap["client/client.go"]
	if !strings.Contains(clientContent, "type Client struct") {
		t.Error("client.go missing Client struct")
	}
	if !strings.Contains(clientContent, "func New(") {
		t.Error("client.go missing New constructor")
	}
	if !strings.Contains(clientContent, "func (c *Client) Do(") {
		t.Error("client.go missing Do method")
	}

	// Check root.go content
	rootContent := fileMap["cmd/root.go"]
	if !strings.Contains(rootContent, "base-url") {
		t.Error("root.go missing base-url flag")
	}
	if !strings.Contains(rootContent, "petstore.example.com") {
		t.Error("root.go missing default base URL")
	}

	// Check custom headers support in root.go
	if !strings.Contains(rootContent, `"header"`) {
		t.Error("root.go missing --header flag")
	}
	if !strings.Contains(rootContent, "headers []string") {
		t.Error("root.go missing headers slice variable")
	}
	if !strings.Contains(rootContent, "key:value") {
		t.Error("root.go missing header format description")
	}

	// Check custom headers support in client.go
	if !strings.Contains(clientContent, "Headers") {
		t.Error("client.go missing Headers field")
	}
	if !strings.Contains(clientContent, "c.Headers") {
		t.Error("client.go missing custom header application logic")
	}

	// Check auth in client.go
	if !strings.Contains(clientContent, "authToken") {
		t.Error("client.go missing authToken field for bearer auth")
	}
	if !strings.Contains(clientContent, `"Bearer "`) {
		t.Error("client.go missing Bearer prefix in auth logic")
	}
	if !strings.Contains(clientContent, "X-API-Key") {
		t.Error("client.go missing X-API-Key header for apiKey auth")
	}

	// Check auth flags in root.go
	if !strings.Contains(rootContent, "auth-token") {
		t.Error("root.go missing auth-token flag")
	}
	if !strings.Contains(rootContent, "Bearer authentication token") {
		t.Error("root.go missing bearer flag description")
	}

	// Check pets.go content
	petsContent := fileMap["cmd/pets.go"]
	if !strings.Contains(petsContent, "petsCmd") {
		t.Error("pets.go missing petsCmd")
	}
	if !strings.Contains(petsContent, `"list-pets"`) {
		t.Error("pets.go missing list-pets command")
	}
	if !strings.Contains(petsContent, `"create-pet"`) {
		t.Error("pets.go missing create-pet command")
	}

	// Check owners.go content
	ownersContent := fileMap["cmd/owners.go"]
	if !strings.Contains(ownersContent, "ownersCmd") {
		t.Error("owners.go missing ownersCmd")
	}
}

func TestEmitDeterministic(t *testing.T) {
	spec, err := parser.Parse(testdataPath("petstore.yaml"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bound, err := binder.Bind(spec)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	plan := planner.Plan(bound)

	files1, err := Emit(plan)
	if err != nil {
		t.Fatalf("emit 1: %v", err)
	}
	files2, err := Emit(plan)
	if err != nil {
		t.Fatalf("emit 2: %v", err)
	}

	if len(files1) != len(files2) {
		t.Fatalf("file counts differ: %d vs %d", len(files1), len(files2))
	}

	// Build maps for comparison (order may differ)
	m1, m2 := make(map[string]string), make(map[string]string)
	for _, f := range files1 {
		m1[f.Name] = string(f.Content)
	}
	for _, f := range files2 {
		m2[f.Name] = string(f.Content)
	}

	for name, content1 := range m1 {
		content2, ok := m2[name]
		if !ok {
			t.Errorf("file %s missing in second emission", name)
			continue
		}
		if content1 != content2 {
			t.Errorf("file %s content differs between emissions", name)
		}
	}
}

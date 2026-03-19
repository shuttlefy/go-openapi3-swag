package main_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/shuttlefy/go-openapi3-swag/internal/builder"
	"github.com/shuttlefy/go-openapi3-swag/internal/extractor"
	"github.com/shuttlefy/go-openapi3-swag/internal/output"
	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

// runPipeline 执行完整流水线，返回序列化后的 JSON 字节。
func runPipeline(t *testing.T, dirs []string) []byte {
	t.Helper()

	p := &parser.GoParser{MaxDepth: -1}
	files, err := p.Parse(dirs)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	e := &extractor.GoExtractor{}
	result, err := e.Extract(files)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	b := builder.NewBuilder()
	doc, err := b.Build(result, files)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	tmp := t.TempDir() + "/out.json"
	if err := output.Write(doc, tmp); err != nil {
		t.Fatalf("write: %v", err)
	}

	data, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	return data
}

// TestEndToEnd_Annotations_JSON 使用 testdata/annotations 验证完整 JSON 输出。
func TestEndToEnd_Annotations_JSON(t *testing.T) {
	got := runPipeline(t, []string{"../../testdata/annotations"})

	var doc map[string]interface{}
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Info
	info := doc["info"].(map[string]interface{})
	if info["title"] != "Pet Store API" {
		t.Errorf("info.title = %v, want Pet Store API", info["title"])
	}
	if info["version"] != "1.0.0" {
		t.Errorf("info.version = %v, want 1.0.0", info["version"])
	}

	// Paths
	paths := doc["paths"].(map[string]interface{})
	if paths["/pets"] == nil {
		t.Error("missing path /pets")
	}
	if paths["/pets/{id}"] == nil {
		t.Error("missing path /pets/{id}")
	}

	// Components.Schemas
	components := doc["components"].(map[string]interface{})
	schemas := components["schemas"].(map[string]interface{})
	for _, name := range []string{"models.Pet", "models.CreatePetRequest", "models.ErrorResponse"} {
		if schemas[name] == nil {
			t.Errorf("missing schema %q", name)
		}
	}
}

// TestEndToEnd_Annotations_YAML 使用 testdata/annotations 验证基本 YAML 输出。
func TestEndToEnd_Annotations_YAML(t *testing.T) {
	p := &parser.GoParser{MaxDepth: -1}
	files, err := p.Parse([]string{"../../testdata/annotations"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	e := &extractor.GoExtractor{}
	result, err := e.Extract(files)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	b := builder.NewBuilder()
	doc, err := b.Build(result, files)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	tmp := t.TempDir() + "/out.yaml"
	if err := output.Write(doc, tmp); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	data, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("read yaml: %v", err)
	}

	yamlStr := string(data)
	for _, want := range []string{
		"openapi: 3.0.3",
		"Pet Store API",
		"/pets",
	} {
		if !strings.Contains(yamlStr, want) {
			t.Errorf("YAML output missing %q", want)
		}
	}
}

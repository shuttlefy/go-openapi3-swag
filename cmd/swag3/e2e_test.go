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

// TestEndToEnd_AnonStructField 验证匿名 struct 字段（[]struct{...} / *struct{...} / 嵌套）
// 能被完整流水线正确解析并出现在 components/schemas 中。
func TestEndToEnd_AnonStructField(t *testing.T) {
	dir := t.TempDir()

	write := func(name, src string) {
		t.Helper()
		if err := os.WriteFile(dir+"/"+name, []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// 全局 meta
	write("meta.go", `package ecs

// @title           ECS API
// @version         1.0.0
func main() {}
`)

	// 含匿名 struct 的模型
	write("models.go", `package ecs

// ECSInstanceTypeDescMapping 实例规格描述映射，字段为 []struct{...} 匿名切片。
type ECSInstanceTypeDescMapping struct {
	ComputingArchitecture []struct {
		Text  string `+"`"+`json:"text"`+"`"+`
		Value string `+"`"+`json:"value"`+"`"+`
	} `+"`"+`json:"computingArchitecture"`+"`"+`
	CustomizedFamily []struct {
		Text  string `+"`"+`json:"text"`+"`"+`
		Value string `+"`"+`json:"value"`+"`"+`
	} `+"`"+`json:"customizeFamily"`+"`"+`
}

// DescribeResult 含 *struct{...} 指针匿名字段。
type DescribeResult struct {
	Header *struct {
		RequestID string `+"`"+`json:"request_id"`+"`"+`
		TraceID   string `+"`"+`json:"trace_id"`+"`"+`
	} `+"`"+`json:"header"`+"`"+`
	Mapping ECSInstanceTypeDescMapping `+"`"+`json:"mapping"`+"`"+`
}
`)

	// 引用这些模型的 handler
	write("handler.go", `package ecs

// DescribeInstanceTypes 查询实例规格类型映射。
//
// @Summary  Describe instance type mapping
// @Tags     ecs
// @Produce  json
// @Success  200 {object} DescribeResult "ok"
// @Router   /ecs/instance-types [get]
func DescribeInstanceTypes() {}
`)

	got := runPipeline(t, []string{dir})

	var doc map[string]interface{}
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	schemas := doc["components"].(map[string]interface{})["schemas"].(map[string]interface{})

	// 顶层结构体必须存在
	for _, name := range []string{
		"ecs.DescribeResult",
		"ecs.ECSInstanceTypeDescMapping",
	} {
		if schemas[name] == nil {
			t.Errorf("missing schema %q", name)
		}
	}

	// 合成的匿名 struct 必须存在且含字段
	for _, synName := range []string{
		"ecs.ECSInstanceTypeDescMapping_ComputingArchitecture",
		"ecs.ECSInstanceTypeDescMapping_CustomizedFamily",
		"ecs.DescribeResult_Header",
	} {
		s, ok := schemas[synName]
		if !ok {
			t.Errorf("missing synthetic schema %q", synName)
			continue
		}
		props, ok := s.(map[string]interface{})["properties"].(map[string]interface{})
		if !ok || len(props) == 0 {
			t.Errorf("synthetic schema %q has no properties", synName)
		}
	}

	// ECSInstanceTypeDescMapping.computingArchitecture 应为 array，items 指向合成 schema
	mapping, ok := schemas["ecs.ECSInstanceTypeDescMapping"].(map[string]interface{})
	if !ok {
		t.Fatal("ecs.ECSInstanceTypeDescMapping is not an object")
	}
	props := mapping["properties"].(map[string]interface{})
	caField, ok := props["computingArchitecture"].(map[string]interface{})
	if !ok {
		t.Fatal("ECSInstanceTypeDescMapping missing computingArchitecture property")
	}
	if caField["type"] != "array" {
		t.Errorf("computingArchitecture.type = %v, want array", caField["type"])
	}
	items, ok := caField["items"].(map[string]interface{})
	if !ok || items["$ref"] == nil {
		t.Error("computingArchitecture.items missing $ref")
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

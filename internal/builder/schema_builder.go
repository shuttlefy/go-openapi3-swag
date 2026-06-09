package builder

import (
	"reflect"
	"strconv"
	"strings"

	spec3 "github.com/shuttlefy/go-openapi3-spec"
	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

// SchemaBuilder 是 Resolver 的门面，对外暴露 Build 和 Components 接口。
type SchemaBuilder struct {
	resolver *Resolver
}

// NewSchemaBuilder 创建 SchemaBuilder 并将自身注入 Resolver（双向引用）。
func NewSchemaBuilder(resolver *Resolver) *SchemaBuilder {
	sb := &SchemaBuilder{resolver: resolver}
	resolver.sb = sb
	return sb
}

// Build 将类型字符串解析为 spec3.Schema。
func (sb *SchemaBuilder) Build(typeStr string, file *parser.RawFile) *spec3.Schema {
	return sb.resolver.Resolve(typeStr, file)
}

// Components 返回已累积构建的 spec3.Components。
func (sb *SchemaBuilder) Components() *spec3.Components {
	return sb.resolver.Components()
}

// ── Struct schema building ────────────────────────────────────────────────────

// buildStructSchema 从 RawStruct 构建 spec3.Schema（无类型参数替换）。
func (sb *SchemaBuilder) buildStructSchema(s parser.RawStruct, file *parser.RawFile) *spec3.Schema {
	return sb.buildStructSchemaWithSubst(s, file, nil)
}

// paramKind 标识 struct 字段所在的 OpenAPI 参数位置，决定使用哪个 Go tag 解析字段名。
type paramKind string

const (
	kindBody     paramKind = "body"
	kindQuery    paramKind = "query"
	kindPath     paramKind = "path"
	kindHeader   paramKind = "header"
	kindFormData paramKind = "formdata"
)

// nameTagsFor 返回该位置下命名 tag 的优先级列表。
// 排第一的是该位置框架（如 gin）真正用于绑定的 tag；
// json 作为兜底，兼容仅写了 json tag 的旧代码。
func nameTagsFor(kind paramKind) []string {
	switch kind {
	case kindQuery, kindFormData:
		return []string{"form", "json"}
	case kindPath:
		return []string{"uri", "json"}
	case kindHeader:
		return []string{"header", "json"}
	default: // body / 未指定
		return []string{"json"}
	}
}

// buildStructSchemaWithSubst 构建 struct schema，argMap 提供泛型类型参数的替换关系。
func (sb *SchemaBuilder) buildStructSchemaWithSubst(s parser.RawStruct, file *parser.RawFile, argMap map[string]string) *spec3.Schema {
	schema := &spec3.Schema{Type: "object"}
	for _, c := range s.Comments {
		if c != "" {
			schema.Description = c
			break
		}
	}

	props := spec3.NewOrderedSchemas()
	var required []string
	var allOf []*spec3.Schema

	for _, field := range s.Fields {
		if field.Embedded {
			// 嵌入字段：通过 allOf 引用
			typeName := strings.TrimPrefix(field.TypeName, "*")
			if es := sb.resolver.Resolve(typeName, file); es != nil {
				allOf = append(allOf, es)
			}
			continue
		}

		propName, fieldSchema, isRequired, skip := sb.buildFieldSchema(field, file, argMap, kindBody)
		if skip || fieldSchema == nil {
			continue
		}
		props.Set(propName, fieldSchema)
		if isRequired {
			required = append(required, propName)
		}
	}

	if len(allOf) > 0 {
		schema.AllOf = allOf
	}
	if len(props.Keys()) > 0 {
		schema.Properties = &props
	}
	if len(required) > 0 {
		schema.Required = required
	}
	return schema
}

// buildFieldSchema 将 RawField 解析为 (name, schema, required, skip)。
// kind 决定使用哪个 tag 作为字段名来源（json/form/uri/header）。
func (sb *SchemaBuilder) buildFieldSchema(
	field parser.RawField,
	file *parser.RawFile,
	argMap map[string]string,
	kind paramKind,
) (string, *spec3.Schema, bool, bool) {
	info := parseStructTags(field.Tag, field.Name, kind)
	if info.skip {
		return "", nil, false, true
	}

	// 泛型类型参数替换。
	// 当发生替换时（TypeName 是类型参数名），具体类型字符串来自调用方注解上下文，
	// 该上下文的 import 已在调用方的 file 中验证过，无需再走 struct 所在文件的 import 检查。
	// 使用 nil file（宽松模式）以避免 struct 文件缺少对应 import 的误报。
	typeStr := field.TypeName
	resolveFile := file
	if argMap != nil {
		if newTypeStr := substituteTypeParam(typeStr, argMap); newTypeStr != typeStr {
			typeStr = newTypeStr
			resolveFile = nil // 已替换为 caller 提供的具体类型，用宽松模式解析
		} else {
			typeStr = newTypeStr
		}
	}

	fs := sb.resolver.Resolve(typeStr, resolveFile)
	if fs == nil {
		return "", nil, false, true
	}

	// clone 避免污染已注册的 schema
	s := shallowCloneSchema(fs)

	if info.description != "" {
		s.Description = info.description
	} else if len(field.Comments) > 0 {
		var nonEmpty []string
		for _, c := range field.Comments {
			if c != "" {
				nonEmpty = append(nonEmpty, c)
			}
		}
		if len(nonEmpty) > 0 {
			s.Description = strings.Join(nonEmpty, " ")
		}
	}

	applyTagConstraints(s, info)
	return info.name, s, info.required, false
}

// substituteTypeParam 在类型字符串中替换类型参数。
func substituteTypeParam(typeStr string, argMap map[string]string) string {
	if concrete, ok := argMap[typeStr]; ok {
		return concrete
	}
	if strings.HasPrefix(typeStr, "[]") {
		if concrete, ok := argMap[typeStr[2:]]; ok {
			return "[]" + concrete
		}
	}
	if strings.HasPrefix(typeStr, "*") {
		if concrete, ok := argMap[typeStr[1:]]; ok {
			return "*" + concrete
		}
	}
	return typeStr
}

// shallowCloneSchema 浅拷贝 schema（避免直接修改共享的已注册 schema）。
func shallowCloneSchema(s *spec3.Schema) *spec3.Schema {
	cp := *s
	return &cp
}

// ── Struct tag parsing ────────────────────────────────────────────────────────

type fieldTagInfo struct {
	name        string // 字段在 OpenAPI 中暴露的名字（按位置选择 json/form/uri/header）
	omitempty   bool
	skip        bool
	required    bool
	description string
	example     string
	enum        []string
	format      string
	defaultVal  string
	readonly    bool
	writeonly   bool
	deprecated  bool
	minimum     *float64
	maximum     *float64
	minLength   *int64
	maxLength   *int64
	pattern     string
	minItems    *int64
	maxItems    *int64
	uniqueItems bool
}

// parseStructTags 解析 Go struct tag 字符串，提取 OpenAPI 相关约束。
// kind 决定字段命名 tag 的优先级；缺省按 body（json）处理。
func parseStructTags(rawTag, fieldName string, kind paramKind) fieldTagInfo {
	info := fieldTagInfo{}
	tag := reflect.StructTag(rawTag)

	// 命名 tag：按位置优先级查找，第一个出现的 tag 决定字段名与 skip 语义。
	// 这与 gin 等框架行为一致（query 只看 form，body 只看 json），同时 json
	// 作为兜底允许仅写 json tag 的旧 struct 在 query/path 等位置仍能正确导出。
	for _, key := range nameTagsFor(kind) {
		v, ok := tag.Lookup(key)
		if !ok {
			continue
		}
		if v == "-" {
			info.skip = true
			return info
		}
		parts := strings.SplitN(v, ",", 2)
		if parts[0] != "" {
			info.name = parts[0]
		}
		if len(parts) > 1 && strings.Contains(parts[1], "omitempty") {
			info.omitempty = true
		}
		break
	}
	if info.name == "" {
		info.name = fieldName
	}

	// required
	if tagContains(tag.Get("binding"), "required") || tagContains(tag.Get("validate"), "required") {
		info.required = true
	}

	info.description = tag.Get("description")
	info.example = tag.Get("example")
	info.format = tag.Get("format")
	info.defaultVal = tag.Get("default")
	info.readonly = tag.Get("readonly") == "true"
	info.writeonly = tag.Get("writeonly") == "true"
	info.deprecated = tag.Get("deprecated") == "true"
	info.pattern = tag.Get("pattern")
	info.uniqueItems = tag.Get("uniqueItems") == "true"

	if s := tag.Get("enums"); s != "" {
		info.enum = strings.Split(s, ",")
	}

	if v, err := strconv.ParseFloat(tag.Get("minimum"), 64); err == nil {
		info.minimum = &v
	}
	if v, err := strconv.ParseFloat(tag.Get("maximum"), 64); err == nil {
		info.maximum = &v
	}
	if v, err := strconv.ParseInt(tag.Get("minLength"), 10, 64); err == nil {
		info.minLength = &v
	}
	if v, err := strconv.ParseInt(tag.Get("maxLength"), 10, 64); err == nil {
		info.maxLength = &v
	}
	if v, err := strconv.ParseInt(tag.Get("minItems"), 10, 64); err == nil {
		info.minItems = &v
	}
	if v, err := strconv.ParseInt(tag.Get("maxItems"), 10, 64); err == nil {
		info.maxItems = &v
	}

	return info
}

func tagContains(tagVal, token string) bool {
	for _, part := range strings.Split(tagVal, ",") {
		if strings.TrimSpace(part) == token {
			return true
		}
	}
	return false
}

// applyTagConstraints 将 fieldTagInfo 中的约束写入 spec3.Schema。
func applyTagConstraints(s *spec3.Schema, info fieldTagInfo) {
	if info.format != "" && s.Format == "" {
		s.Format = info.format
	}
	if info.example != "" {
		s.Example = info.example
	}
	if info.defaultVal != "" {
		s.Default = info.defaultVal
	}
	if len(info.enum) > 0 {
		s.Enum = make([]interface{}, len(info.enum))
		for i, v := range info.enum {
			s.Enum[i] = v
		}
	}
	if info.readonly {
		s.ReadOnly = true
	}
	if info.writeonly {
		s.WriteOnly = true
	}
	if info.deprecated {
		s.Deprecated = true
	}
	if info.minimum != nil {
		s.Minimum = info.minimum
	}
	if info.maximum != nil {
		s.Maximum = info.maximum
	}
	if info.minLength != nil {
		s.MinLength = info.minLength
	}
	if info.maxLength != nil {
		s.MaxLength = info.maxLength
	}
	if info.pattern != "" {
		s.Pattern = info.pattern
	}
	if info.minItems != nil {
		s.MinItems = info.minItems
	}
	if info.maxItems != nil {
		s.MaxItems = info.maxItems
	}
	if info.uniqueItems {
		s.UniqueItems = true
	}
}

package builder

import (
	"sort"
	"strconv"
	"strings"

	spec "github.com/shuttlefy/go-openapi3-spec"
	"github.com/shuttlefy/go-openapi3-swag/internal/extractor"
	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

// SchemaBuilder converts RawStruct definitions into spec3 Schema objects and
// manages the collected schemas for components.
type SchemaBuilder struct {
	resolver     *Resolver
	structs      map[string]*parser.RawStruct    // name → struct
	aliases      map[string]*parser.RawTypeAlias // name → type alias
	constsByType map[string][]parser.RawConst    // aliasName → consts
	schemas      *spec.OrderedSchemas            // built schemas
	building     map[string]bool                 // cycle detection
	unknownTypes map[string]bool                 // unregistered type names encountered
}

func NewSchemaBuilder(resolver *Resolver) *SchemaBuilder {
	schemas := spec.NewOrderedSchemas()
	return &SchemaBuilder{
		resolver:     resolver,
		structs:      make(map[string]*parser.RawStruct),
		aliases:      make(map[string]*parser.RawTypeAlias),
		constsByType: make(map[string][]parser.RawConst),
		schemas:      &schemas,
		building:     make(map[string]bool),
		unknownTypes: make(map[string]bool),
	}
}

// UnknownTypeNames returns sorted names of types referenced but never registered.
func (sb *SchemaBuilder) UnknownTypeNames() []string {
	names := make([]string, 0, len(sb.unknownTypes))
	for n := range sb.unknownTypes {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// RegisterStruct registers a raw struct for later schema building.
func (sb *SchemaBuilder) RegisterStruct(s *parser.RawStruct) {
	sb.structs[s.Name] = s
	sb.resolver.Register(s.Name)
}

// RegisterTypeAlias registers a type alias (e.g. type Status string) so that
// fields using it receive a $ref and a named schema is emitted in components.
func (sb *SchemaBuilder) RegisterTypeAlias(a *parser.RawTypeAlias) {
	sb.aliases[a.Name] = a
	sb.resolver.Register(a.Name)
}

// RegisterConst records a typed constant so its value can be added as an enum
// entry on the corresponding alias schema.
func (sb *SchemaBuilder) RegisterConst(c parser.RawConst) {
	if c.TypeName != "" {
		sb.constsByType[c.TypeName] = append(sb.constsByType[c.TypeName], c)
	}
}

// BuildAll builds schemas for all registered structs and type aliases.
func (sb *SchemaBuilder) BuildAll() {
	for name := range sb.structs {
		sb.getOrBuild(name)
	}
	for name := range sb.aliases {
		sb.getOrBuild(name)
	}
}

// Schemas returns the built OrderedSchemas for use in Components.
func (sb *SchemaBuilder) Schemas() *spec.OrderedSchemas {
	return sb.schemas
}

// SchemaForType returns the spec3 Schema for a Go type expression string.
// For registered struct names it returns a $ref; for primitives it returns inline schemas.
func (sb *SchemaBuilder) SchemaForType(typeName string) *spec.Schema {
	return sb.goTypeToSchema(typeName)
}

// SchemaForTypeExpr builds a schema from an extractor.TypeExpr, handling composite overrides.
func (sb *SchemaBuilder) SchemaForTypeExpr(te extractor.TypeExpr, isArray bool) *spec.Schema {
	var inner *spec.Schema
	if len(te.Overrides) == 0 {
		inner = sb.goTypeToSchema(te.Name)
	} else {
		inner = sb.buildCompositeSchema(te)
	}

	if isArray {
		return &spec.Schema{
			Type:  "array",
			Items: inner,
		}
	}
	return inner
}

func (sb *SchemaBuilder) getOrBuild(name string) *spec.Schema {
	if existing := sb.schemas.Get(name); existing != nil {
		return existing
	}
	if raw, ok := sb.structs[name]; ok {
		if sb.building[name] {
			return sb.resolver.RefSchema(name)
		}
		sb.building[name] = true
		schema := sb.buildStruct(raw)
		delete(sb.building, name)
		sb.schemas.Set(name, schema)
		return schema
	}
	if alias, ok := sb.aliases[name]; ok {
		schema := sb.buildAliasSchema(alias)
		sb.schemas.Set(name, schema)
		return schema
	}
	return nil
}

func (sb *SchemaBuilder) buildStruct(raw *parser.RawStruct) *spec.Schema {
	props := spec.NewOrderedSchemas()
	var required []string
	var embedRefs []*spec.Schema

	for _, f := range raw.Fields {
		// Anonymous embedded field (no field name in source).
		if f.Name == "" {
			if f.JSONName == "-" {
				continue
			}
			// Tagged embedded field: treat as a named property.
			if f.JSONName != "" {
				propSchema := sb.buildFieldSchema(f)
				props.Set(f.JSONName, propSchema)
				if f.Required {
					required = append(required, f.JSONName)
				}
				continue
			}
			// True struct embedding: collect for allOf expansion.
			baseType := strings.TrimPrefix(f.TypeName, "*")
			if baseType != "" {
				embedRefs = append(embedRefs, sb.goTypeToSchema(baseType))
			}
			continue
		}

		propName := fieldName(f)
		if propName == "-" {
			continue
		}
		propSchema := sb.buildFieldSchema(f)
		props.Set(propName, propSchema)
		if f.Required {
			required = append(required, propName)
		}
	}

	ownSchema := &spec.Schema{
		Type:        "object",
		Properties:  &props,
		Description: commentsToDescription(raw.Comments),
	}
	if len(required) > 0 {
		ownSchema.Required = required
	}

	if len(embedRefs) == 0 {
		return ownSchema
	}

	// Produce allOf: [embed refs...] + own schema (if non-empty).
	allOf := make([]*spec.Schema, 0, len(embedRefs)+1)
	allOf = append(allOf, embedRefs...)
	if len(props.Keys()) > 0 || len(required) > 0 {
		allOf = append(allOf, ownSchema)
	}
	out := &spec.Schema{AllOf: allOf}
	if desc := commentsToDescription(raw.Comments); desc != "" {
		out.Description = desc
	}
	return out
}

func (sb *SchemaBuilder) buildFieldSchema(f parser.RawField) *spec.Schema {
	schema := sb.goTypeToSchema(f.TypeName)

	// description tag takes priority; fall back to Go doc comment on the field.
	desc := f.Description
	if desc == "" {
		desc = commentsToDescription(f.Comments)
	}
	if desc != "" {
		schema.Description = desc
	}
	if f.Example != "" {
		schema.Example = f.Example
	}
	if len(f.Enums) > 0 {
		enums := make([]interface{}, len(f.Enums))
		for i, e := range f.Enums {
			enums[i] = e
		}
		schema.Enum = enums
	}
	if f.Format != "" {
		schema.Format = f.Format
	}
	if f.Default != "" {
		schema.Default = f.Default
	}
	if f.ReadOnly {
		schema.ReadOnly = true
	}
	if f.WriteOnly {
		schema.WriteOnly = true
	}
	if f.Deprecated {
		schema.Deprecated = true
	}
	if f.Pattern != "" {
		schema.Pattern = f.Pattern
	}
	schema.Minimum = f.Minimum
	schema.Maximum = f.Maximum
	schema.MinLength = f.MinLength
	schema.MaxLength = f.MaxLength
	schema.MinItems = f.MinItems
	schema.MaxItems = f.MaxItems
	if f.UniqueItems {
		schema.UniqueItems = true
	}

	return schema
}

func (sb *SchemaBuilder) goTypeToSchema(typeName string) *spec.Schema {
	if strings.HasPrefix(typeName, "*") {
		inner := sb.goTypeToSchema(typeName[1:])
		inner.Nullable = true
		return inner
	}

	if strings.HasPrefix(typeName, "[]") {
		items := sb.goTypeToSchema(typeName[2:])
		return &spec.Schema{
			Type:  "array",
			Items: items,
		}
	}

	if strings.HasPrefix(typeName, "map[") {
		valueType := extractMapValueType(typeName)
		valSchema := sb.goTypeToSchema(valueType)
		return &spec.Schema{
			Type:                 "object",
			AdditionalProperties: valSchema,
		}
	}

	// interface{} / any → empty schema (accepts any JSON value).
	if typeName == "interface{}" || typeName == "any" {
		return &spec.Schema{}
	}

	if t, f, ok := primitiveType(typeName); ok {
		s := &spec.Schema{Type: t}
		if f != "" {
			s.Format = f
		}
		return s
	}

	if sb.resolver.IsRegistered(typeName) {
		sb.getOrBuild(typeName)
		return sb.resolver.RefSchema(typeName)
	}

	// Record unresolved reference for diagnostic reporting.
	sb.unknownTypes[typeName] = true
	return &spec.Schema{Type: "object"}
}

// buildCompositeSchema builds an allOf schema from a TypeExpr with field overrides.
//
// Simple override:
//
//	PageData{data=[]User}
//	→ allOf: [$ref PageData, {properties: {data: {type: array, items: $ref User}}}]
//
// Nested composite override:
//
//	BaseResponse{data=PagedList{list=[]Pet}}
//	→ allOf: [$ref BaseResponse, {properties: {data: allOf[$ref PagedList, {properties:{list:[...]}}]}}]
//
// Each override value is parsed with ParseTypeExpr so that composite expressions
// (e.g. "PagedList{list=[]Pet}") are handled recursively, not treated as plain type names.
func (sb *SchemaBuilder) buildCompositeSchema(te extractor.TypeExpr) *spec.Schema {
	baseRef := sb.goTypeToSchema(te.Name)

	overrideProps := spec.NewOrderedSchemas()
	for _, ov := range te.Overrides {
		// Recursively parse the override value — it may itself be a composite
		// type expression (e.g. "PagedList{list=[]Pet}"), an array ("[]Pet"),
		// a map ("map[string]Pet"), or a simple ref ("Pet").
		ovExpr := extractor.ParseTypeExpr(ov.TypeExpr)
		overrideProps.Set(ov.Field, sb.SchemaForTypeExpr(ovExpr, false))
	}
	overrideSchema := &spec.Schema{
		Type:       "object",
		Properties: &overrideProps,
	}

	return &spec.Schema{
		AllOf: []*spec.Schema{baseRef, overrideSchema},
	}
}

func fieldName(f parser.RawField) string {
	if f.JSONName != "" {
		return f.JSONName
	}
	return f.Name
}

var goTypePrimitives = map[string][2]string{
	"string":    {"string", ""},
	"bool":      {"boolean", ""},
	"int":       {"integer", "int32"},
	"int8":      {"integer", "int32"},
	"int16":     {"integer", "int32"},
	"int32":     {"integer", "int32"},
	"int64":     {"integer", "int64"},
	"uint":      {"integer", "int32"},
	"uint8":     {"integer", "int32"},
	"uint16":    {"integer", "int32"},
	"uint32":    {"integer", "int32"},
	"uint64":    {"integer", "int64"},
	"float32":   {"number", "float"},
	"float64":   {"number", "double"},
	"byte":      {"string", "byte"},
	"time.Time": {"string", "date-time"},
}

func primitiveType(typeName string) (string, string, bool) {
	if p, ok := goTypePrimitives[typeName]; ok {
		return p[0], p[1], true
	}
	return "", "", false
}

// paramTypePrimitives maps annotation type names to OpenAPI type+format.
var paramTypePrimitives = map[string][2]string{
	"string":  {"string", ""},
	"integer": {"integer", "int64"},
	"int":     {"integer", "int32"},
	"int64":   {"integer", "int64"},
	"number":  {"number", "double"},
	"boolean": {"boolean", ""},
	"bool":    {"boolean", ""},
	"file":    {"string", "binary"},
}

// ParamTypeSchema returns a schema for an annotation param type name.
func ParamTypeSchema(typeName, format string) spec.Schema {
	if p, ok := paramTypePrimitives[typeName]; ok {
		s := spec.Schema{Type: p[0]}
		if format != "" {
			s.Format = format
		} else if p[1] != "" {
			s.Format = p[1]
		}
		return s
	}
	return spec.Schema{Type: "string"}
}

func extractMapValueType(typeName string) string {
	depth := 0
	for i, ch := range typeName {
		if ch == '[' {
			depth++
		} else if ch == ']' {
			depth--
			if depth == 0 {
				return typeName[i+1:]
			}
		}
	}
	return "string"
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// toEnumInterfaces converts string enums to []interface{} for the spec.
func toEnumInterfaces(enums []string) []interface{} {
	if len(enums) == 0 {
		return nil
	}
	out := make([]interface{}, len(enums))
	for i, e := range enums {
		out[i] = e
	}
	return out
}

// tryParseDefault attempts to parse a default value string into an appropriate type.
func tryParseDefault(val string) interface{} {
	if val == "" {
		return nil
	}
	if i, err := strconv.ParseInt(val, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return f
	}
	if b, err := strconv.ParseBool(val); err == nil {
		return b
	}
	return val
}

// buildAliasSchema builds an OpenAPI schema for a named type alias, e.g.
//
//	type Status string
//
// If typed constants are registered for the alias they are emitted as:
//
//   - enum          – the raw constant values (["available", "pending", …])
//   - x-enum-varnames     – the Go identifier names (["StatusAvailable", …])
//   - x-enumDescriptions  – per-value descriptions from the Go doc comments
//     (Redoc / Scalar / Stoplight all render this field)
//
// x-enumDescriptions is only added when at least one constant has a comment.
func (sb *SchemaBuilder) buildAliasSchema(a *parser.RawTypeAlias) *spec.Schema {
	schema := sb.goTypeToSchema(a.Underlying)

	if consts, ok := sb.constsByType[a.Name]; ok && len(consts) > 0 {
		enums := make([]interface{}, 0, len(consts))
		varnames := make([]interface{}, 0, len(consts))
		descriptions := make([]interface{}, 0, len(consts))
		hasDesc := false

		for _, c := range consts {
			enums = append(enums, unquoteConstValue(c.Value))
			varnames = append(varnames, c.Name)
			desc := commentsToDescription(c.Comments)
			descriptions = append(descriptions, desc)
			if desc != "" {
				hasDesc = true
			}
		}

		schema.Enum = enums
		schema.Extensions.Set("x-enum-varnames", varnames)
		if hasDesc {
			schema.Extensions.Set("x-enumDescriptions", descriptions)
		}
	}

	if desc := commentsToDescription(a.Comments); desc != "" {
		schema.Description = desc
	}
	return schema
}

// commentsToDescription converts a slice of raw Go comment strings (e.g.
// ["// Pet is an animal.", "// It lives in the store."]) to a plain-text
// description suitable for an OpenAPI description field.
func commentsToDescription(comments []string) string {
	var lines []string
	for _, c := range comments {
		c = strings.TrimSpace(c)
		switch {
		case strings.HasPrefix(c, "// "):
			lines = append(lines, c[3:])
		case strings.HasPrefix(c, "//"):
			lines = append(lines, c[2:])
		case strings.HasPrefix(c, "/*"):
			c = strings.TrimPrefix(c, "/*")
			c = strings.TrimSuffix(c, "*/")
			c = strings.TrimSpace(c)
			if c != "" {
				lines = append(lines, c)
			}
		}
	}
	return strings.Join(lines, " ")
}

// unquoteConstValue converts a raw Go const literal string to its Go value.
// String literals are unquoted; integer and float literals are parsed.
func unquoteConstValue(v string) interface{} {
	if strings.HasPrefix(v, `"`) || strings.HasPrefix(v, "`") {
		if s, err := strconv.Unquote(v); err == nil {
			return s
		}
	}
	if i, err := strconv.ParseInt(v, 0, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}
	return v
}

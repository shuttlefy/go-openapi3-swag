package builder

import (
	"strings"

	spec3 "github.com/shuttlefy/go-openapi3-spec"
	"github.com/shuttlefy/go-openapi3-swag/internal/extractor"
	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

// OperationBuilder 将 OperationAnnotation 转换为 spec3.Operation。
type OperationBuilder struct {
	schema *SchemaBuilder
}

func NewOperationBuilder(schema *SchemaBuilder) *OperationBuilder {
	return &OperationBuilder{schema: schema}
}

// Build 构建单个 spec3.Operation。
// fileIndex 是 filePath → *parser.RawFile 的索引，用于解析注解中的类型引用。
func (ob *OperationBuilder) Build(
	op extractor.OperationAnnotation,
	fileIndex map[string]*parser.RawFile,
) (*spec3.Operation, error) {
	file := fileIndex[op.FilePath]

	oper := &spec3.Operation{
		OperationID: op.OperationID,
		Tags:        op.Tags,
		Deprecated:  op.Deprecated,
		Responses:   spec3.NewOrderedResponses(),
	}

	oper.Summary = strPtr(op.Summary)
	oper.Description = strPtr(op.Description)

	// 参数（非 body / formData）
	for _, p := range op.Params {
		if inBody(p.In) {
			continue
		}
		oper.Parameters = append(oper.Parameters, ob.buildParam(p, file))
	}

	// 请求体
	if rb := ob.buildRequestBody(op, file); rb != nil {
		oper.RequestBody = *rb
	}

	// 响应
	for _, resp := range op.Responses {
		oper.Responses.Set(resp.Code, ob.buildResponse(resp, op.Headers, file))
	}

	// 安全要求
	for _, sec := range op.Security {
		var sr spec3.SecurityRequirement
		sr.Set(sec.Name, sec.Scopes...)
		oper.Security = append(oper.Security, sr)
	}

	return oper, nil
}

func inBody(in string) bool {
	s := strings.ToLower(in)
	return s == "body" || s == "formdata"
}

// ── Parameter ─────────────────────────────────────────────────────────────────

func (ob *OperationBuilder) buildParam(p extractor.ParamAnnotation, file *parser.RawFile) spec3.Parameter {
	param := spec3.Parameter{
		Name:        p.Name,
		In:          strings.ToLower(p.In),
		Required:    p.Required,
		Description: strPtr(p.Description),
	}

	if schema := ob.resolveParamSchema(p.TypeName, p.Format, file); schema != nil {
		param.Schema = *schema
	}
	return param
}

func (ob *OperationBuilder) resolveParamSchema(typeName, format string, file *parser.RawFile) *spec3.Schema {
	s := ob.schema.Build(normalizePrimitive(typeName), file)
	if s == nil {
		return nil
	}
	if format != "" {
		cp := *s
		cp.Format = format
		return &cp
	}
	return s
}

// normalizePrimitive 将注解中的原始类型别名统一为内部类型名。
func normalizePrimitive(t string) string {
	switch strings.ToLower(t) {
	case "integer":
		return "integer"
	case "number":
		return "number"
	case "boolean":
		return "bool"
	case "file":
		return "file"
	}
	return t
}

// ── Request Body ──────────────────────────────────────────────────────────────

func (ob *OperationBuilder) buildRequestBody(op extractor.OperationAnnotation, file *parser.RawFile) *spec3.RequestBody {
	var bodyParams, formParams []extractor.ParamAnnotation
	for _, p := range op.Params {
		switch strings.ToLower(p.In) {
		case "body":
			bodyParams = append(bodyParams, p)
		case "formdata":
			formParams = append(formParams, p)
		}
	}
	if len(bodyParams) == 0 && len(formParams) == 0 {
		return nil
	}

	content := spec3.NewOrderedMediaTypes()

	if len(bodyParams) > 0 {
		p := bodyParams[0]
		ct := firstMIME(op.Accept, "application/json")
		mt := &spec3.MediaType{Schema: ob.schema.Build(p.TypeName, file)}
		content.Set(ct, mt)
	}

	if len(formParams) > 0 {
		s := buildFormDataSchema(formParams)
		content.Set("multipart/form-data", &spec3.MediaType{Schema: &s})
	}

	rb := &spec3.RequestBody{Required: true, Content: content}
	return rb
}

func firstMIME(accept []string, fallback string) string {
	if len(accept) > 0 {
		return accept[0]
	}
	return fallback
}

func buildFormDataSchema(params []extractor.ParamAnnotation) spec3.Schema {
	props := spec3.NewOrderedSchemas()
	var required []string

	for _, p := range params {
		var s *spec3.Schema
		switch strings.ToLower(p.TypeName) {
		case "file":
			s = &spec3.Schema{Type: "string", Format: "binary"}
		default:
			if ps, ok := primitiveSchema(normalizePrimitive(p.TypeName)); ok {
				s = ps
			} else {
				s = &spec3.Schema{Type: "string"}
			}
		}
		if p.Description != "" {
			s.Description = p.Description
		}
		props.Set(p.Name, s)
		if p.Required {
			required = append(required, p.Name)
		}
	}

	schema := spec3.Schema{Type: "object", Properties: &props}
	if len(required) > 0 {
		schema.Required = required
	}
	return schema
}

// ── Response ──────────────────────────────────────────────────────────────────

func (ob *OperationBuilder) buildResponse(
	resp extractor.ResponseAnnotation,
	headers []extractor.HeaderAnnotation,
	file *parser.RawFile,
) *spec3.Response {
	r := &spec3.Response{Description: strPtr(resp.Description)}

	if resp.TypeName != "" || resp.WrapType != "" {
		if schema := ob.buildResponseSchema(resp, file); schema != nil {
			ct := spec3.NewOrderedMediaTypes()
			ct.Set("application/json", &spec3.MediaType{Schema: schema})
			r.Content = &ct
		}
	}

	// response headers（匹配状态码）
	for _, h := range headers {
		if h.Code != resp.Code {
			continue
		}
		oh := spec3.NewOrderedHeaders()
		hHeader := &spec3.Header{Parameter: spec3.Parameter{
			Description: strPtr(h.Description),
			Schema:      spec3.Schema{Type: h.TypeName},
		}}
		oh.Set(h.Name, hHeader)
		r.Headers = &oh
	}

	return r
}

func (ob *OperationBuilder) buildResponseSchema(resp extractor.ResponseAnnotation, file *parser.RawFile) *spec3.Schema {
	if resp.TypeName == "" {
		// 纯原始类型包装，如 {string}
		if s, ok := primitiveSchema(resp.WrapType); ok {
			return s
		}
		return nil
	}
	return ob.schema.Build(resp.TypeName, file)
}

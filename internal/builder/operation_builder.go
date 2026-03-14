package builder

import (
	spec "github.com/shuttlefy/go-openapi3-spec"
	"github.com/shuttlefy/go-openapi3-swag/internal/extractor"
)

// OperationBuilder converts OperationAnnotation into spec3 Operation objects.
type OperationBuilder struct {
	schema *SchemaBuilder
}

func NewOperationBuilder(schema *SchemaBuilder) *OperationBuilder {
	return &OperationBuilder{schema: schema}
}

func (ob *OperationBuilder) Build(anno extractor.OperationAnnotation) *spec.Operation {
	op := &spec.Operation{
		OperationID: anno.OperationID,
		Summary:     strPtr(anno.Summary),
		Description: strPtr(anno.Description),
		Tags:        anno.Tags,
		Deprecated:  anno.Deprecated,
	}

	op.Parameters = ob.buildParameters(anno.Params)
	op.RequestBody = ob.buildRequestBody(anno)
	op.Responses = ob.buildResponses(anno)
	op.Security = ob.buildSecurity(anno.Security)

	return op
}

func (ob *OperationBuilder) buildParameters(params []extractor.ParamAnnotation) []spec.Parameter {
	if len(params) == 0 {
		return nil
	}

	var out []spec.Parameter
	for _, p := range params {
		schema := ParamTypeSchema(p.TypeName, p.Format)
		if p.Default != "" {
			schema.Default = tryParseDefault(p.Default)
		}
		if len(p.Enums) > 0 {
			schema.Enum = toEnumInterfaces(p.Enums)
		}

		param := spec.Parameter{
			Name:        p.Name,
			In:          p.In,
			Description: strPtr(p.Description),
			Required:    p.Required,
			Schema:      schema,
		}
		out = append(out, param)
	}
	return out
}

func (ob *OperationBuilder) buildRequestBody(anno extractor.OperationAnnotation) spec.RequestBody {
	rb := anno.RequestBody
	if rb == nil {
		return spec.RequestBody{}
	}

	content := spec.NewOrderedMediaTypes()

	if rb.IsForm {
		formSchema := ob.buildFormSchema(rb)
		contentTypes := anno.Accept
		if len(contentTypes) == 0 {
			contentTypes = []string{"multipart/form-data"}
		}
		for _, ct := range contentTypes {
			content.Set(ct, &spec.MediaType{Schema: formSchema})
		}
	} else {
		bodySchema := ob.schema.SchemaForTypeExpr(rb.Type, false)
		contentTypes := anno.Accept
		if len(contentTypes) == 0 {
			contentTypes = []string{"application/json"}
		}
		for _, ct := range contentTypes {
			content.Set(ct, &spec.MediaType{Schema: bodySchema})
		}
	}

	return spec.RequestBody{
		Description: strPtr(rb.Description),
		Content:     content,
		Required:    rb.Required,
	}
}

func (ob *OperationBuilder) buildFormSchema(rb *extractor.RequestBodyAnnotation) *spec.Schema {
	if len(rb.Fields) == 0 && rb.TypeName != "" {
		return ob.schema.SchemaForType(rb.TypeName)
	}

	props := spec.NewOrderedSchemas()
	var required []string
	for _, f := range rb.Fields {
		s := ParamTypeSchema(f.TypeName, "")
		props.Set(f.Name, &s)
		if f.Required {
			required = append(required, f.Name)
		}
	}

	schema := &spec.Schema{
		Type:       "object",
		Properties: &props,
	}
	if len(required) > 0 {
		schema.Required = required
	}
	return schema
}

func (ob *OperationBuilder) buildResponses(anno extractor.OperationAnnotation) spec.OrderedResponses {
	responses := spec.NewOrderedResponses()

	produce := anno.Produce
	if len(produce) == 0 {
		produce = []string{"application/json"}
	}

	for _, ra := range anno.Responses {
		resp := ob.buildSingleResponse(ra, produce)
		responses.Set(ra.Code, resp)
	}

	return responses
}

func (ob *OperationBuilder) buildSingleResponse(ra extractor.ResponseAnnotation, produce []string) *spec.Response {
	resp := &spec.Response{
		Description: strPtr(ra.Description),
	}

	hasBody := ra.TypeName != "" || ra.Type.Name != "" || ra.IsPrimitive
	if hasBody {
		content := spec.NewOrderedMediaTypes()
		var schema *spec.Schema

		if ra.IsPrimitive {
			s := primitiveAnnotationSchema(ra.TypeName)
			schema = &s
		} else if ra.Type.Name != "" {
			schema = ob.schema.SchemaForTypeExpr(ra.Type, ra.IsArray)
		} else {
			inner := ob.schema.SchemaForType(ra.TypeName)
			if ra.IsArray {
				schema = &spec.Schema{Type: "array", Items: inner}
			} else {
				schema = inner
			}
		}

		for _, ct := range produce {
			content.Set(ct, &spec.MediaType{Schema: schema})
		}
		resp.Content = &content
	}

	if len(ra.Headers) > 0 {
		headers := spec.NewOrderedHeaders()
		for _, h := range ra.Headers {
			hSchema := ParamTypeSchema(h.TypeName, "")
			headers.Set(h.Name, &spec.Header{
				Parameter: spec.Parameter{
					Description: strPtr(h.Description),
					Schema:      hSchema,
				},
			})
		}
		resp.Headers = &headers
	}

	return resp
}

func (ob *OperationBuilder) buildSecurity(reqs []extractor.SecurityRequirement) []spec.SecurityRequirement {
	if len(reqs) == 0 {
		return nil
	}
	var out []spec.SecurityRequirement
	for _, r := range reqs {
		var sr spec.SecurityRequirement
		scopes := r.Scopes
		if scopes == nil {
			scopes = []string{}
		}
		sr.Set(r.Name, scopes...)
		out = append(out, sr)
	}
	return out
}

func primitiveAnnotationSchema(typeName string) spec.Schema {
	switch typeName {
	case "string", "{string}":
		return spec.Schema{Type: "string"}
	case "integer", "{integer}":
		return spec.Schema{Type: "integer"}
	case "number", "{number}":
		return spec.Schema{Type: "number"}
	case "boolean", "{boolean}":
		return spec.Schema{Type: "boolean"}
	default:
		return spec.Schema{Type: "string"}
	}
}

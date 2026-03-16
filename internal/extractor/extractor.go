package extractor

import (
	"fmt"
	"strings"

	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

var oauthFlowCanonical = map[string]string{
	"implicit":          "implicit",
	"password":          "password",
	"clientcredentials": "clientCredentials",
	"authorizationcode": "authorizationCode",
}

type GoExtractor struct{}

func NewGoExtractor() *GoExtractor {
	return &GoExtractor{}
}

func (e *GoExtractor) Extract(ast *parser.RawAST) (*ExtractResult, error) {
	result := &ExtractResult{}

	e.extractGlobal(ast, result)
	e.extractOperations(ast, result)

	return result, nil
}

// extractGlobal scans functions for global annotations.
// Supports both Swagger 2.0 legacy tags (host/basePath/schemes) and
// OpenAPI 3 native tags (server).
func (e *GoExtractor) extractGlobal(ast *parser.RawAST, result *ExtractResult) {
	// Accumulate multi-line security definitions keyed by "type.name"
	secDefs := map[string]*SecurityDefAnnotation{}
	var secDefOrder []string

	for _, fn := range ast.Functions {
		for _, comment := range fn.Comments {
			name, value := parseTag(comment)
			switch name {
			case "title":
				result.Global.Title = value
			case "version":
				result.Global.Version = value
			case "description":
				result.Global.Description = value
			case "termsofservice":
				result.Global.TermsOfService = value
			case "contact.name":
				result.Global.Contact.Name = value
			case "contact.url":
				result.Global.Contact.URL = value
			case "contact.email":
				result.Global.Contact.Email = value
			case "license.name":
				result.Global.License.Name = value
			case "license.url":
				result.Global.License.URL = value

			// Legacy Swagger 2.0 — builder will convert to servers[]
			case "host":
				result.Global.Host = value
			case "basepath":
				result.Global.BasePath = value
			case "schemes":
				result.Global.Schemes = strings.Fields(value)

			// OpenAPI 3 native servers
			case "server":
				result.Global.Servers = append(result.Global.Servers, parseServer(value))

			case "externaldocs.url":
				e.ensureExternalDocs(result)
				result.Global.ExternalDocs.URL = value
			case "externaldocs.description":
				e.ensureExternalDocs(result)
				result.Global.ExternalDocs.Description = value

			case "tag":
				result.Global.Tags = append(result.Global.Tags, parseTagAnnotation(value))

			default:
				e.parseSecurityDef(name, value, secDefs, &secDefOrder)
			}
		}
	}

	for _, key := range secDefOrder {
		result.Global.SecurityDefs = append(result.Global.SecurityDefs, *secDefs[key])
	}
}

func (e *GoExtractor) ensureExternalDocs(result *ExtractResult) {
	if result.Global.ExternalDocs == nil {
		result.Global.ExternalDocs = &ExternalDocsAnnotation{}
	}
}

func parseTagAnnotation(value string) TagAnnotation {
	parts := strings.SplitN(value, " ", 2)
	t := TagAnnotation{Name: parts[0]}
	if len(parts) > 1 {
		t.Description = strings.Trim(parts[1], `"`)
	}
	return t
}

// parseSecurityDef handles multi-line security definition tags.
// Patterns:
//
//	@securityDefinitions.apikey ApiKeyAuth
//	@securityDefinitions.apikey.in header
//	@securityDefinitions.apikey.name X-API-Key
//	@securityDefinitions.basic BasicAuth
//	@securityDefinitions.oauth2.implicit OAuth2Implicit
//	@securityDefinitions.oauth2.implicit.authorizationUrl https://...
//	@securityDefinitions.oauth2.implicit.scope.read "Read access"
func (e *GoExtractor) parseSecurityDef(tagName, value string, defs map[string]*SecurityDefAnnotation, order *[]string) {
	if !strings.HasPrefix(tagName, "securitydefinitions.") {
		return
	}

	rest := tagName[len("securitydefinitions."):]
	parts := strings.SplitN(rest, ".", -1)
	if len(parts) == 0 {
		return
	}

	schemeType := parts[0]
	// Normalize lowercased flow types back to canonical casing
	for i, p := range parts {
		if canonical, ok := oauthFlowCanonical[p]; ok {
			parts[i] = canonical
		}
	}

	// Determine the key for this security scheme
	var key string
	switch {
	case schemeType == "oauth2" && len(parts) >= 2:
		// oauth2.implicit, oauth2.password, etc.
		flowType := parts[1]
		if len(parts) == 2 {
			// @securityDefinitions.oauth2.implicit SchemeName
			key = schemeType + "." + flowType + "." + value
			if _, ok := defs[key]; !ok {
				defs[key] = &SecurityDefAnnotation{
					Name:          value,
					Type:          "oauth2",
					OAuthFlowType: flowType,
					Scopes:        map[string]string{},
				}
				*order = append(*order, key)
			}
			return
		}
		// Sub-properties: find the scheme name — it's the last set name for this flow
		schemeName := findOAuth2SchemeName(defs, schemeType, flowType)
		if schemeName == "" {
			return
		}
		key = schemeType + "." + flowType + "." + schemeName
		sd := defs[key]
		if sd == nil {
			return
		}
		subProp := parts[2]
		switch subProp {
		case "authorizationurl":
			sd.AuthorizationURL = value
		case "tokenurl":
			sd.TokenURL = value
		case "scope":
			if len(parts) >= 4 {
				scopeName := parts[3]
				sd.Scopes[scopeName] = strings.Trim(value, `"`)
			}
		case "description":
			sd.Description = value
		}
		return

	default:
		if len(parts) == 1 {
			// @securityDefinitions.apikey SchemeName
			key = schemeType + "." + value
			if _, ok := defs[key]; !ok {
				sd := &SecurityDefAnnotation{Name: value}
				switch schemeType {
				case "apikey":
					sd.Type = "apiKey"
				case "basic":
					sd.Type = "http"
					sd.Scheme = "basic"
				case "bearer":
					sd.Type = "http"
					sd.Scheme = "bearer"
				case "openidconnect":
					sd.Type = "openIdConnect"
				}
				defs[key] = sd
				*order = append(*order, key)
			}
			return
		}
		// Sub-property: @securityDefinitions.apikey.in header
		schemeName := findSchemeName(defs, schemeType)
		if schemeName == "" {
			return
		}
		key = schemeType + "." + schemeName
		sd := defs[key]
		if sd == nil {
			return
		}
		subProp := parts[1]
		switch subProp {
		case "in":
			sd.In = value
		case "name":
			sd.FieldName = value
		case "scheme":
			sd.Scheme = value
		case "bearerformat":
			sd.BearerFormat = value
		case "description":
			sd.Description = value
		case "openidconnecturl":
			sd.OpenIDConnectURL = value
		}
	}
}

func findSchemeName(defs map[string]*SecurityDefAnnotation, schemeType string) string {
	for k, v := range defs {
		if strings.HasPrefix(k, schemeType+".") && v != nil {
			return v.Name
		}
	}
	return ""
}

func findOAuth2SchemeName(defs map[string]*SecurityDefAnnotation, schemeType, flowType string) string {
	prefix := schemeType + "." + flowType + "."
	for k, v := range defs {
		if strings.HasPrefix(k, prefix) && v != nil {
			return v.Name
		}
	}
	return ""
}

func (e *GoExtractor) extractOperations(ast *parser.RawAST, result *ExtractResult) {
	for _, fn := range ast.Functions {
		op := e.parseOperation(fn)
		if op == nil {
			if hasSwagAnnotations(fn) {
				if fn.Name == "main" {
					continue
				}
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					Level:    DiagWarn,
					FilePath: fn.FilePath,
					Line:     fn.Line,
					Message:  fmt.Sprintf("function %q has swagger annotations but no @Router tag", fn.Name),
				})
			}
			continue
		}
		result.Operations = append(result.Operations, *op)
	}
}

// hasSwagAnnotations reports whether fn has any operation-level swagger tags.
func hasSwagAnnotations(fn parser.RawFunc) bool {
	for _, c := range fn.Comments {
		name, _ := parseTag(c)
		switch name {
		case "summary", "description", "tags", "param", "success", "failure",
			"id", "accept", "produce", "deprecated", "header", "security":
			return true
		}
	}
	return false
}

// parseOperation builds an OperationAnnotation from a function's comments.
// Returns nil if the function has no @Router annotation.
func (e *GoExtractor) parseOperation(fn parser.RawFunc) *OperationAnnotation {
	op := &OperationAnnotation{
		FuncName: fn.Name,
		FilePath: fn.FilePath,
		Line:     fn.Line,
	}

	hasRouter := false
	var descLines []string
	var bodyParams []ParamAnnotation
	var formParams []ParamAnnotation
	var headers []ResponseHeaderAnnotation

	for _, comment := range fn.Comments {
		name, value := parseTag(comment)
		switch name {
		case "summary":
			op.Summary = value
		case "description":
			op.Description = value
		case "id":
			op.OperationID = value
		case "tags":
			for _, t := range strings.Split(value, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					op.Tags = append(op.Tags, t)
				}
			}
		case "accept":
			op.Accept = parseMimeTypes(value)
		case "produce":
			op.Produce = parseMimeTypes(value)
		case "router":
			op.Route = parseRouter(value)
			hasRouter = true
		case "param":
			p := parseParam(value)
			switch p.In {
			case "body":
				bodyParams = append(bodyParams, p)
			case "formdata", "formData":
				p.In = "formData"
				formParams = append(formParams, p)
			default:
				op.Params = append(op.Params, p)
			}
		case "success", "failure":
			op.Responses = append(op.Responses, parseResponse(value))
		case "header":
			headers = append(headers, parseHeader(value))
		case "security":
			op.Security = append(op.Security, parseSecurity(value))
		case "deprecated":
			op.Deprecated = true
		case "":
			line := strings.TrimSpace(comment)
			line = strings.TrimPrefix(line, "//")
			line = strings.TrimSpace(line)
			if line != "" {
				descLines = append(descLines, line)
			}
		}
	}

	if !hasRouter {
		return nil
	}

	if op.Description == "" && len(descLines) > 0 {
		op.Description = strings.Join(descLines, " ")
	}

	// Convert body/formData params into RequestBody
	if len(bodyParams) > 0 {
		bp := bodyParams[0]
		op.RequestBody = &RequestBodyAnnotation{
			TypeName:    bp.TypeName,
			Type:        ParseTypeExpr(bp.TypeName),
			Required:    bp.Required,
			Description: bp.Description,
		}
	} else if len(formParams) > 0 {
		rb := &RequestBodyAnnotation{IsForm: true, Required: true}
		for _, fp := range formParams {
			rb.Fields = append(rb.Fields, FormFieldAnnotation{
				Name:        fp.Name,
				TypeName:    fp.TypeName,
				Required:    fp.Required,
				Description: fp.Description,
			})
		}
		op.RequestBody = rb
	}

	// Attach headers to their matching responses
	for i := range op.Responses {
		for _, h := range headers {
			if h.Code == op.Responses[i].Code {
				op.Responses[i].Headers = append(op.Responses[i].Headers, h)
			}
		}
	}

	return op
}

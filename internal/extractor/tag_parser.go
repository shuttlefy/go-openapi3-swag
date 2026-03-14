package extractor

import (
	"strings"
)

var mimeAliases = map[string]string{
	"json":                  "application/json",
	"xml":                   "application/xml",
	"plain":                 "text/plain",
	"html":                  "text/html",
	"mpfd":                  "multipart/form-data",
	"x-www-form-urlencoded": "application/x-www-form-urlencoded",
	"json-api":              "application/vnd.api+json",
	"json-stream":           "application/x-json-stream",
	"octet-stream":          "application/octet-stream",
	"png":                   "image/png",
	"jpeg":                  "image/jpeg",
	"gif":                   "image/gif",
}

// primitiveTypeSet identifies types that are primitive schema types, not model refs.
var primitiveTypeSet = map[string]bool{
	"string": true, "integer": true, "number": true, "boolean": true, "file": true,
}

// parseTag parses a single comment line like "// @Summary Get user by ID"
// into a tag name and raw value. Returns ("", "") if the line is not a tag.
func parseTag(line string) (name, value string) {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "//")
	line = strings.TrimSpace(line)

	if !strings.HasPrefix(line, "@") {
		return "", ""
	}

	line = line[1:] // strip "@"
	idx := strings.IndexFunc(line, func(r rune) bool {
		return r == ' ' || r == '\t'
	})
	if idx == -1 {
		return strings.ToLower(line), ""
	}
	return strings.ToLower(line[:idx]), strings.TrimSpace(line[idx+1:])
}

// parseRouter parses "@Router /users/{id} [get]" value into path and method.
func parseRouter(value string) RouteInfo {
	parts := strings.Fields(value)
	if len(parts) < 2 {
		return RouteInfo{}
	}
	method := strings.Trim(parts[1], "[]")
	return RouteInfo{
		Path:   parts[0],
		Method: strings.ToUpper(method),
	}
}

// parseParam parses "@Param name in type required "description"" value.
// Format: name in type required "description"
// Extended: name in type required format "description"
func parseParam(value string) ParamAnnotation {
	p := ParamAnnotation{}
	parts, desc := splitQuotedFields(value)

	if len(parts) >= 1 {
		p.Name = parts[0]
	}
	if len(parts) >= 2 {
		p.In = parts[1]
	}
	if len(parts) >= 3 {
		p.TypeName = parts[2]
	}
	if len(parts) >= 4 {
		p.Required = strings.EqualFold(parts[3], "true")
	}
	if len(parts) >= 5 {
		p.Format = parts[4]
	}
	p.Description = desc
	return p
}

// splitQuotedFields splits "field1 field2 ... "quoted description"" into
// unquoted fields and the first quoted string.
func splitQuotedFields(s string) (fields []string, desc string) {
	quoteIdx := strings.Index(s, `"`)
	var fieldPart string
	if quoteIdx >= 0 {
		fieldPart = s[:quoteIdx]
		rest := s[quoteIdx:]
		endQuote := strings.LastIndex(rest, `"`)
		if endQuote > 0 {
			desc = rest[1:endQuote]
		} else {
			desc = strings.Trim(rest, `"`)
		}
	} else {
		fieldPart = s
	}
	for _, f := range strings.Fields(fieldPart) {
		fields = append(fields, f)
	}
	return
}

// parseResponse parses "@Success 200 {object} UserResponse "description"" value.
// Supports: {object}, {array}, {string}, {integer}, {number}, {boolean}, and no-body (code only).
func parseResponse(value string) ResponseAnnotation {
	r := ResponseAnnotation{}

	parts, desc := splitQuotedFields(value)
	r.Description = desc

	if len(parts) == 0 {
		return r
	}

	r.Code = parts[0]

	if len(parts) == 1 {
		return r
	}

	if len(parts) >= 2 {
		wrapper := strings.Trim(parts[1], "{}")
		wrapperLower := strings.ToLower(wrapper)
		r.IsArray = wrapperLower == "array"
		r.IsPrimitive = primitiveTypeSet[wrapperLower] && wrapperLower != "file"
	}
	if len(parts) >= 3 {
		r.TypeName = parts[2]
		r.Type = ParseTypeExpr(parts[2])
	}
	return r
}

// parseHeader parses "@Header code {type} name "description"".
func parseHeader(value string) ResponseHeaderAnnotation {
	h := ResponseHeaderAnnotation{}
	parts, desc := splitQuotedFields(value)
	h.Description = desc

	if len(parts) >= 1 {
		h.Code = parts[0]
	}
	if len(parts) >= 2 {
		h.TypeName = strings.Trim(parts[1], "{}")
	}
	if len(parts) >= 3 {
		h.Name = parts[2]
	}
	return h
}

// parseMimeTypes expands MIME type aliases ("json" → "application/json").
func parseMimeTypes(value string) []string {
	var result []string
	for _, raw := range strings.Split(value, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if full, ok := mimeAliases[raw]; ok {
			result = append(result, full)
		} else {
			result = append(result, raw)
		}
	}
	return result
}

// parseSecurity parses "@Security OAuth2[read, write]" or "@Security ApiKeyAuth".
func parseSecurity(value string) SecurityRequirement {
	sr := SecurityRequirement{}
	bracketIdx := strings.Index(value, "[")
	if bracketIdx < 0 {
		sr.Name = strings.TrimSpace(value)
		return sr
	}

	sr.Name = strings.TrimSpace(value[:bracketIdx])
	scopePart := value[bracketIdx:]
	scopePart = strings.Trim(scopePart, "[]")
	for _, s := range strings.Split(scopePart, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			sr.Scopes = append(sr.Scopes, s)
		}
	}
	return sr
}

// parseServer parses "@server url description".
func parseServer(value string) ServerAnnotation {
	parts := strings.SplitN(value, " ", 2)
	s := ServerAnnotation{URL: parts[0]}
	if len(parts) > 1 {
		s.Description = strings.Trim(parts[1], `"`)
	}
	return s
}

// ParseTypeExpr parses a possibly-composite type expression.
//
// Simple:      "User"           → TypeExpr{Name:"User"}
// Array:       "[]User"         → TypeExpr{Name:"[]User"}
// Composite:   "PageData{data=[]User}"
//
//	→ TypeExpr{Name:"PageData", Overrides:[{Field:"data",TypeExpr:"[]User"}]}
//
// Multi-field: "PageData{data=[]User,code=int}"
//
//	→ TypeExpr{Name:"PageData", Overrides:[..., ...]}
//
// Nested:      "Response{data=PageData{items=[]User}}"
//
//	→ TypeExpr{Name:"Response", Overrides:[{Field:"data",TypeExpr:"PageData{items=[]User}"}]}
//
// The TypeExpr strings stored in FieldOverride.TypeExpr are themselves valid
// inputs to ParseTypeExpr, enabling recursive schema construction in the builder.
func ParseTypeExpr(raw string) TypeExpr {
	braceIdx := strings.Index(raw, "{")
	if braceIdx < 0 {
		return TypeExpr{Name: raw}
	}

	baseName := raw[:braceIdx]
	rest := raw[braceIdx:]

	// Strip outer braces
	if len(rest) < 2 || rest[0] != '{' || rest[len(rest)-1] != '}' {
		return TypeExpr{Name: raw}
	}
	// Note: override values (ov.TypeExpr strings) are intentionally kept as raw
	// strings here so that FieldOverride remains a plain data struct.  The builder
	// calls ParseTypeExpr recursively on each override value when building schemas.
	inner := rest[1 : len(rest)-1]

	var overrides []FieldOverride
	for _, pair := range splitOverridePairs(inner) {
		eqIdx := strings.Index(pair, "=")
		if eqIdx < 0 {
			continue
		}
		field := strings.TrimSpace(pair[:eqIdx])
		typExpr := strings.TrimSpace(pair[eqIdx+1:])
		if field != "" && typExpr != "" {
			overrides = append(overrides, FieldOverride{
				Field:    field,
				TypeExpr: typExpr,
			})
		}
	}

	return TypeExpr{Name: baseName, Overrides: overrides}
}

// splitOverridePairs splits "data=[]User,code=int" respecting nested braces.
// e.g. "data=PageData{items=[]User},code=int" → ["data=PageData{items=[]User}", "code=int"]
func splitOverridePairs(s string) []string {
	var pairs []string
	depth := 0
	start := 0
	for i, ch := range s {
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
		case ',':
			if depth == 0 {
				pairs = append(pairs, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		pairs = append(pairs, s[start:])
	}
	return pairs
}

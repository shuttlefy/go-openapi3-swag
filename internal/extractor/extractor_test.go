package extractor

import (
	"testing"

	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

func buildAST(fns ...parser.RawFunc) *parser.RawAST {
	return &parser.RawAST{
		Package:   "main",
		Functions: fns,
	}
}

func fn(name string, comments ...string) parser.RawFunc {
	return parser.RawFunc{
		Name:     name,
		FilePath: "test.go",
		Line:     1,
		Comments: comments,
	}
}

// ============================================================
// Tag Parser Tests
// ============================================================

func TestParseTag_Basic(t *testing.T) {
	tests := []struct {
		line     string
		wantName string
		wantVal  string
	}{
		{"// @Summary Get user by ID", "summary", "Get user by ID"},
		{"//  @Description  Some desc  ", "description", "Some desc"},
		{"// @Tags users, admin", "tags", "users, admin"},
		{"// @Deprecated", "deprecated", ""},
		{"// not a tag", "", ""},
		{"", "", ""},
		{"// @Router /users/{id} [get]", "router", "/users/{id} [get]"},
		{"// @contact.name John", "contact.name", "John"},
		{"// @license.name MIT", "license.name", "MIT"},
	}

	for _, tt := range tests {
		name, val := parseTag(tt.line)
		if name != tt.wantName || val != tt.wantVal {
			t.Errorf("parseTag(%q) = (%q, %q), want (%q, %q)",
				tt.line, name, val, tt.wantName, tt.wantVal)
		}
	}
}

func TestParseRouter(t *testing.T) {
	ri := parseRouter("/users/{id} [get]")
	if ri.Path != "/users/{id}" || ri.Method != "GET" {
		t.Errorf("got %+v, want /users/{id} GET", ri)
	}

	ri = parseRouter("/pets [post]")
	if ri.Path != "/pets" || ri.Method != "POST" {
		t.Errorf("got %+v, want /pets POST", ri)
	}
}

func TestParseParam(t *testing.T) {
	p := parseParam(`id path int true "User ID"`)
	if p.Name != "id" || p.In != "path" || p.TypeName != "int" || !p.Required || p.Description != "User ID" {
		t.Errorf("got %+v", p)
	}

	p = parseParam(`status query string false "Filter status"`)
	if p.Name != "status" || p.In != "query" || p.Required {
		t.Errorf("got %+v", p)
	}

	p = parseParam(`body body CreateUserRequest true "User to create"`)
	if p.Name != "body" || p.In != "body" || p.TypeName != "CreateUserRequest" {
		t.Errorf("got %+v", p)
	}
}

func TestParseParam_WithFormat(t *testing.T) {
	p := parseParam(`id path integer true int64 "User ID"`)
	if p.Name != "id" || p.TypeName != "integer" || p.Format != "int64" {
		t.Errorf("got %+v", p)
	}

	p = parseParam(`created_at query string false date-time "Creation date"`)
	if p.Format != "date-time" {
		t.Errorf("format = %q, want date-time", p.Format)
	}
}

func TestParseResponse(t *testing.T) {
	r := parseResponse(`200 {object} UserResponse "OK"`)
	if r.Code != "200" || r.TypeName != "UserResponse" || r.IsArray || r.Description != "OK" {
		t.Errorf("got %+v", r)
	}

	r = parseResponse(`200 {array} UserResponse "List of users"`)
	if r.Code != "200" || r.TypeName != "UserResponse" || !r.IsArray || r.Description != "List of users" {
		t.Errorf("got %+v", r)
	}

	r = parseResponse(`404 {object} ErrorResponse "Not found"`)
	if r.Code != "404" || r.TypeName != "ErrorResponse" {
		t.Errorf("got %+v", r)
	}
}

func TestParseResponse_Primitive(t *testing.T) {
	r := parseResponse(`200 {string} string "A raw string"`)
	if !r.IsPrimitive {
		t.Errorf("expected IsPrimitive=true, got %+v", r)
	}

	r = parseResponse(`200 {integer} int "A count"`)
	if !r.IsPrimitive {
		t.Errorf("expected IsPrimitive=true for integer, got %+v", r)
	}

	r = parseResponse(`200 {object} Foo "Not primitive"`)
	if r.IsPrimitive {
		t.Errorf("expected IsPrimitive=false for object")
	}
}

func TestParseResponse_NoBody(t *testing.T) {
	r := parseResponse(`204 "No Content"`)
	if r.Code != "204" || r.TypeName != "" || r.Description != "No Content" {
		t.Errorf("got %+v", r)
	}
}

func TestParseHeader(t *testing.T) {
	h := parseHeader(`200 {string} X-Request-Id "Request ID"`)
	if h.Code != "200" || h.TypeName != "string" || h.Name != "X-Request-Id" || h.Description != "Request ID" {
		t.Errorf("got %+v", h)
	}
}

func TestParseMimeTypes(t *testing.T) {
	result := parseMimeTypes("json, xml")
	if len(result) != 2 || result[0] != "application/json" || result[1] != "application/xml" {
		t.Errorf("got %v", result)
	}

	result = parseMimeTypes("application/json")
	if len(result) != 1 || result[0] != "application/json" {
		t.Errorf("got %v", result)
	}

	result = parseMimeTypes("mpfd")
	if len(result) != 1 || result[0] != "multipart/form-data" {
		t.Errorf("got %v", result)
	}
}

func TestParseSecurity(t *testing.T) {
	sr := parseSecurity("ApiKeyAuth")
	if sr.Name != "ApiKeyAuth" || len(sr.Scopes) != 0 {
		t.Errorf("got %+v", sr)
	}

	sr = parseSecurity("OAuth2[read, write]")
	if sr.Name != "OAuth2" || len(sr.Scopes) != 2 || sr.Scopes[0] != "read" || sr.Scopes[1] != "write" {
		t.Errorf("got %+v", sr)
	}

	sr = parseSecurity("OAuth2[admin]")
	if sr.Name != "OAuth2" || len(sr.Scopes) != 1 || sr.Scopes[0] != "admin" {
		t.Errorf("got %+v", sr)
	}
}

func TestParseServer(t *testing.T) {
	s := parseServer(`https://api.example.com "Production server"`)
	if s.URL != "https://api.example.com" || s.Description != "Production server" {
		t.Errorf("got %+v", s)
	}

	s = parseServer("http://localhost:8080")
	if s.URL != "http://localhost:8080" || s.Description != "" {
		t.Errorf("got %+v", s)
	}
}

// ============================================================
// Extractor Tests — Operations (basic, preserved from before)
// ============================================================

func TestExtract_SingleRoute(t *testing.T) {
	ast := buildAST(fn("GetUser",
		"// @Summary Get user by ID",
		"// @Description Returns a single user",
		"// @Tags users",
		"// @Accept json",
		"// @Produce json",
		"// @Param id path int true \"User ID\"",
		"// @Success 200 {object} UserResponse \"OK\"",
		"// @Failure 404 {object} ErrorResponse \"Not found\"",
		"// @Router /users/{id} [get]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(result.Operations))
	}

	op := result.Operations[0]
	if op.Summary != "Get user by ID" {
		t.Errorf("summary = %q", op.Summary)
	}
	if op.Description != "Returns a single user" {
		t.Errorf("description = %q", op.Description)
	}
	if op.Route.Path != "/users/{id}" || op.Route.Method != "GET" {
		t.Errorf("route = %+v", op.Route)
	}
	if len(op.Tags) != 1 || op.Tags[0] != "users" {
		t.Errorf("tags = %v", op.Tags)
	}
	if len(op.Accept) != 1 || op.Accept[0] != "application/json" {
		t.Errorf("accept = %v", op.Accept)
	}
	if len(op.Produce) != 1 || op.Produce[0] != "application/json" {
		t.Errorf("produce = %v", op.Produce)
	}
	if len(op.Params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(op.Params))
	}
	if op.Params[0].Name != "id" || op.Params[0].In != "path" || !op.Params[0].Required {
		t.Errorf("param = %+v", op.Params[0])
	}
	if len(op.Responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(op.Responses))
	}
}

func TestExtract_MultipleParams(t *testing.T) {
	ast := buildAST(fn("SearchUsers",
		"// @Summary Search users",
		"// @Param q query string true \"Search query\"",
		"// @Param page query int false \"Page number\"",
		"// @Param limit query int false \"Page size\"",
		"// @Success 200 {array} UserResponse \"Users\"",
		"// @Router /users [get]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if len(op.Params) != 3 {
		t.Fatalf("expected 3 params, got %d", len(op.Params))
	}
}

func TestExtract_MultipleResponses(t *testing.T) {
	ast := buildAST(fn("UpdateUser",
		"// @Summary Update user",
		"// @Success 200 {object} UserResponse \"Updated\"",
		"// @Failure 400 {object} ErrorResponse \"Bad request\"",
		"// @Failure 404 {object} ErrorResponse \"Not found\"",
		"// @Failure 500 {object} ErrorResponse \"Internal error\"",
		"// @Router /users/{id} [put]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if len(op.Responses) != 4 {
		t.Fatalf("expected 4 responses, got %d", len(op.Responses))
	}
	codes := []string{"200", "400", "404", "500"}
	for i, c := range codes {
		if op.Responses[i].Code != c {
			t.Errorf("response[%d].Code = %q, want %q", i, op.Responses[i].Code, c)
		}
	}
}

func TestExtract_SkipNoRouter(t *testing.T) {
	ast := buildAST(
		fn("HelperFunc",
			"// @Summary This is a helper",
		),
		fn("GetUser",
			"// @Summary Get user",
			"// @Router /users/{id} [get]",
		),
	)

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Operations) != 1 {
		t.Fatalf("expected 1 operation (skip no-router), got %d", len(result.Operations))
	}
	if result.Operations[0].FuncName != "GetUser" {
		t.Errorf("expected GetUser, got %s", result.Operations[0].FuncName)
	}
}

func TestExtract_OperationID(t *testing.T) {
	ast := buildAST(fn("ListPets",
		"// @Summary List pets",
		"// @ID listPets",
		"// @Router /pets [get]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	if result.Operations[0].OperationID != "listPets" {
		t.Errorf("operationId = %q", result.Operations[0].OperationID)
	}
}

func TestExtract_Deprecated(t *testing.T) {
	ast := buildAST(fn("OldEndpoint",
		"// @Summary Old endpoint",
		"// @Deprecated",
		"// @Router /old [get]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	if !result.Operations[0].Deprecated {
		t.Error("expected deprecated=true")
	}
}

func TestExtract_MultipleTags(t *testing.T) {
	ast := buildAST(fn("AdminCreateUser",
		"// @Summary Admin creates user",
		"// @Tags admin, users",
		"// @Router /admin/users [post]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	tags := result.Operations[0].Tags
	if len(tags) != 2 || tags[0] != "admin" || tags[1] != "users" {
		t.Errorf("tags = %v", tags)
	}
}

func TestExtract_ArrayResponse(t *testing.T) {
	ast := buildAST(fn("ListUsers",
		"// @Summary List users",
		"// @Success 200 {array} UserResponse \"Users list\"",
		"// @Router /users [get]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	resp := result.Operations[0].Responses[0]
	if !resp.IsArray {
		t.Error("expected IsArray=true")
	}
	if resp.TypeName != "UserResponse" {
		t.Errorf("typeName = %q", resp.TypeName)
	}
}

func TestExtract_DescriptionFallback(t *testing.T) {
	ast := buildAST(fn("GetUser",
		"// GetUser retrieves a user",
		"// from the database by ID.",
		"// @Summary Get user",
		"// @Router /users/{id} [get]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if op.Description != "GetUser retrieves a user from the database by ID." {
		t.Errorf("description fallback = %q", op.Description)
	}
}

func TestExtract_EmptyAST(t *testing.T) {
	ast := buildAST()

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Operations) != 0 {
		t.Errorf("expected 0 operations, got %d", len(result.Operations))
	}
}

func TestExtract_FuncMetadata(t *testing.T) {
	f := parser.RawFunc{
		Name:     "GetUser",
		FilePath: "handler/user.go",
		Line:     42,
		Comments: []string{
			"// @Summary Get user",
			"// @Router /users/{id} [get]",
		},
	}
	ast := buildAST(f)

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if op.FuncName != "GetUser" {
		t.Errorf("funcName = %q", op.FuncName)
	}
	if op.FilePath != "handler/user.go" {
		t.Errorf("filePath = %q", op.FilePath)
	}
	if op.Line != 42 {
		t.Errorf("line = %d", op.Line)
	}
}

// ============================================================
// Extractor Tests — Global Annotations
// ============================================================

func TestExtract_GlobalAnnotations(t *testing.T) {
	ast := buildAST(fn("main",
		"// @title Pet Store API",
		"// @version 1.0",
		"// @description A pet store server",
		"// @host localhost:8080",
		"// @BasePath /api/v1",
		"// @schemes http https",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	g := result.Global
	if g.Title != "Pet Store API" {
		t.Errorf("title = %q", g.Title)
	}
	if g.Version != "1.0" {
		t.Errorf("version = %q", g.Version)
	}
	if g.Description != "A pet store server" {
		t.Errorf("description = %q", g.Description)
	}
	if g.Host != "localhost:8080" {
		t.Errorf("host = %q", g.Host)
	}
	if g.BasePath != "/api/v1" {
		t.Errorf("basePath = %q", g.BasePath)
	}
	if len(g.Schemes) != 2 || g.Schemes[0] != "http" || g.Schemes[1] != "https" {
		t.Errorf("schemes = %v", g.Schemes)
	}
}

func TestExtract_GlobalTags(t *testing.T) {
	ast := buildAST(fn("main",
		"// @title My API",
		"// @version 1.0",
		`// @tag users "User management"`,
		`// @tag pets "Pet operations"`,
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Global.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(result.Global.Tags))
	}
	if result.Global.Tags[0].Name != "users" || result.Global.Tags[0].Description != "User management" {
		t.Errorf("tag[0] = %+v", result.Global.Tags[0])
	}
	if result.Global.Tags[1].Name != "pets" || result.Global.Tags[1].Description != "Pet operations" {
		t.Errorf("tag[1] = %+v", result.Global.Tags[1])
	}
}

func TestExtract_GlobalContact(t *testing.T) {
	ast := buildAST(fn("main",
		"// @title My API",
		"// @version 1.0",
		"// @contact.name API Support",
		"// @contact.url https://www.example.com/support",
		"// @contact.email support@example.com",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	c := result.Global.Contact
	if c.Name != "API Support" {
		t.Errorf("contact.name = %q", c.Name)
	}
	if c.URL != "https://www.example.com/support" {
		t.Errorf("contact.url = %q", c.URL)
	}
	if c.Email != "support@example.com" {
		t.Errorf("contact.email = %q", c.Email)
	}
}

func TestExtract_GlobalLicense(t *testing.T) {
	ast := buildAST(fn("main",
		"// @title My API",
		"// @version 1.0",
		"// @license.name Apache 2.0",
		"// @license.url https://www.apache.org/licenses/LICENSE-2.0.html",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	l := result.Global.License
	if l.Name != "Apache 2.0" {
		t.Errorf("license.name = %q", l.Name)
	}
	if l.URL != "https://www.apache.org/licenses/LICENSE-2.0.html" {
		t.Errorf("license.url = %q", l.URL)
	}
}

func TestExtract_GlobalTermsOfService(t *testing.T) {
	ast := buildAST(fn("main",
		"// @title My API",
		"// @version 1.0",
		"// @termsOfService https://example.com/terms",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	if result.Global.TermsOfService != "https://example.com/terms" {
		t.Errorf("termsOfService = %q", result.Global.TermsOfService)
	}
}

func TestExtract_GlobalExternalDocs(t *testing.T) {
	ast := buildAST(fn("main",
		"// @title My API",
		"// @version 1.0",
		"// @externalDocs.url https://example.com/docs",
		"// @externalDocs.description Find out more",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	if result.Global.ExternalDocs == nil {
		t.Fatal("expected externalDocs to be set")
	}
	if result.Global.ExternalDocs.URL != "https://example.com/docs" {
		t.Errorf("externalDocs.url = %q", result.Global.ExternalDocs.URL)
	}
	if result.Global.ExternalDocs.Description != "Find out more" {
		t.Errorf("externalDocs.description = %q", result.Global.ExternalDocs.Description)
	}
}

func TestExtract_GlobalServers(t *testing.T) {
	ast := buildAST(fn("main",
		"// @title My API",
		"// @version 1.0",
		`// @server https://api.example.com "Production"`,
		`// @server https://staging.example.com "Staging"`,
		"// @server http://localhost:8080",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	servers := result.Global.Servers
	if len(servers) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(servers))
	}
	if servers[0].URL != "https://api.example.com" || servers[0].Description != "Production" {
		t.Errorf("server[0] = %+v", servers[0])
	}
	if servers[1].URL != "https://staging.example.com" || servers[1].Description != "Staging" {
		t.Errorf("server[1] = %+v", servers[1])
	}
	if servers[2].URL != "http://localhost:8080" || servers[2].Description != "" {
		t.Errorf("server[2] = %+v", servers[2])
	}
}

// ============================================================
// Extractor Tests — OpenAPI 3 specific: RequestBody
// ============================================================

func TestExtract_BodyParamBecomesRequestBody(t *testing.T) {
	ast := buildAST(fn("CreateUser",
		"// @Summary Create a new user",
		"// @Accept json",
		"// @Param body body CreateUserRequest true \"User to create\"",
		"// @Success 201 {object} UserResponse \"Created\"",
		"// @Router /users [post]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]

	// body params should NOT appear in Params
	if len(op.Params) != 0 {
		t.Errorf("expected 0 params (body should be in RequestBody), got %d", len(op.Params))
	}

	if op.RequestBody == nil {
		t.Fatal("expected RequestBody to be set")
	}
	rb := op.RequestBody
	if rb.TypeName != "CreateUserRequest" {
		t.Errorf("requestBody.typeName = %q", rb.TypeName)
	}
	if !rb.Required {
		t.Error("expected requestBody.required=true")
	}
	if rb.Description != "User to create" {
		t.Errorf("requestBody.description = %q", rb.Description)
	}
	if rb.IsForm {
		t.Error("expected requestBody.isForm=false for body param")
	}
}

func TestExtract_FormDataBecomesRequestBody(t *testing.T) {
	ast := buildAST(fn("UploadAvatar",
		"// @Summary Upload avatar",
		"// @Accept mpfd",
		"// @Param file formData file true \"Avatar file\"",
		"// @Param name formData string true \"User name\"",
		"// @Success 200 {object} UploadResult \"OK\"",
		"// @Router /users/avatar [post]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if len(op.Params) != 0 {
		t.Errorf("expected 0 params (formData should be in RequestBody), got %d", len(op.Params))
	}
	if op.RequestBody == nil {
		t.Fatal("expected RequestBody to be set")
	}
	rb := op.RequestBody
	if !rb.IsForm {
		t.Error("expected requestBody.isForm=true")
	}
	if len(rb.Fields) != 2 {
		t.Fatalf("expected 2 form fields, got %d", len(rb.Fields))
	}
	if rb.Fields[0].Name != "file" || rb.Fields[0].TypeName != "file" {
		t.Errorf("field[0] = %+v", rb.Fields[0])
	}
	if rb.Fields[1].Name != "name" || rb.Fields[1].TypeName != "string" {
		t.Errorf("field[1] = %+v", rb.Fields[1])
	}
}

func TestExtract_MixedParamsAndBody(t *testing.T) {
	ast := buildAST(fn("UpdateUser",
		"// @Summary Update user",
		"// @Param id path int true \"User ID\"",
		"// @Param body body UpdateUserRequest true \"User data\"",
		"// @Param X-Token header string true \"Auth token\"",
		"// @Success 200 {object} UserResponse \"OK\"",
		"// @Router /users/{id} [put]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	// path + header = 2 params, body → RequestBody
	if len(op.Params) != 2 {
		t.Fatalf("expected 2 params (path + header), got %d", len(op.Params))
	}
	if op.Params[0].Name != "id" || op.Params[0].In != "path" {
		t.Errorf("params[0] = %+v", op.Params[0])
	}
	if op.Params[1].Name != "X-Token" || op.Params[1].In != "header" {
		t.Errorf("params[1] = %+v", op.Params[1])
	}
	if op.RequestBody == nil || op.RequestBody.TypeName != "UpdateUserRequest" {
		t.Errorf("requestBody = %+v", op.RequestBody)
	}
}

// ============================================================
// Extractor Tests — Accept / Produce
// ============================================================

func TestExtract_AcceptProduce(t *testing.T) {
	ast := buildAST(fn("CreateUser",
		"// @Summary Create user",
		"// @Accept json, xml",
		"// @Produce json",
		"// @Router /users [post]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if len(op.Accept) != 2 {
		t.Fatalf("expected 2 accept types, got %d", len(op.Accept))
	}
	if op.Accept[0] != "application/json" || op.Accept[1] != "application/xml" {
		t.Errorf("accept = %v", op.Accept)
	}
	if len(op.Produce) != 1 || op.Produce[0] != "application/json" {
		t.Errorf("produce = %v", op.Produce)
	}
}

// ============================================================
// Extractor Tests — Security with Scopes
// ============================================================

func TestExtract_SecuritySimple(t *testing.T) {
	ast := buildAST(fn("DeleteUser",
		"// @Summary Delete user",
		"// @Security ApiKeyAuth",
		"// @Router /users/{id} [delete]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if len(op.Security) != 1 {
		t.Fatalf("expected 1 security, got %d", len(op.Security))
	}
	if op.Security[0].Name != "ApiKeyAuth" || len(op.Security[0].Scopes) != 0 {
		t.Errorf("security = %+v", op.Security[0])
	}
}

func TestExtract_SecurityWithScopes(t *testing.T) {
	ast := buildAST(fn("CreatePet",
		"// @Summary Create pet",
		"// @Security OAuth2[write, admin]",
		"// @Router /pets [post]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if len(op.Security) != 1 {
		t.Fatalf("expected 1 security, got %d", len(op.Security))
	}
	sec := op.Security[0]
	if sec.Name != "OAuth2" {
		t.Errorf("security.name = %q", sec.Name)
	}
	if len(sec.Scopes) != 2 || sec.Scopes[0] != "write" || sec.Scopes[1] != "admin" {
		t.Errorf("security.scopes = %v", sec.Scopes)
	}
}

func TestExtract_MultipleSecuritySchemes(t *testing.T) {
	ast := buildAST(fn("AdminAction",
		"// @Summary Admin action",
		"// @Security ApiKeyAuth",
		"// @Security OAuth2[admin]",
		"// @Router /admin/action [post]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if len(op.Security) != 2 {
		t.Fatalf("expected 2 security, got %d", len(op.Security))
	}
	if op.Security[0].Name != "ApiKeyAuth" {
		t.Errorf("security[0] = %+v", op.Security[0])
	}
	if op.Security[1].Name != "OAuth2" || len(op.Security[1].Scopes) != 1 {
		t.Errorf("security[1] = %+v", op.Security[1])
	}
}

// ============================================================
// Extractor Tests — Response Headers
// ============================================================

func TestExtract_ResponseHeaders(t *testing.T) {
	ast := buildAST(fn("GetUser",
		"// @Summary Get user",
		"// @Success 200 {object} UserResponse \"OK\"",
		`// @Header 200 {string} X-Request-Id "Request tracking ID"`,
		`// @Header 200 {integer} X-RateLimit-Remaining "Remaining requests"`,
		"// @Router /users/{id} [get]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	resp := op.Responses[0]
	if len(resp.Headers) != 2 {
		t.Fatalf("expected 2 headers on 200 response, got %d", len(resp.Headers))
	}
	if resp.Headers[0].Name != "X-Request-Id" || resp.Headers[0].TypeName != "string" {
		t.Errorf("header[0] = %+v", resp.Headers[0])
	}
	if resp.Headers[1].Name != "X-RateLimit-Remaining" || resp.Headers[1].TypeName != "integer" {
		t.Errorf("header[1] = %+v", resp.Headers[1])
	}
}

// ============================================================
// Extractor Tests — Primitive and No-Body Responses
// ============================================================

func TestExtract_PrimitiveResponse(t *testing.T) {
	ast := buildAST(fn("GetHealth",
		"// @Summary Health check",
		`// @Success 200 {string} string "OK"`,
		"// @Router /health [get]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	resp := result.Operations[0].Responses[0]
	if !resp.IsPrimitive {
		t.Error("expected IsPrimitive=true")
	}
	if resp.TypeName != "string" {
		t.Errorf("typeName = %q", resp.TypeName)
	}
}

func TestExtract_NoBodyResponse(t *testing.T) {
	ast := buildAST(fn("DeleteUser",
		"// @Summary Delete user",
		`// @Success 204 "No Content"`,
		"// @Router /users/{id} [delete]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	resp := result.Operations[0].Responses[0]
	if resp.Code != "204" {
		t.Errorf("code = %q", resp.Code)
	}
	if resp.TypeName != "" {
		t.Errorf("expected empty typeName for 204, got %q", resp.TypeName)
	}
	if resp.Description != "No Content" {
		t.Errorf("description = %q", resp.Description)
	}
}

// ============================================================
// Extractor Tests — Security Definitions (Global)
// ============================================================

func TestExtract_GlobalSecurityDef_ApiKey(t *testing.T) {
	ast := buildAST(fn("main",
		"// @title My API",
		"// @version 1.0",
		"// @securityDefinitions.apikey ApiKeyAuth",
		"// @securityDefinitions.apikey.in header",
		"// @securityDefinitions.apikey.name X-API-Key",
		"// @securityDefinitions.apikey.description API key auth",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Global.SecurityDefs) != 1 {
		t.Fatalf("expected 1 security def, got %d", len(result.Global.SecurityDefs))
	}
	sd := result.Global.SecurityDefs[0]
	if sd.Name != "ApiKeyAuth" {
		t.Errorf("name = %q", sd.Name)
	}
	if sd.Type != "apiKey" {
		t.Errorf("type = %q", sd.Type)
	}
	if sd.In != "header" {
		t.Errorf("in = %q", sd.In)
	}
	if sd.FieldName != "X-API-Key" {
		t.Errorf("fieldName = %q", sd.FieldName)
	}
	if sd.Description != "API key auth" {
		t.Errorf("description = %q", sd.Description)
	}
}

func TestExtract_GlobalSecurityDef_Basic(t *testing.T) {
	ast := buildAST(fn("main",
		"// @title My API",
		"// @version 1.0",
		"// @securityDefinitions.basic BasicAuth",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Global.SecurityDefs) != 1 {
		t.Fatalf("expected 1 security def, got %d", len(result.Global.SecurityDefs))
	}
	sd := result.Global.SecurityDefs[0]
	if sd.Name != "BasicAuth" || sd.Type != "http" || sd.Scheme != "basic" {
		t.Errorf("got %+v", sd)
	}
}

func TestExtract_GlobalSecurityDef_Bearer(t *testing.T) {
	ast := buildAST(fn("main",
		"// @title My API",
		"// @version 1.0",
		"// @securityDefinitions.bearer BearerAuth",
		"// @securityDefinitions.bearer.bearerFormat JWT",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Global.SecurityDefs) != 1 {
		t.Fatalf("expected 1 security def, got %d", len(result.Global.SecurityDefs))
	}
	sd := result.Global.SecurityDefs[0]
	if sd.Name != "BearerAuth" || sd.Type != "http" || sd.Scheme != "bearer" {
		t.Errorf("got %+v", sd)
	}
	if sd.BearerFormat != "JWT" {
		t.Errorf("bearerFormat = %q", sd.BearerFormat)
	}
}

func TestExtract_GlobalSecurityDef_OAuth2(t *testing.T) {
	ast := buildAST(fn("main",
		"// @title My API",
		"// @version 1.0",
		"// @securityDefinitions.oauth2.authorizationCode OAuth2",
		"// @securityDefinitions.oauth2.authorizationCode.authorizationUrl https://example.com/oauth/authorize",
		"// @securityDefinitions.oauth2.authorizationCode.tokenUrl https://example.com/oauth/token",
		`// @securityDefinitions.oauth2.authorizationCode.scope.read "Read access"`,
		`// @securityDefinitions.oauth2.authorizationCode.scope.write "Write access"`,
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Global.SecurityDefs) != 1 {
		t.Fatalf("expected 1 security def, got %d", len(result.Global.SecurityDefs))
	}
	sd := result.Global.SecurityDefs[0]
	if sd.Name != "OAuth2" || sd.Type != "oauth2" {
		t.Errorf("got %+v", sd)
	}
	if sd.OAuthFlowType != "authorizationCode" {
		t.Errorf("flowType = %q", sd.OAuthFlowType)
	}
	if sd.AuthorizationURL != "https://example.com/oauth/authorize" {
		t.Errorf("authorizationUrl = %q", sd.AuthorizationURL)
	}
	if sd.TokenURL != "https://example.com/oauth/token" {
		t.Errorf("tokenUrl = %q", sd.TokenURL)
	}
	if len(sd.Scopes) != 2 {
		t.Fatalf("expected 2 scopes, got %d", len(sd.Scopes))
	}
	if sd.Scopes["read"] != "Read access" {
		t.Errorf("scope[read] = %q", sd.Scopes["read"])
	}
	if sd.Scopes["write"] != "Write access" {
		t.Errorf("scope[write] = %q", sd.Scopes["write"])
	}
}

// ============================================================
// Extractor Tests — Full Realistic API
// ============================================================

func TestExtract_FullRealisticAPI(t *testing.T) {
	ast := buildAST(
		fn("main",
			"// @title Pet Store API",
			"// @version 2.0.0",
			"// @description A sample pet store server.",
			"// @termsOfService https://example.com/terms",
			"// @contact.name API Support",
			"// @contact.email support@example.com",
			"// @license.name Apache 2.0",
			"// @license.url https://www.apache.org/licenses/LICENSE-2.0.html",
			`// @server https://api.petstore.com "Production"`,
			`// @server http://localhost:8080 "Development"`,
			`// @tag pets "Everything about your Pets"`,
			"// @securityDefinitions.apikey ApiKeyAuth",
			"// @securityDefinitions.apikey.in header",
			"// @securityDefinitions.apikey.name X-API-Key",
		),
		fn("ListPets",
			"// @Summary List all pets",
			"// @Description Get a list of all pets in the store",
			"// @ID listPets",
			"// @Tags pets",
			"// @Accept json",
			"// @Produce json, xml",
			"// @Param limit query int false \"Max items\"",
			"// @Param offset query int false \"Offset\"",
			"// @Success 200 {array} Pet \"Pets list\"",
			`// @Header 200 {string} X-Total-Count "Total number of pets"`,
			"// @Failure 500 {object} Error \"Server error\"",
			"// @Security ApiKeyAuth",
			"// @Router /pets [get]",
		),
		fn("CreatePet",
			"// @Summary Create a pet",
			"// @Tags pets",
			"// @Accept json",
			"// @Produce json",
			"// @Param body body CreatePetRequest true \"Pet to create\"",
			"// @Success 201 {object} Pet \"Created\"",
			"// @Failure 400 {object} Error \"Validation error\"",
			"// @Security ApiKeyAuth",
			"// @Router /pets [post]",
		),
		fn("DeletePet",
			"// @Summary Delete a pet",
			"// @Tags pets",
			"// @Param id path int true \"Pet ID\"",
			`// @Success 204 "No Content"`,
			"// @Failure 404 {object} Error \"Not found\"",
			"// @Security ApiKeyAuth",
			"// @Deprecated",
			"// @Router /pets/{id} [delete]",
		),
	)

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	// Global
	g := result.Global
	if g.Title != "Pet Store API" || g.Version != "2.0.0" {
		t.Errorf("info: title=%q, version=%q", g.Title, g.Version)
	}
	if g.Contact.Name != "API Support" || g.Contact.Email != "support@example.com" {
		t.Errorf("contact = %+v", g.Contact)
	}
	if g.License.Name != "Apache 2.0" {
		t.Errorf("license = %+v", g.License)
	}
	if len(g.Servers) != 2 {
		t.Errorf("servers count = %d", len(g.Servers))
	}
	if len(g.Tags) != 1 || g.Tags[0].Name != "pets" {
		t.Errorf("tags = %+v", g.Tags)
	}
	if len(g.SecurityDefs) != 1 || g.SecurityDefs[0].Type != "apiKey" {
		t.Errorf("securityDefs = %+v", g.SecurityDefs)
	}
	if g.SecurityDefs[0].In != "header" || g.SecurityDefs[0].FieldName != "X-API-Key" {
		t.Errorf("apiKey details = %+v", g.SecurityDefs[0])
	}

	// Operations
	if len(result.Operations) != 3 {
		t.Fatalf("expected 3 operations, got %d", len(result.Operations))
	}

	// ListPets
	list := result.Operations[0]
	if list.OperationID != "listPets" {
		t.Errorf("listPets.operationId = %q", list.OperationID)
	}
	if len(list.Params) != 2 {
		t.Errorf("listPets.params count = %d", len(list.Params))
	}
	if len(list.Produce) != 2 {
		t.Errorf("listPets.produce count = %d", len(list.Produce))
	}
	if len(list.Responses) > 0 && len(list.Responses[0].Headers) != 1 {
		t.Errorf("listPets 200 headers count = %d", len(list.Responses[0].Headers))
	}

	// CreatePet — body → RequestBody
	create := result.Operations[1]
	if create.RequestBody == nil || create.RequestBody.TypeName != "CreatePetRequest" {
		t.Errorf("createPet.requestBody = %+v", create.RequestBody)
	}
	if len(create.Params) != 0 {
		t.Errorf("createPet should have 0 path/query params, got %d", len(create.Params))
	}

	// DeletePet — deprecated, 204 no-body
	del := result.Operations[2]
	if !del.Deprecated {
		t.Error("deletePet should be deprecated")
	}
	if del.Responses[0].Code != "204" || del.Responses[0].TypeName != "" {
		t.Errorf("deletePet 204 response = %+v", del.Responses[0])
	}
}

// ============================================================
// parseTypeExpr Tests
// ============================================================

func TestParseTypeExpr_Simple(t *testing.T) {
	te := ParseTypeExpr("User")
	if te.Name != "User" || len(te.Overrides) != 0 {
		t.Errorf("got %+v", te)
	}
}

func TestParseTypeExpr_Array(t *testing.T) {
	te := ParseTypeExpr("[]User")
	if te.Name != "[]User" || len(te.Overrides) != 0 {
		t.Errorf("got %+v", te)
	}
}

func TestParseTypeExpr_SingleOverride(t *testing.T) {
	te := ParseTypeExpr("PageData{data=[]User}")
	if te.Name != "PageData" {
		t.Errorf("name = %q", te.Name)
	}
	if len(te.Overrides) != 1 {
		t.Fatalf("expected 1 override, got %d", len(te.Overrides))
	}
	if te.Overrides[0].Field != "data" || te.Overrides[0].TypeExpr != "[]User" {
		t.Errorf("override = %+v", te.Overrides[0])
	}
}

func TestParseTypeExpr_MultiOverride(t *testing.T) {
	te := ParseTypeExpr("PageData{data=[]User,code=int}")
	if te.Name != "PageData" {
		t.Errorf("name = %q", te.Name)
	}
	if len(te.Overrides) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(te.Overrides))
	}
	if te.Overrides[0].Field != "data" || te.Overrides[0].TypeExpr != "[]User" {
		t.Errorf("override[0] = %+v", te.Overrides[0])
	}
	if te.Overrides[1].Field != "code" || te.Overrides[1].TypeExpr != "int" {
		t.Errorf("override[1] = %+v", te.Overrides[1])
	}
}

func TestParseTypeExpr_NestedComposite(t *testing.T) {
	te := ParseTypeExpr("Response{data=PageData{items=[]User},meta=Meta}")
	if te.Name != "Response" {
		t.Errorf("name = %q", te.Name)
	}
	if len(te.Overrides) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(te.Overrides))
	}
	if te.Overrides[0].Field != "data" || te.Overrides[0].TypeExpr != "PageData{items=[]User}" {
		t.Errorf("override[0] = %+v", te.Overrides[0])
	}
	if te.Overrides[1].Field != "meta" || te.Overrides[1].TypeExpr != "Meta" {
		t.Errorf("override[1] = %+v", te.Overrides[1])
	}
}

func TestParseTypeExpr_MapOverride(t *testing.T) {
	te := ParseTypeExpr("PageData{data=map[string]User}")
	if te.Name != "PageData" {
		t.Errorf("name = %q", te.Name)
	}
	if len(te.Overrides) != 1 {
		t.Fatalf("expected 1 override, got %d", len(te.Overrides))
	}
	if te.Overrides[0].TypeExpr != "map[string]User" {
		t.Errorf("override typeExpr = %q", te.Overrides[0].TypeExpr)
	}
}

// ============================================================
// Extractor Tests — Composite Response Types
// ============================================================

func TestExtract_CompositeResponse_PageData(t *testing.T) {
	ast := buildAST(fn("ListUsers",
		"// @Summary List users",
		`// @Success 200 {object} PageData{data=[]User} "Paginated users"`,
		"// @Router /users [get]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	resp := result.Operations[0].Responses[0]
	if resp.TypeName != "PageData{data=[]User}" {
		t.Errorf("typeName = %q", resp.TypeName)
	}
	if resp.Type.Name != "PageData" {
		t.Errorf("type.name = %q", resp.Type.Name)
	}
	if len(resp.Type.Overrides) != 1 {
		t.Fatalf("expected 1 override, got %d", len(resp.Type.Overrides))
	}
	ov := resp.Type.Overrides[0]
	if ov.Field != "data" || ov.TypeExpr != "[]User" {
		t.Errorf("override = %+v", ov)
	}
}

func TestExtract_CompositeResponse_MultiField(t *testing.T) {
	ast := buildAST(fn("ListOrders",
		"// @Summary List orders",
		`// @Success 200 {object} PageData{data=[]Order,total=int64} "Paginated orders"`,
		"// @Router /orders [get]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	resp := result.Operations[0].Responses[0]
	if resp.Type.Name != "PageData" {
		t.Errorf("type.name = %q", resp.Type.Name)
	}
	if len(resp.Type.Overrides) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(resp.Type.Overrides))
	}
	if resp.Type.Overrides[0].Field != "data" || resp.Type.Overrides[0].TypeExpr != "[]Order" {
		t.Errorf("override[0] = %+v", resp.Type.Overrides[0])
	}
	if resp.Type.Overrides[1].Field != "total" || resp.Type.Overrides[1].TypeExpr != "int64" {
		t.Errorf("override[1] = %+v", resp.Type.Overrides[1])
	}
}

func TestExtract_CompositeResponse_ArrayWrapper(t *testing.T) {
	ast := buildAST(fn("ListUsers",
		"// @Summary List users",
		`// @Success 200 {array} PageData{data=User} "Users"`,
		"// @Router /users [get]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	resp := result.Operations[0].Responses[0]
	if !resp.IsArray {
		t.Error("expected IsArray=true")
	}
	if resp.Type.Name != "PageData" {
		t.Errorf("type.name = %q", resp.Type.Name)
	}
}

func TestExtract_CompositeResponse_Simple(t *testing.T) {
	ast := buildAST(fn("GetUser",
		"// @Summary Get user",
		`// @Success 200 {object} User "OK"`,
		"// @Router /users/{id} [get]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	resp := result.Operations[0].Responses[0]
	if resp.Type.Name != "User" {
		t.Errorf("type.name = %q", resp.Type.Name)
	}
	if len(resp.Type.Overrides) != 0 {
		t.Errorf("expected 0 overrides for simple type, got %d", len(resp.Type.Overrides))
	}
}

func TestExtract_CompositeRequestBody(t *testing.T) {
	ast := buildAST(fn("CreateBatch",
		"// @Summary Create batch",
		`// @Param body body BatchRequest{items=[]CreateUserRequest} true "Batch"`,
		"// @Success 200 {object} BatchResponse \"OK\"",
		"// @Router /batch [post]",
	))

	e := NewGoExtractor()
	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if op.RequestBody == nil {
		t.Fatal("expected RequestBody to be set")
	}
	rb := op.RequestBody
	if rb.Type.Name != "BatchRequest" {
		t.Errorf("type.name = %q", rb.Type.Name)
	}
	if len(rb.Type.Overrides) != 1 {
		t.Fatalf("expected 1 override, got %d", len(rb.Type.Overrides))
	}
	if rb.Type.Overrides[0].Field != "items" || rb.Type.Overrides[0].TypeExpr != "[]CreateUserRequest" {
		t.Errorf("override = %+v", rb.Type.Overrides[0])
	}
}

func TestExtractor_MissingRouter_Warning(t *testing.T) {
	e := NewGoExtractor()

	ast := buildAST(
		// Has @Summary and @Tags but NO @Router → should warn.
		fn("HandlerNoRouter",
			"// @Summary Missing router",
			"// @Tags users",
		),
		// Has @Router → normal operation, no warning.
		fn("HandlerWithRouter",
			"// @Summary With router",
			"// @Router /ok [get]",
			"// @Success 200",
		),
		// No swagger annotations at all → no warning.
		fn("PlainFunction"),
	)

	result, err := e.Extract(ast)
	if err != nil {
		t.Fatal(err)
	}

	// One warning for HandlerNoRouter.
	warns := 0
	for _, d := range result.Diagnostics {
		if d.Level == DiagWarn {
			warns++
			if d.Message == "" {
				t.Error("warning message should not be empty")
			}
		}
	}
	if warns != 1 {
		t.Errorf("expected 1 warning, got %d (diagnostics: %+v)", warns, result.Diagnostics)
	}

	// HandlerWithRouter should be a real operation.
	if len(result.Operations) != 1 {
		t.Errorf("expected 1 operation, got %d", len(result.Operations))
	}
}

func TestExtractor_EmptyProject(t *testing.T) {
	e := NewGoExtractor()

	// No functions at all.
	result, err := e.Extract(&parser.RawAST{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Operations) != 0 {
		t.Errorf("expected 0 operations, got %d", len(result.Operations))
	}
	if len(result.Diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %+v", result.Diagnostics)
	}
}

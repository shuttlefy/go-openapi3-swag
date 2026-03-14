package builder

import (
	"strings"
	"testing"

	"github.com/shuttlefy/go-openapi3-swag/internal/extractor"
	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

func TestBuilder_BasicCRUD(t *testing.T) {
	rawAST := &parser.RawAST{
		Structs: []parser.RawStruct{
			{
				Name: "User",
				Fields: []parser.RawField{
					{Name: "ID", TypeName: "int64", JSONName: "id"},
					{Name: "Name", TypeName: "string", JSONName: "name", Required: true},
					{Name: "Email", TypeName: "string", JSONName: "email", Required: true},
				},
			},
			{
				Name: "CreateUserReq",
				Fields: []parser.RawField{
					{Name: "Name", TypeName: "string", JSONName: "name", Required: true},
					{Name: "Email", TypeName: "string", JSONName: "email", Required: true},
				},
			},
		},
	}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{
			Title:       "User API",
			Description: "User management",
			Version:     "1.0.0",
			Servers: []extractor.ServerAnnotation{
				{URL: "https://api.example.com", Description: "Production"},
			},
			Tags: []extractor.TagAnnotation{
				{Name: "users", Description: "User operations"},
			},
			SecurityDefs: []extractor.SecurityDefAnnotation{
				{
					Name:         "BearerAuth",
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
				},
			},
		},
		Operations: []extractor.OperationAnnotation{
			{
				FuncName:    "ListUsers",
				Tags:        []string{"users"},
				Summary:     "List all users",
				OperationID: "listUsers",
				Route:       extractor.RouteInfo{Method: "get", Path: "/users"},
				Params: []extractor.ParamAnnotation{
					{Name: "limit", In: "query", TypeName: "integer", Description: "max results"},
				},
				Responses: []extractor.ResponseAnnotation{
					{Code: "200", TypeName: "User", IsArray: true, Description: "success"},
				},
			},
			{
				FuncName:    "CreateUser",
				Tags:        []string{"users"},
				Summary:     "Create user",
				OperationID: "createUser",
				Route:       extractor.RouteInfo{Method: "post", Path: "/users"},
				RequestBody: &extractor.RequestBodyAnnotation{
					TypeName: "CreateUserReq",
					Type:     extractor.TypeExpr{Name: "CreateUserReq"},
					Required: true,
				},
				Responses: []extractor.ResponseAnnotation{
					{Code: "201", TypeName: "User", Description: "created"},
				},
			},
			{
				FuncName:    "GetUser",
				Tags:        []string{"users"},
				Summary:     "Get user by ID",
				OperationID: "getUser",
				Route:       extractor.RouteInfo{Method: "get", Path: "/users/{id}"},
				Params: []extractor.ParamAnnotation{
					{Name: "id", In: "path", TypeName: "integer", Required: true},
				},
				Responses: []extractor.ResponseAnnotation{
					{Code: "200", TypeName: "User", Description: "success"},
					{Code: "404", Description: "not found"},
				},
			},
			{
				FuncName:    "DeleteUser",
				Tags:        []string{"users"},
				Summary:     "Delete user",
				OperationID: "deleteUser",
				Route:       extractor.RouteInfo{Method: "delete", Path: "/users/{id}"},
				Params: []extractor.ParamAnnotation{
					{Name: "id", In: "path", TypeName: "integer", Required: true},
				},
				Responses: []extractor.ResponseAnnotation{
					{Code: "204", Description: "deleted"},
				},
				Security: []extractor.SecurityRequirement{
					{Name: "BearerAuth"},
				},
			},
		},
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	// Info
	if doc.Info.Title != "User API" {
		t.Errorf("Info.Title = %q", doc.Info.Title)
	}
	if doc.Info.Version != "1.0.0" {
		t.Errorf("Info.Version = %q", doc.Info.Version)
	}

	// Servers
	if len(doc.Servers) != 1 || doc.Servers[0].URL != "https://api.example.com" {
		t.Errorf("Servers = %v", doc.Servers)
	}

	// Tags
	if len(doc.Tags) != 1 || doc.Tags[0].Name != "users" {
		t.Errorf("Tags = %v", doc.Tags)
	}

	// Paths
	pathKeys := doc.Paths.Keys()
	if len(pathKeys) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(pathKeys), pathKeys)
	}

	usersPath := doc.Paths.Get("/users")
	if usersPath == nil {
		t.Fatal("missing /users path")
	}
	if usersPath.Get == nil {
		t.Error("missing GET /users")
	}
	if usersPath.Post == nil {
		t.Error("missing POST /users")
	}

	// GET /users
	listOp := usersPath.Get
	if listOp.OperationID != "listUsers" {
		t.Errorf("listUsers.OperationID = %q", listOp.OperationID)
	}
	if len(listOp.Parameters) != 1 {
		t.Errorf("listUsers params = %d", len(listOp.Parameters))
	} else {
		p := listOp.Parameters[0]
		if p.Name != "limit" || p.In != "query" {
			t.Errorf("param = %+v", p)
		}
	}
	resp200 := listOp.Responses.Get("200")
	if resp200 == nil {
		t.Fatal("missing 200 response")
	}
	if resp200.Content == nil {
		t.Fatal("missing response content")
	}
	mt := resp200.Content.Get("application/json")
	if mt == nil || mt.Schema == nil {
		t.Fatal("missing response schema")
	}
	if mt.Schema.Type != "array" {
		t.Errorf("response schema type = %q, want array", mt.Schema.Type)
	}
	if mt.Schema.Items == nil || mt.Schema.Items.Ref != "#/components/schemas/User" {
		t.Error("response items should $ref User")
	}

	// POST /users — request body
	createOp := usersPath.Post
	if createOp.RequestBody.Content.Keys() == nil {
		t.Fatal("missing request body content")
	}
	bodyMT := createOp.RequestBody.Content.Get("application/json")
	if bodyMT == nil || bodyMT.Schema == nil {
		t.Fatal("missing request body schema")
	}
	if bodyMT.Schema.Ref != "#/components/schemas/CreateUserReq" {
		t.Errorf("requestBody schema ref = %q", bodyMT.Schema.Ref)
	}
	if !createOp.RequestBody.Required {
		t.Error("requestBody.Required should be true")
	}

	// GET /users/{id}
	userByIdPath := doc.Paths.Get("/users/{id}")
	if userByIdPath == nil || userByIdPath.Get == nil {
		t.Fatal("missing GET /users/{id}")
	}
	getOp := userByIdPath.Get
	if len(getOp.Parameters) != 1 || getOp.Parameters[0].Name != "id" || !getOp.Parameters[0].Required {
		t.Error("GET /users/{id} param wrong")
	}
	resp404 := getOp.Responses.Get("404")
	if resp404 == nil {
		t.Error("missing 404 response")
	}

	// DELETE /users/{id} — security
	deleteOp := userByIdPath.Delete
	if deleteOp == nil {
		t.Fatal("missing DELETE /users/{id}")
	}
	if len(deleteOp.Security) != 1 {
		t.Fatalf("security len = %d", len(deleteOp.Security))
	}
	secScopes := deleteOp.Security[0].Get("BearerAuth")
	if secScopes == nil {
		t.Error("missing BearerAuth security requirement")
	}

	// Components — schemas
	if doc.Components == nil {
		t.Fatal("missing components")
	}
	if doc.Components.Schemas.Get("User") == nil {
		t.Error("missing User schema in components")
	}
	if doc.Components.Schemas.Get("CreateUserReq") == nil {
		t.Error("missing CreateUserReq schema in components")
	}

	// Components — securitySchemes
	bearerScheme := doc.Components.SecuritySchemes.Get("BearerAuth")
	if bearerScheme == nil {
		t.Fatal("missing BearerAuth security scheme")
	}
	if bearerScheme.Type != "http" || bearerScheme.Scheme != "bearer" {
		t.Errorf("BearerAuth = type=%q scheme=%q", bearerScheme.Type, bearerScheme.Scheme)
	}
	if bearerScheme.BearerFormat != "JWT" {
		t.Errorf("BearerFormat = %q", bearerScheme.BearerFormat)
	}
}

func TestBuilder_CompositeResponse(t *testing.T) {
	rawAST := &parser.RawAST{
		Structs: []parser.RawStruct{
			{
				Name: "PageData",
				Fields: []parser.RawField{
					{Name: "Code", TypeName: "int", JSONName: "code"},
					{Name: "Message", TypeName: "string", JSONName: "message"},
					{Name: "Data", TypeName: "interface{}", JSONName: "data"},
				},
			},
			{
				Name: "User",
				Fields: []parser.RawField{
					{Name: "ID", TypeName: "int64", JSONName: "id"},
				},
			},
		},
	}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{
			Title:   "Test",
			Version: "1.0.0",
		},
		Operations: []extractor.OperationAnnotation{
			{
				FuncName:    "ListUsers",
				OperationID: "listUsers",
				Route:       extractor.RouteInfo{Method: "get", Path: "/users"},
				Responses: []extractor.ResponseAnnotation{
					{
						Code:        "200",
						Description: "success",
						Type: extractor.TypeExpr{
							Name: "PageData",
							Overrides: []extractor.FieldOverride{
								{Field: "data", TypeExpr: "[]User"},
							},
						},
					},
				},
			},
		},
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	op := doc.Paths.Get("/users").Get
	resp := op.Responses.Get("200")
	mt := resp.Content.Get("application/json")
	schema := mt.Schema

	if len(schema.AllOf) != 2 {
		t.Fatalf("expected allOf, got %+v", schema)
	}
	if schema.AllOf[0].Ref != "#/components/schemas/PageData" {
		t.Errorf("allOf[0].$ref = %q", schema.AllOf[0].Ref)
	}

	dataProp := schema.AllOf[1].Properties.Get("data")
	if dataProp.Type != "array" {
		t.Errorf("data.Type = %q, want array", dataProp.Type)
	}
	if dataProp.Items.Ref != "#/components/schemas/User" {
		t.Errorf("data.Items.$ref = %q", dataProp.Items.Ref)
	}
}

func TestBuilder_FormDataRequestBody(t *testing.T) {
	rawAST := &parser.RawAST{}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{
			Title:   "Upload API",
			Version: "1.0.0",
		},
		Operations: []extractor.OperationAnnotation{
			{
				FuncName:    "Upload",
				OperationID: "upload",
				Route:       extractor.RouteInfo{Method: "post", Path: "/upload"},
				RequestBody: &extractor.RequestBodyAnnotation{
					IsForm:      true,
					Required:    true,
					Description: "file upload",
					Fields: []extractor.FormFieldAnnotation{
						{Name: "file", TypeName: "file", Required: true},
						{Name: "desc", TypeName: "string"},
					},
				},
				Responses: []extractor.ResponseAnnotation{
					{Code: "200", Description: "ok"},
				},
			},
		},
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	op := doc.Paths.Get("/upload").Post
	rb := op.RequestBody
	if !rb.Required {
		t.Error("RequestBody.Required should be true")
	}

	mt := rb.Content.Get("multipart/form-data")
	if mt == nil {
		t.Fatal("missing multipart/form-data content")
	}
	if mt.Schema.Type != "object" {
		t.Errorf("form schema type = %q", mt.Schema.Type)
	}
	fileProp := mt.Schema.Properties.Get("file")
	if fileProp == nil {
		t.Fatal("missing file property")
	}
	if fileProp.Type != "string" || fileProp.Format != "binary" {
		t.Errorf("file prop = type=%q format=%q", fileProp.Type, fileProp.Format)
	}
	if len(mt.Schema.Required) != 1 || mt.Schema.Required[0] != "file" {
		t.Errorf("form required = %v", mt.Schema.Required)
	}
}

func TestBuilder_ResponseHeaders(t *testing.T) {
	rawAST := &parser.RawAST{}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{Title: "API", Version: "1"},
		Operations: []extractor.OperationAnnotation{
			{
				Route: extractor.RouteInfo{Method: "get", Path: "/ping"},
				Responses: []extractor.ResponseAnnotation{
					{
						Code:        "200",
						TypeName:    "string",
						IsPrimitive: true,
						Description: "pong",
						Headers: []extractor.ResponseHeaderAnnotation{
							{Code: "200", Name: "X-Request-Id", TypeName: "string", Description: "req id"},
						},
					},
				},
			},
		},
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatal(err)
	}

	resp := doc.Paths.Get("/ping").Get.Responses.Get("200")
	if resp.Headers == nil {
		t.Fatal("missing headers")
	}
	h := resp.Headers.Get("X-Request-Id")
	if h == nil {
		t.Fatal("missing X-Request-Id header")
	}
	if h.Schema.Type != "string" {
		t.Errorf("header type = %q", h.Schema.Type)
	}
}

func TestBuilder_OAuth2SecurityScheme(t *testing.T) {
	rawAST := &parser.RawAST{}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{
			Title:   "API",
			Version: "1",
			SecurityDefs: []extractor.SecurityDefAnnotation{
				{
					Name:             "OAuth2",
					Type:             "oauth2",
					OAuthFlowType:    "authorizationCode",
					AuthorizationURL: "https://auth.example.com/authorize",
					TokenURL:         "https://auth.example.com/token",
					Scopes: map[string]string{
						"read":  "read access",
						"write": "write access",
					},
				},
			},
		},
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatal(err)
	}

	scheme := doc.Components.SecuritySchemes.Get("OAuth2")
	if scheme == nil {
		t.Fatal("missing OAuth2 scheme")
	}
	if scheme.Type != "oauth2" {
		t.Errorf("type = %q", scheme.Type)
	}
	flow := scheme.Flows.AuthorizationCode
	if flow.AuthorizationURL != "https://auth.example.com/authorize" {
		t.Errorf("authorizationURL = %q", flow.AuthorizationURL)
	}
	if flow.TokenURL != "https://auth.example.com/token" {
		t.Errorf("tokenURL = %q", flow.TokenURL)
	}

	readScope := flow.Scopes.Get("read")
	if readScope == nil || *readScope != "read access" {
		t.Errorf("read scope = %v", readScope)
	}
}

func TestBuilder_ServerFallback(t *testing.T) {
	rawAST := &parser.RawAST{}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{
			Title:    "API",
			Version:  "1",
			Host:     "api.example.com",
			BasePath: "/v2",
			Schemes:  []string{"https"},
		},
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatal(err)
	}

	if len(doc.Servers) != 1 {
		t.Fatalf("servers len = %d", len(doc.Servers))
	}
	if doc.Servers[0].URL != "https://api.example.com/v2" {
		t.Errorf("server URL = %q", doc.Servers[0].URL)
	}
}

func TestBuilder_PrimitiveResponse(t *testing.T) {
	rawAST := &parser.RawAST{}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{Title: "API", Version: "1"},
		Operations: []extractor.OperationAnnotation{
			{
				Route: extractor.RouteInfo{Method: "get", Path: "/health"},
				Responses: []extractor.ResponseAnnotation{
					{Code: "200", TypeName: "string", IsPrimitive: true, Description: "ok"},
				},
			},
		},
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatal(err)
	}

	resp := doc.Paths.Get("/health").Get.Responses.Get("200")
	mt := resp.Content.Get("application/json")
	if mt.Schema.Type != "string" {
		t.Errorf("schema type = %q, want string", mt.Schema.Type)
	}
}

func TestBuilder_DeprecatedOperation(t *testing.T) {
	rawAST := &parser.RawAST{}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{Title: "API", Version: "1"},
		Operations: []extractor.OperationAnnotation{
			{
				Route:      extractor.RouteInfo{Method: "get", Path: "/old"},
				Deprecated: true,
				Responses: []extractor.ResponseAnnotation{
					{Code: "200", Description: "ok"},
				},
			},
		},
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatal(err)
	}

	op := doc.Paths.Get("/old").Get
	if !op.Deprecated {
		t.Error("expected Deprecated=true")
	}
}

func TestBuilder_MultipleMethodsSamePath(t *testing.T) {
	rawAST := &parser.RawAST{}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{Title: "API", Version: "1"},
		Operations: []extractor.OperationAnnotation{
			{
				OperationID: "getItems",
				Route:       extractor.RouteInfo{Method: "get", Path: "/items"},
				Responses:   []extractor.ResponseAnnotation{{Code: "200", Description: "ok"}},
			},
			{
				OperationID: "createItem",
				Route:       extractor.RouteInfo{Method: "post", Path: "/items"},
				Responses:   []extractor.ResponseAnnotation{{Code: "201", Description: "created"}},
			},
			{
				OperationID: "updateItem",
				Route:       extractor.RouteInfo{Method: "put", Path: "/items"},
				Responses:   []extractor.ResponseAnnotation{{Code: "200", Description: "ok"}},
			},
		},
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatal(err)
	}

	item := doc.Paths.Get("/items")
	if item.Get == nil || item.Get.OperationID != "getItems" {
		t.Error("GET /items wrong")
	}
	if item.Post == nil || item.Post.OperationID != "createItem" {
		t.Error("POST /items wrong")
	}
	if item.Put == nil || item.Put.OperationID != "updateItem" {
		t.Error("PUT /items wrong")
	}
}

func TestBuilder_ExternalDocs(t *testing.T) {
	rawAST := &parser.RawAST{}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{
			Title:   "API",
			Version: "1",
			ExternalDocs: &extractor.ExternalDocsAnnotation{
				URL:         "https://docs.example.com",
				Description: "Full docs",
			},
		},
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatal(err)
	}

	if doc.ExternalDocs == nil {
		t.Fatal("missing externalDocs")
	}
	if doc.ExternalDocs.URL != "https://docs.example.com" {
		t.Errorf("externalDocs.URL = %q", doc.ExternalDocs.URL)
	}
}

func TestBuilder_ContactAndLicense(t *testing.T) {
	rawAST := &parser.RawAST{}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{
			Title:          "API",
			Version:        "1",
			TermsOfService: "https://example.com/tos",
			Contact: extractor.ContactAnnotation{
				Name:  "Support",
				Email: "support@example.com",
				URL:   "https://example.com/support",
			},
			License: extractor.LicenseAnnotation{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
		},
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatal(err)
	}

	if doc.Info.TermsOfService != "https://example.com/tos" {
		t.Errorf("termsOfService = %q", doc.Info.TermsOfService)
	}
	if doc.Info.Contact == nil || doc.Info.Contact.Name != "Support" {
		t.Error("contact wrong")
	}
	if doc.Info.License == nil || doc.Info.License.Name != "MIT" {
		t.Error("license wrong")
	}
}

func TestBuilder_AcceptProduce(t *testing.T) {
	rawAST := &parser.RawAST{
		Structs: []parser.RawStruct{
			{
				Name:   "Payload",
				Fields: []parser.RawField{{Name: "Data", TypeName: "string", JSONName: "data"}},
			},
		},
	}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{Title: "API", Version: "1"},
		Operations: []extractor.OperationAnnotation{
			{
				Route:  extractor.RouteInfo{Method: "post", Path: "/data"},
				Accept: []string{"application/xml", "application/json"},
				RequestBody: &extractor.RequestBodyAnnotation{
					TypeName: "Payload",
					Type:     extractor.TypeExpr{Name: "Payload"},
					Required: true,
				},
				Produce: []string{"application/xml"},
				Responses: []extractor.ResponseAnnotation{
					{Code: "200", TypeName: "Payload", Description: "ok"},
				},
			},
		},
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatal(err)
	}

	op := doc.Paths.Get("/data").Post
	rbKeys := op.RequestBody.Content.Keys()
	if len(rbKeys) != 2 {
		t.Errorf("requestBody content keys = %v", rbKeys)
	}

	respKeys := op.Responses.Get("200").Content.Keys()
	if len(respKeys) != 1 || respKeys[0] != "application/xml" {
		t.Errorf("response content keys = %v", respKeys)
	}
}

func TestBuilder_FuncScopeStructsExcluded(t *testing.T) {
	// Function-local structs whose owning function has NO operation annotation
	// must NOT appear in components.
	rawAST := &parser.RawAST{
		Structs: []parser.RawStruct{
			{
				Name:      "LocalType",
				FuncScope: "SomeFunc", // no operation for SomeFunc
				Fields:    []parser.RawField{{Name: "X", TypeName: "int"}},
			},
			{
				Name:   "TopLevel",
				Fields: []parser.RawField{{Name: "Y", TypeName: "string", JSONName: "y"}},
			},
		},
	}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{Title: "API", Version: "1"},
		// No operation whose FuncName == "SomeFunc"
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatal(err)
	}

	if doc.Components.Schemas.Get("LocalType") != nil {
		t.Error("FuncScope struct should not be registered when its function has no operation")
	}
	if doc.Components.Schemas.Get("TopLevel") == nil {
		t.Error("TopLevel struct should be registered")
	}
}

func TestBuilder_FuncLocalStructRegistered(t *testing.T) {
	// Function-local struct whose owning function HAS an operation annotation
	// MUST appear in components so the $ref resolves.
	rawAST := &parser.RawAST{
		Structs: []parser.RawStruct{
			{
				Name:      "SearchFilter",
				FuncScope: "SearchItems", // function with an operation
				Fields: []parser.RawField{
					{Name: "Keywords", TypeName: "[]string", JSONName: "keywords"},
					{Name: "Statuses", TypeName: "[]string", JSONName: "statuses"},
				},
			},
		},
	}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{Title: "API", Version: "1"},
		Operations: []extractor.OperationAnnotation{
			{
				FuncName: "SearchItems",
				Route:    extractor.RouteInfo{Method: "post", Path: "/search"},
				RequestBody: &extractor.RequestBodyAnnotation{
					TypeName: "SearchFilter",
					Type:     extractor.TypeExpr{Name: "SearchFilter"},
					Required: true,
				},
				Responses: []extractor.ResponseAnnotation{
					{Code: "200", Description: "ok"},
				},
			},
		},
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatal(err)
	}

	schema := doc.Components.Schemas.Get("SearchFilter")
	if schema == nil {
		t.Fatal("SearchFilter (function-local) should be registered when its function has an operation")
	}
	if schema.Properties.Get("keywords") == nil {
		t.Error("SearchFilter should have keywords property")
	}

	// Request body must reference it correctly.
	rb := doc.Paths.Get("/search").Post.RequestBody
	if !rb.Required {
		t.Error("requestBody should be required")
	}
	mt := rb.Content.Get("application/json")
	if mt == nil || mt.Schema.Ref != "#/components/schemas/SearchFilter" {
		t.Errorf("requestBody schema ref = %q", mt.Schema.Ref)
	}
}

func TestBuilder_EmptyProject(t *testing.T) {
	rawAST := &parser.RawAST{}
	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{
			Title:   "Empty API",
			Version: "0.0.1",
		},
	}

	b := NewBuilder()
	doc, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	if doc.Info.Title != "Empty API" {
		t.Errorf("info.title = %q", doc.Info.Title)
	}
	if doc.Info.Version != "0.0.1" {
		t.Errorf("info.version = %q", doc.Info.Version)
	}
	// No paths — Paths should be empty or nil.
	if doc.Paths != nil && len(doc.Paths.Keys()) != 0 {
		t.Errorf("expected no paths, got %v", doc.Paths.Keys())
	}
	// No warnings for empty project.
	if len(b.Warnings()) != 0 {
		t.Errorf("unexpected warnings: %v", b.Warnings())
	}
}

func TestBuilder_UnknownTypeWarning(t *testing.T) {
	rawAST := &parser.RawAST{
		Structs: []parser.RawStruct{
			{
				Name: "Order",
				Fields: []parser.RawField{
					{Name: "Item", TypeName: "GhostType", JSONName: "item"},
				},
			},
		},
	}
	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{Title: "API", Version: "1"},
	}

	b := NewBuilder()
	_, err := b.Build(result, rawAST)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	warns := b.Warnings()
	if len(warns) == 0 {
		t.Fatal("expected a warning for unknown type GhostType")
	}
	found := false
	for _, w := range warns {
		if strings.Contains(w, "GhostType") {
			found = true
		}
	}
	if !found {
		t.Errorf("warning should mention GhostType; got: %v", warns)
	}
}

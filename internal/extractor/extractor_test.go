package extractor

import (
	"testing"

	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

// ── parseCommentLines ─────────────────────────────────────────────────────────

func TestParseCommentLines(t *testing.T) {
	t.Run("mixed tags and plain lines", func(t *testing.T) {
		lines := []string{
			"This is a plain description.",
			"@Summary Get user",
			"@Tags users, admin",
			"",
			"@Router /users [get]",
		}
		tags, plain := parseCommentLines(lines)
		if len(tags) != 3 {
			t.Fatalf("tags len = %d, want 3", len(tags))
		}
		if tags[0].name != "summary" || tags[0].value != "Get user" {
			t.Errorf("tags[0] = %+v", tags[0])
		}
		if tags[2].name != "router" || tags[2].value != "/users [get]" {
			t.Errorf("tags[2] = %+v", tags[2])
		}
		if len(plain) != 1 || plain[0] != "This is a plain description." {
			t.Errorf("plain = %v", plain)
		}
	})

	t.Run("tag names are lowercased", func(t *testing.T) {
		tags, _ := parseCommentLines([]string{"@Summary text", "@ROUTER /x [get]", "@Tags A"})
		for _, tag := range tags {
			for _, ch := range tag.name {
				if ch >= 'A' && ch <= 'Z' {
					t.Errorf("tag name %q contains uppercase", tag.name)
				}
			}
		}
	})

	t.Run("tag with no value", func(t *testing.T) {
		tags, _ := parseCommentLines([]string{"@Deprecated"})
		if len(tags) != 1 || tags[0].name != "deprecated" || tags[0].value != "" {
			t.Errorf("got %+v", tags)
		}
	})

	t.Run("empty lines are skipped in plain", func(t *testing.T) {
		_, plain := parseCommentLines([]string{"", "  ", "hello"})
		if len(plain) != 1 || plain[0] != "hello" {
			t.Errorf("plain = %v", plain)
		}
	})
}

// ── extractLastQuoted ─────────────────────────────────────────────────────────

func TestExtractLastQuoted(t *testing.T) {
	cases := []struct {
		input        string
		wantQuoted   string
		wantRest     string
	}{
		{`id path int true "User ID"`, "User ID", "id path int true"},
		{`limit query integer false int32 "Page size"`, "Page size", "limit query integer false int32"},
		{`200 {object} models.User "OK"`, "OK", "200 {object} models.User"},
		{`204 "No Content"`, "No Content", "204"},
		{`no quotes here`, "", "no quotes here"},
		{`""`, "", ""},
	}
	for _, tc := range cases {
		q, r := extractLastQuoted(tc.input)
		if q != tc.wantQuoted || r != tc.wantRest {
			t.Errorf("extractLastQuoted(%q) = (%q, %q), want (%q, %q)",
				tc.input, q, r, tc.wantQuoted, tc.wantRest)
		}
	}
}

// ── parseMIMETypes ────────────────────────────────────────────────────────────

func TestParseMIMETypes(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"json", []string{"application/json"}},
		{"json, xml", []string{"application/json", "application/xml"}},
		{"mpfd", []string{"multipart/form-data"}},
		{"application/json", []string{"application/json"}}, // already full
		{"json, application/xml", []string{"application/json", "application/xml"}},
	}
	for _, tc := range cases {
		got := parseMIMETypes(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("parseMIMETypes(%q) len=%d, want %d: %v", tc.input, len(got), len(tc.want), got)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("parseMIMETypes(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}

// ── parseParamTag ─────────────────────────────────────────────────────────────

func TestParseParamTag(t *testing.T) {
	t.Run("path param no format", func(t *testing.T) {
		p, err := parseParamTag(`id path int true "User ID"`)
		if err != nil {
			t.Fatal(err)
		}
		if p.Name != "id" || p.In != "path" || p.TypeName != "int" {
			t.Errorf("got %+v", p)
		}
		if !p.Required {
			t.Error("Required should be true")
		}
		if p.Description != "User ID" {
			t.Errorf("Description = %q", p.Description)
		}
		if p.Format != "" {
			t.Errorf("Format should be empty, got %q", p.Format)
		}
	})

	t.Run("query param with format", func(t *testing.T) {
		p, err := parseParamTag(`limit query integer false int32 "Page size"`)
		if err != nil {
			t.Fatal(err)
		}
		if p.Name != "limit" || p.In != "query" || p.TypeName != "integer" {
			t.Errorf("got %+v", p)
		}
		if p.Required {
			t.Error("Required should be false")
		}
		if p.Format != "int32" {
			t.Errorf("Format = %q, want int32", p.Format)
		}
	})

	t.Run("body param with model type", func(t *testing.T) {
		p, err := parseParamTag(`body body models.CreateUserRequest true "Request body"`)
		if err != nil {
			t.Fatal(err)
		}
		if p.In != "body" || p.TypeName != "models.CreateUserRequest" {
			t.Errorf("got %+v", p)
		}
	})

	t.Run("date-time format", func(t *testing.T) {
		p, err := parseParamTag(`created_at query string false date-time "Creation date"`)
		if err != nil {
			t.Fatal(err)
		}
		if p.Format != "date-time" {
			t.Errorf("Format = %q, want date-time", p.Format)
		}
	})

	t.Run("in is lowercased", func(t *testing.T) {
		p, err := parseParamTag(`X-Token Header string true "Auth token"`)
		if err != nil {
			t.Fatal(err)
		}
		if p.In != "header" {
			t.Errorf("In = %q, want header", p.In)
		}
	})

	t.Run("too few fields returns error", func(t *testing.T) {
		_, err := parseParamTag(`id path`)
		if err == nil {
			t.Error("expected error for too few fields")
		}
	})
}

// ── parseResponseTag ──────────────────────────────────────────────────────────

func TestParseResponseTag(t *testing.T) {
	t.Run("object response", func(t *testing.T) {
		r, err := parseResponseTag(`200 {object} models.UserResponse "OK"`)
		if err != nil {
			t.Fatal(err)
		}
		if r.Code != "200" || r.WrapType != "object" || r.TypeName != "models.UserResponse" {
			t.Errorf("got %+v", r)
		}
		if r.IsArray {
			t.Error("IsArray should be false")
		}
		if r.Description != "OK" {
			t.Errorf("Description = %q", r.Description)
		}
	})

	t.Run("array response", func(t *testing.T) {
		r, err := parseResponseTag(`200 {array} []models.User "Users list"`)
		if err != nil {
			t.Fatal(err)
		}
		if r.WrapType != "array" || r.TypeName != "[]models.User" {
			t.Errorf("got %+v", r)
		}
		if !r.IsArray {
			t.Error("IsArray should be true")
		}
	})

	t.Run("no-body response (204)", func(t *testing.T) {
		r, err := parseResponseTag(`204 "No Content"`)
		if err != nil {
			t.Fatal(err)
		}
		if r.Code != "204" || r.WrapType != "" || r.TypeName != "" {
			t.Errorf("got %+v", r)
		}
		if r.Description != "No Content" {
			t.Errorf("Description = %q", r.Description)
		}
	})

	t.Run("composite type in response", func(t *testing.T) {
		r, err := parseResponseTag(`200 {object} common.PageData{data=[]models.User} "Paged"`)
		if err != nil {
			t.Fatal(err)
		}
		if r.TypeName != "common.PageData{data=[]models.User}" {
			t.Errorf("TypeName = %q", r.TypeName)
		}
	})

	t.Run("missing status code returns error", func(t *testing.T) {
		_, err := parseResponseTag(``)
		if err == nil {
			t.Error("expected error for empty value")
		}
	})
}

// ── parseRouterTag ────────────────────────────────────────────────────────────

func TestParseRouterTag(t *testing.T) {
	cases := []struct {
		input      string
		wantPath   string
		wantMethod string
		wantErr    bool
	}{
		{"/users [get]", "/users", "GET", false},
		{"/users/{id} [DELETE]", "/users/{id}", "DELETE", false},
		{"/pets/{petId}/photos [post]", "/pets/{petId}/photos", "POST", false},
		{"no brackets", "", "", true},
		{"/path []", "/path", "", false},
	}
	for _, tc := range cases {
		r, err := parseRouterTag(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseRouterTag(%q): expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseRouterTag(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if r.Path != tc.wantPath || r.Method != tc.wantMethod {
			t.Errorf("parseRouterTag(%q) = %+v, want {%q, %q}", tc.input, r, tc.wantPath, tc.wantMethod)
		}
	}
}

// ── parseSecurityTag ──────────────────────────────────────────────────────────

func TestParseSecurityTag(t *testing.T) {
	t.Run("no scopes", func(t *testing.T) {
		s := parseSecurityTag("ApiKeyAuth")
		if s.Name != "ApiKeyAuth" || len(s.Scopes) != 0 {
			t.Errorf("got %+v", s)
		}
	})

	t.Run("with scopes", func(t *testing.T) {
		s := parseSecurityTag("OAuth2[read, write]")
		if s.Name != "OAuth2" {
			t.Errorf("Name = %q, want OAuth2", s.Name)
		}
		if len(s.Scopes) != 2 || s.Scopes[0] != "read" || s.Scopes[1] != "write" {
			t.Errorf("Scopes = %v", s.Scopes)
		}
	})

	t.Run("single scope", func(t *testing.T) {
		s := parseSecurityTag("BearerAuth[admin]")
		if len(s.Scopes) != 1 || s.Scopes[0] != "admin" {
			t.Errorf("Scopes = %v", s.Scopes)
		}
	})
}

// ── parseHeaderTag ────────────────────────────────────────────────────────────

func TestParseHeaderTag(t *testing.T) {
	t.Run("string header", func(t *testing.T) {
		h, err := parseHeaderTag(`200 {string} X-Request-Id "Request tracking ID"`)
		if err != nil {
			t.Fatal(err)
		}
		if h.Code != "200" || h.TypeName != "string" || h.Name != "X-Request-Id" {
			t.Errorf("got %+v", h)
		}
		if h.Description != "Request tracking ID" {
			t.Errorf("Description = %q", h.Description)
		}
	})

	t.Run("integer header", func(t *testing.T) {
		h, err := parseHeaderTag(`200 {integer} X-RateLimit-Remaining "Remaining requests"`)
		if err != nil {
			t.Fatal(err)
		}
		if h.TypeName != "integer" || h.Name != "X-RateLimit-Remaining" {
			t.Errorf("got %+v", h)
		}
	})

	t.Run("too few fields returns error", func(t *testing.T) {
		_, err := parseHeaderTag(`200 {string}`)
		if err == nil {
			t.Error("expected error")
		}
	})
}

// ── parseTagDecl / parseServerTag ─────────────────────────────────────────────

func TestParseTagDecl(t *testing.T) {
	tag := parseTagDecl(`pets "Everything about your Pets"`)
	if tag.Name != "pets" || tag.Description != "Everything about your Pets" {
		t.Errorf("got %+v", tag)
	}
}

func TestParseServerTag(t *testing.T) {
	t.Run("with description", func(t *testing.T) {
		s := parseServerTag(`https://api.example.com "Production"`)
		if s.URL != "https://api.example.com" || s.Description != "Production" {
			t.Errorf("got %+v", s)
		}
	})

	t.Run("without description", func(t *testing.T) {
		s := parseServerTag("http://localhost:8080")
		if s.URL != "http://localhost:8080" || s.Description != "" {
			t.Errorf("got %+v", s)
		}
	})
}

// ── parseSecurityDefKey ───────────────────────────────────────────────────────

func TestParseSecurityDefKey(t *testing.T) {
	cases := []struct {
		input         string
		wantSchemeKey string
		wantAttr      string
	}{
		{"apikey", "apikey", ""},
		{"apikey.in", "apikey", "in"},
		{"apikey.name", "apikey", "name"},
		{"apikey.description", "apikey", "description"},
		{"basic", "basic", ""},
		{"bearer", "bearer", ""},
		{"bearer.bearerformat", "bearer", "bearerformat"},
		{"openidconnect", "openidconnect", ""},
		{"openidconnect.openidconnecturl", "openidconnect", "openidconnecturl"},
		{"oauth2.authorizationcode", "oauth2.authorizationcode", ""},
		{"oauth2.authorizationcode.authorizationurl", "oauth2.authorizationcode", "authorizationurl"},
		{"oauth2.authorizationcode.tokenurl", "oauth2.authorizationcode", "tokenurl"},
		{"oauth2.authorizationcode.scope.read", "oauth2.authorizationcode", "scope.read"},
		{"oauth2.implicit", "oauth2.implicit", ""},
		{"oauth2.password", "oauth2.password", ""},
		{"oauth2.clientcredentials", "oauth2.clientcredentials", ""},
		{"oauth2.unknown", "", ""}, // 未知 flow
	}
	for _, tc := range cases {
		sk, attr := parseSecurityDefKey(tc.input)
		if sk != tc.wantSchemeKey || attr != tc.wantAttr {
			t.Errorf("parseSecurityDefKey(%q) = (%q, %q), want (%q, %q)",
				tc.input, sk, attr, tc.wantSchemeKey, tc.wantAttr)
		}
	}
}

// ── secDefCtx（security definition 构建） ─────────────────────────────────────

func TestSecDefCtx_APIKey(t *testing.T) {
	ctx := newSecDefCtx()
	ctx.applyTag("apikey", "ApiKeyAuth")
	ctx.applyTag("apikey.in", "header")
	ctx.applyTag("apikey.name", "X-API-Key")
	ctx.applyTag("apikey.description", "API key auth")

	if len(ctx.defs) != 1 {
		t.Fatalf("defs len = %d, want 1", len(ctx.defs))
	}
	d := ctx.defs[0]
	if d.Name != "ApiKeyAuth" || d.Type != "apiKey" {
		t.Errorf("Name/Type = %q/%q", d.Name, d.Type)
	}
	if d.In != "header" || d.KeyName != "X-API-Key" {
		t.Errorf("In=%q KeyName=%q", d.In, d.KeyName)
	}
	if d.Description != "API key auth" {
		t.Errorf("Description = %q", d.Description)
	}
}

func TestSecDefCtx_Basic(t *testing.T) {
	ctx := newSecDefCtx()
	ctx.applyTag("basic", "BasicAuth")

	if len(ctx.defs) != 1 {
		t.Fatalf("defs len = %d, want 1", len(ctx.defs))
	}
	d := ctx.defs[0]
	if d.Name != "BasicAuth" || d.Type != "http" || d.Scheme != "basic" {
		t.Errorf("got %+v", d)
	}
}

func TestSecDefCtx_Bearer(t *testing.T) {
	ctx := newSecDefCtx()
	ctx.applyTag("bearer", "BearerAuth")
	ctx.applyTag("bearer.bearerformat", "JWT")

	d := ctx.defs[0]
	if d.Type != "http" || d.Scheme != "bearer" || d.BearerFormat != "JWT" {
		t.Errorf("got %+v", d)
	}
}

func TestSecDefCtx_OpenIDConnect(t *testing.T) {
	ctx := newSecDefCtx()
	ctx.applyTag("openidconnect", "MyOIDC")
	ctx.applyTag("openidconnect.openidconnecturl", "https://example.com/.well-known/openid-configuration")

	d := ctx.defs[0]
	if d.Type != "openIdConnect" || d.OpenIDConnectURL != "https://example.com/.well-known/openid-configuration" {
		t.Errorf("got %+v", d)
	}
}

func TestSecDefCtx_OAuth2AuthorizationCode(t *testing.T) {
	ctx := newSecDefCtx()
	ctx.applyTag("oauth2.authorizationcode", "OAuth2")
	ctx.applyTag("oauth2.authorizationcode.authorizationurl", "https://example.com/oauth/authorize")
	ctx.applyTag("oauth2.authorizationcode.tokenurl", "https://example.com/oauth/token")
	ctx.applyTag("oauth2.authorizationcode.scope.read", `"Read access"`)
	ctx.applyTag("oauth2.authorizationcode.scope.write", `"Write access"`)

	if len(ctx.defs) != 1 {
		t.Fatalf("defs len = %d, want 1", len(ctx.defs))
	}
	d := ctx.defs[0]
	if d.Name != "OAuth2" || d.Type != "oauth2" {
		t.Errorf("Name/Type = %q/%q", d.Name, d.Type)
	}
	if len(d.Flows) != 1 {
		t.Fatalf("Flows len = %d, want 1", len(d.Flows))
	}
	flow := d.Flows[0]
	if flow.Type != "authorizationCode" {
		t.Errorf("flow.Type = %q, want authorizationCode", flow.Type)
	}
	if flow.AuthorizationURL != "https://example.com/oauth/authorize" {
		t.Errorf("AuthorizationURL = %q", flow.AuthorizationURL)
	}
	if flow.TokenURL != "https://example.com/oauth/token" {
		t.Errorf("TokenURL = %q", flow.TokenURL)
	}
	if flow.Scopes["read"] != "Read access" || flow.Scopes["write"] != "Write access" {
		t.Errorf("Scopes = %v", flow.Scopes)
	}
}

func TestSecDefCtx_MultipleSchemes(t *testing.T) {
	ctx := newSecDefCtx()
	ctx.applyTag("apikey", "ApiKeyAuth")
	ctx.applyTag("apikey.in", "header")
	ctx.applyTag("bearer", "BearerAuth")
	ctx.applyTag("bearer.bearerformat", "JWT")

	if len(ctx.defs) != 2 {
		t.Fatalf("defs len = %d, want 2", len(ctx.defs))
	}
	if ctx.defs[0].Name != "ApiKeyAuth" || ctx.defs[1].Name != "BearerAuth" {
		t.Errorf("got %v", ctx.defs)
	}
}

// ── GoExtractor.Extract（集成） ───────────────────────────────────────────────

// makeFile 构造单文件 RawFile，pkg 为包名，fns 为函数列表。
func makeFile(pkg string, fns ...parser.RawFunc) *parser.RawFile {
	return &parser.RawFile{Package: pkg, FilePath: "/fake/" + pkg + ".go", Functions: fns}
}

// makeFunc 构造一个带注释的 RawFunc。
func makeFunc(name string, line int, comments ...string) parser.RawFunc {
	return parser.RawFunc{Name: name, FilePath: "/fake/file.go", Line: line, Comments: comments}
}

func TestExtract_GlobalAnnotation(t *testing.T) {
	fn := makeFunc("main", 1,
		"@title Pet Store API",
		"@version 2.0.0",
		"@description A sample pet store server.",
		"@termsOfService https://example.com/terms",
		"@contact.name API Support",
		"@contact.email support@example.com",
		"@contact.url https://www.example.com/support",
		"@license.name Apache 2.0",
		"@license.url https://www.apache.org/licenses/LICENSE-2.0.html",
		`@server https://api.petstore.com "Production"`,
		"@server http://localhost:8080",
		`@externalDocs.url https://example.com/docs`,
		`@externalDocs.description Find out more`,
		`@tag pets "Everything about Pets"`,
		`@tag users "User operations"`,
		"@host api.legacy.com",
		"@BasePath /api/v1",
		"@schemes http https",
	)

	result, err := (&GoExtractor{}).Extract([]*parser.RawFile{makeFile("main", fn)})
	if err != nil {
		t.Fatal(err)
	}

	g := result.Global
	if g.Title != "Pet Store API" {
		t.Errorf("Title = %q", g.Title)
	}
	if g.Version != "2.0.0" {
		t.Errorf("Version = %q", g.Version)
	}
	if g.Description != "A sample pet store server." {
		t.Errorf("Description = %q", g.Description)
	}
	if g.TermsOfService != "https://example.com/terms" {
		t.Errorf("TermsOfService = %q", g.TermsOfService)
	}
	if g.Contact.Name != "API Support" || g.Contact.Email != "support@example.com" {
		t.Errorf("Contact = %+v", g.Contact)
	}
	if g.License.Name != "Apache 2.0" {
		t.Errorf("License = %+v", g.License)
	}
	if len(g.Servers) != 2 || g.Servers[0].URL != "https://api.petstore.com" || g.Servers[0].Description != "Production" {
		t.Errorf("Servers = %+v", g.Servers)
	}
	if g.Servers[1].URL != "http://localhost:8080" || g.Servers[1].Description != "" {
		t.Errorf("Servers[1] = %+v", g.Servers[1])
	}
	if g.ExternalDocs.URL != "https://example.com/docs" || g.ExternalDocs.Description != "Find out more" {
		t.Errorf("ExternalDocs = %+v", g.ExternalDocs)
	}
	if len(g.Tags) != 2 || g.Tags[0].Name != "pets" || g.Tags[1].Name != "users" {
		t.Errorf("Tags = %+v", g.Tags)
	}
	if g.Host != "api.legacy.com" || g.BasePath != "/api/v1" {
		t.Errorf("Host/BasePath = %q/%q", g.Host, g.BasePath)
	}
	if len(g.Schemes) != 2 {
		t.Errorf("Schemes = %v", g.Schemes)
	}
}

func TestExtract_GlobalSecurityDefs(t *testing.T) {
	fn := makeFunc("main", 1,
		"@securityDefinitions.apikey ApiKeyAuth",
		"@securityDefinitions.apikey.in header",
		"@securityDefinitions.apikey.name X-API-Key",
		"@securityDefinitions.bearer BearerAuth",
		"@securityDefinitions.bearer.bearerFormat JWT",
		"@securityDefinitions.oauth2.authorizationCode OAuth2",
		"@securityDefinitions.oauth2.authorizationCode.authorizationUrl https://example.com/oauth/authorize",
		"@securityDefinitions.oauth2.authorizationCode.tokenUrl https://example.com/oauth/token",
		`@securityDefinitions.oauth2.authorizationCode.scope.read "Read access"`,
		`@securityDefinitions.oauth2.authorizationCode.scope.write "Write access"`,
	)

	result, err := (&GoExtractor{}).Extract([]*parser.RawFile{makeFile("main", fn)})
	if err != nil {
		t.Fatal(err)
	}

	defs := result.Global.SecurityDefs
	if len(defs) != 3 {
		t.Fatalf("SecurityDefs len = %d, want 3", len(defs))
	}

	apiKey := defs[0]
	if apiKey.Name != "ApiKeyAuth" || apiKey.Type != "apiKey" || apiKey.In != "header" || apiKey.KeyName != "X-API-Key" {
		t.Errorf("apiKey = %+v", apiKey)
	}

	bearer := defs[1]
	if bearer.Name != "BearerAuth" || bearer.Scheme != "bearer" || bearer.BearerFormat != "JWT" {
		t.Errorf("bearer = %+v", bearer)
	}

	oauth2 := defs[2]
	if oauth2.Name != "OAuth2" || oauth2.Type != "oauth2" || len(oauth2.Flows) != 1 {
		t.Errorf("oauth2 = %+v", oauth2)
	}
	flow := oauth2.Flows[0]
	if flow.Type != "authorizationCode" {
		t.Errorf("flow.Type = %q", flow.Type)
	}
	if flow.Scopes["read"] != "Read access" || flow.Scopes["write"] != "Write access" {
		t.Errorf("flow.Scopes = %v", flow.Scopes)
	}
}

func TestExtract_OperationBasic(t *testing.T) {
	fn := makeFunc("ListPets", 10,
		"ListPets returns all pets.",
		"@Summary List all pets",
		"@Description Get a paginated list of all pets",
		"@ID listPets",
		"@Tags pets",
		"@Accept json",
		"@Produce json, xml",
		`@Param limit query integer false int32 "Max items"`,
		`@Param offset query integer false int32 "Offset"`,
		`@Success 200 {object} common.PageData{data=[]models.Pet} "Paginated pets"`,
		`@Header 200 {string} X-Total-Count "Total number of pets"`,
		`@Failure 500 {object} models.Error "Server error"`,
		"@Security ApiKeyAuth",
		"@Router /pets [get]",
	)

	result, err := (&GoExtractor{}).Extract([]*parser.RawFile{makeFile("handlers", fn)})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Operations) != 1 {
		t.Fatalf("Operations len = %d, want 1", len(result.Operations))
	}
	op := result.Operations[0]

	if op.FuncName != "ListPets" {
		t.Errorf("FuncName = %q", op.FuncName)
	}
	if op.Summary != "List all pets" {
		t.Errorf("Summary = %q", op.Summary)
	}
	if op.Description != "Get a paginated list of all pets" {
		t.Errorf("Description = %q", op.Description)
	}
	if op.OperationID != "listPets" {
		t.Errorf("OperationID = %q", op.OperationID)
	}
	if len(op.Tags) != 1 || op.Tags[0] != "pets" {
		t.Errorf("Tags = %v", op.Tags)
	}
	if len(op.Accept) != 1 || op.Accept[0] != "application/json" {
		t.Errorf("Accept = %v", op.Accept)
	}
	if len(op.Produce) != 2 {
		t.Errorf("Produce = %v", op.Produce)
	}
	if op.Route.Method != "GET" || op.Route.Path != "/pets" {
		t.Errorf("Route = %+v", op.Route)
	}

	// Params
	if len(op.Params) != 2 {
		t.Fatalf("Params len = %d, want 2", len(op.Params))
	}
	if op.Params[0].Name != "limit" || op.Params[0].Format != "int32" {
		t.Errorf("Params[0] = %+v", op.Params[0])
	}

	// Responses
	if len(op.Responses) != 2 {
		t.Fatalf("Responses len = %d, want 2", len(op.Responses))
	}
	ok200 := op.Responses[0]
	if ok200.Code != "200" || ok200.WrapType != "object" || ok200.TypeName != "common.PageData{data=[]models.Pet}" {
		t.Errorf("Responses[0] = %+v", ok200)
	}

	// Headers
	if len(op.Headers) != 1 || op.Headers[0].Name != "X-Total-Count" {
		t.Errorf("Headers = %+v", op.Headers)
	}

	// Security
	if len(op.Security) != 1 || op.Security[0].Name != "ApiKeyAuth" {
		t.Errorf("Security = %+v", op.Security)
	}
}

func TestExtract_OperationDeprecated(t *testing.T) {
	fn := makeFunc("DeletePet", 20,
		"@Summary Delete a pet",
		"@Tags pets",
		`@Param id path integer true int64 "Pet ID"`,
		`@Success 204 "No Content"`,
		`@Failure 404 {object} models.Error "Not found"`,
		"@Deprecated",
		"@Security ApiKeyAuth",
		"@Router /pets/{id} [delete]",
	)

	result, err := (&GoExtractor{}).Extract([]*parser.RawFile{makeFile("handlers", fn)})
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if !op.Deprecated {
		t.Error("Deprecated should be true")
	}
	if op.Route.Method != "DELETE" || op.Route.Path != "/pets/{id}" {
		t.Errorf("Route = %+v", op.Route)
	}
	if op.Params[0].Format != "int64" {
		t.Errorf("Params[0].Format = %q", op.Params[0].Format)
	}
	// 204 no-body response
	if op.Responses[0].WrapType != "" || op.Responses[0].TypeName != "" {
		t.Errorf("204 response should have no body: %+v", op.Responses[0])
	}
}

func TestExtract_OperationFormData(t *testing.T) {
	fn := makeFunc("UploadPhoto", 30,
		"@Summary Upload pet photo",
		"@Tags pets",
		"@Accept mpfd",
		"@Produce json",
		`@Param id path integer true "Pet ID"`,
		`@Param file formData file true "Photo file"`,
		`@Param caption formData string false "Photo caption"`,
		`@Success 200 {object} models.UploadResult "OK"`,
		"@Router /pets/{id}/photos [post]",
	)

	result, err := (&GoExtractor{}).Extract([]*parser.RawFile{makeFile("handlers", fn)})
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if op.Accept[0] != "multipart/form-data" {
		t.Errorf("Accept = %v", op.Accept)
	}
	if len(op.Params) != 3 {
		t.Fatalf("Params len = %d, want 3", len(op.Params))
	}
	if op.Params[1].In != "formdata" || op.Params[1].TypeName != "file" {
		t.Errorf("Params[1] = %+v", op.Params[1])
	}
}

func TestExtract_OperationDescriptionFallback(t *testing.T) {
	// 没有 @Description，普通注释行应成为 Description
	fn := makeFunc("GetUser", 5,
		"GetUser returns a user by ID.",
		"This is the second line.",
		"@Summary Get user",
		"@Router /users/{id} [get]",
	)

	result, err := (&GoExtractor{}).Extract([]*parser.RawFile{makeFile("handlers", fn)})
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if op.Description != "GetUser returns a user by ID.\nThis is the second line." {
		t.Errorf("Description = %q", op.Description)
	}
}

func TestExtract_OperationDescriptionExplicit(t *testing.T) {
	// 有 @Description 时，不使用普通注释行
	fn := makeFunc("GetUser", 5,
		"This plain line should be ignored.",
		"@Summary Get user",
		"@Description Explicit description.",
		"@Router /users/{id} [get]",
	)

	result, err := (&GoExtractor{}).Extract([]*parser.RawFile{makeFile("handlers", fn)})
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if op.Description != "Explicit description." {
		t.Errorf("Description = %q", op.Description)
	}
}

func TestExtract_OperationSecurityWithScopes(t *testing.T) {
	fn := makeFunc("CreatePet", 15,
		"@Summary Create a pet",
		"@Security OAuth2[write]",
		"@Security ApiKeyAuth",
		"@Router /pets [post]",
	)

	result, err := (&GoExtractor{}).Extract([]*parser.RawFile{makeFile("handlers", fn)})
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if len(op.Security) != 2 {
		t.Fatalf("Security len = %d, want 2", len(op.Security))
	}
	if op.Security[0].Name != "OAuth2" || len(op.Security[0].Scopes) != 1 || op.Security[0].Scopes[0] != "write" {
		t.Errorf("Security[0] = %+v", op.Security[0])
	}
	if op.Security[1].Name != "ApiKeyAuth" || len(op.Security[1].Scopes) != 0 {
		t.Errorf("Security[1] = %+v", op.Security[1])
	}
}

func TestExtract_NoRouterIgnored(t *testing.T) {
	// 没有 @Router 的函数不应产生 OperationAnnotation
	fn := makeFunc("helperFunc", 5,
		"@Summary Should be ignored",
		"@Tags internal",
	)

	result, err := (&GoExtractor{}).Extract([]*parser.RawFile{makeFile("handlers", fn)})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Operations) != 0 {
		t.Errorf("expected 0 operations, got %d", len(result.Operations))
	}
}

func TestExtract_NoComments(t *testing.T) {
	fn := parser.RawFunc{Name: "silent", FilePath: "/f.go", Line: 1}

	result, err := (&GoExtractor{}).Extract([]*parser.RawFile{makeFile("pkg", fn)})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Operations) != 0 {
		t.Errorf("expected 0 operations")
	}
}

func TestExtract_MultipleFilesAndFunctions(t *testing.T) {
	// 全局注解在第一个文件，操作注解分布在两个文件
	mainFn := makeFunc("main", 1,
		"@title My API",
		"@version 1.0.0",
	)
	op1 := makeFunc("CreateUser", 10,
		"@Summary Create user",
		"@Router /users [post]",
	)
	op2 := makeFunc("ListOrders", 20,
		"@Summary List orders",
		"@Router /orders [get]",
	)

	files := []*parser.RawFile{
		makeFile("main", mainFn),
		makeFile("handlers", op1),
		makeFile("handlers", op2),
	}

	result, err := (&GoExtractor{}).Extract(files)
	if err != nil {
		t.Fatal(err)
	}

	if result.Global.Title != "My API" || result.Global.Version != "1.0.0" {
		t.Errorf("Global = %+v", result.Global)
	}
	if len(result.Operations) != 2 {
		t.Fatalf("Operations len = %d, want 2", len(result.Operations))
	}
	if result.Operations[0].Route.Path != "/users" || result.Operations[1].Route.Path != "/orders" {
		t.Errorf("routes = %v %v", result.Operations[0].Route, result.Operations[1].Route)
	}
}

func TestExtract_TagsCaseInsensitive(t *testing.T) {
	// 标签名不区分大小写
	fn := makeFunc("Handler", 5,
		"@SUMMARY Upper case",
		"@Tags Users",
		"@ROUTER /x [GET]",
	)

	result, err := (&GoExtractor{}).Extract([]*parser.RawFile{makeFile("h", fn)})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Operations) != 1 {
		t.Fatalf("expected 1 operation")
	}
	op := result.Operations[0]
	if op.Summary != "Upper case" {
		t.Errorf("Summary = %q", op.Summary)
	}
	if op.Route.Method != "GET" {
		t.Errorf("Method = %q", op.Route.Method)
	}
}

func TestExtract_MultipleTagsOnOperation(t *testing.T) {
	fn := makeFunc("Handler", 5,
		"@Tags users, admin, internal",
		"@Router /x [get]",
	)

	result, err := (&GoExtractor{}).Extract([]*parser.RawFile{makeFile("h", fn)})
	if err != nil {
		t.Fatal(err)
	}

	op := result.Operations[0]
	if len(op.Tags) != 3 || op.Tags[0] != "users" || op.Tags[1] != "admin" || op.Tags[2] != "internal" {
		t.Errorf("Tags = %v", op.Tags)
	}
}

func TestExtract_EmptyFiles(t *testing.T) {
	result, err := (&GoExtractor{}).Extract([]*parser.RawFile{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Operations) != 0 {
		t.Error("expected 0 operations for empty input")
	}
}

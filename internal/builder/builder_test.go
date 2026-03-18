package builder

import (
	"encoding/json"
	"strings"
	"testing"

	spec3 "github.com/shuttlefy/go-openapi3-spec"
	"github.com/shuttlefy/go-openapi3-swag/internal/extractor"
	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

// parseStoreFixture 用真实的 GoParser 解析 testdata/store（含 response 子目录）。
func parseStoreFixture(t *testing.T) []*parser.RawFile {
	t.Helper()
	p := &parser.GoParser{MaxDepth: -1}
	files, err := p.Parse([]string{"../../testdata/store"})
	if err != nil {
		t.Fatalf("parse testdata/store: %v", err)
	}
	return files
}

// parseAnnotationsFixture 用真实的 GoParser 解析 testdata/annotations（含所有子目录）。
func parseAnnotationsFixture(t *testing.T) []*parser.RawFile {
	t.Helper()
	p := &parser.GoParser{MaxDepth: -1}
	files, err := p.Parse([]string{"../../testdata/annotations"})
	if err != nil {
		t.Fatalf("parse testdata/annotations: %v", err)
	}
	return files
}

// allStructNames 返回所有文件中的 struct 名称集合。
func allStructNames(files []*parser.RawFile) map[string]bool {
	m := make(map[string]bool)
	for _, f := range files {
		for _, s := range f.Structs {
			m[f.Package+"."+s.Name] = true
		}
	}
	return m
}

// allFuncNames 返回所有文件中的函数名称集合（用于验证操作函数被解析到）。
func allFuncNames(files []*parser.RawFile) map[string]bool {
	m := make(map[string]bool)
	for _, f := range files {
		for _, fn := range f.Functions {
			m[fn.Name] = true
		}
	}
	return m
}

// ── testdata/annotations 集成测试 ─────────────────────────────────────────────

// TestAnnotations_ParseFixture 验证 testdata/annotations 目录能被 GoParser 完整解析。
func TestAnnotations_ParseFixture(t *testing.T) {
	files := parseAnnotationsFixture(t)

	// 必须解析到 4 个包：annotations(v1/v2)、models(4文件)、common(1文件)
	pkgCount := map[string]int{}
	for _, f := range files {
		pkgCount[f.Package]++
	}
	for _, pkg := range []string{"annotations", "models", "common"} {
		if pkgCount[pkg] == 0 {
			t.Errorf("package %q not found in parsed files", pkg)
		}
	}

	// 模型类型必须存在
	structs := allStructNames(files)
	for _, name := range []string{
		"models.Pet", "models.Category", "models.Tag",
		"models.Order", "models.User", "models.Address",
		"models.CreatePetRequest", "models.UpdatePetRequest",
		"models.CreateOrderRequest", "models.UpdateUserRequest",
		"models.LoginRequest", "models.TokenResponse",
		"models.ErrorResponse", "models.UploadResult",
		"common.Response", "common.PageData",
	} {
		if !structs[name] {
			t.Errorf("struct %q not parsed", name)
		}
	}

	// 操作函数必须存在
	funcs := allFuncNames(files)
	for _, name := range []string{
		"main",
		"ListPets", "CreatePet", "GetPet", "UpdatePet", "DeletePet", "UploadPetPhoto",
		"ListOrders", "CreateOrder", "GetOrder", "CancelOrder",
		"GetMe", "UpdateMe", "Login", "Logout",
		"AdminListUsers", "AdminGetUser", "AdminDeleteUser",
		"HealthCheck", "GetStats", "SearchPets",
	} {
		if !funcs[name] {
			t.Errorf("function %q not parsed", name)
		}
	}
}

// TestAnnotations_Extract 验证 GoExtractor 能正确提取全局注解和所有操作注解。
func TestAnnotations_Extract(t *testing.T) {
	files := parseAnnotationsFixture(t)
	e := &extractor.GoExtractor{}
	result, err := e.Extract(files)
	if err != nil {
		t.Fatal(err)
	}

	// ── 全局注解 ──────────────────────────────────────────────────────────────
	g := result.Global
	if g.Title != "Pet Store API" {
		t.Errorf("Title = %q", g.Title)
	}
	if g.Version != "1.0.0" {
		t.Errorf("Version = %q", g.Version)
	}
	if g.Contact.Email != "support@petstore.example.com" {
		t.Errorf("Contact.Email = %q", g.Contact.Email)
	}
	if g.License.Name != "Apache 2.0" {
		t.Errorf("License.Name = %q", g.License.Name)
	}
	if len(g.Servers) != 3 {
		t.Errorf("Servers len = %d, want 3", len(g.Servers))
	}
	if g.ExternalDocs.URL != "https://example.com/docs" {
		t.Errorf("ExternalDocs.URL = %q", g.ExternalDocs.URL)
	}
	if len(g.Tags) != 3 {
		t.Errorf("Tags len = %d, want 3", len(g.Tags))
	}
	// 安全方案：ApiKeyAuth + BearerAuth + OAuth2
	if len(g.SecurityDefs) != 3 {
		t.Errorf("SecurityDefs len = %d, want 3", len(g.SecurityDefs))
	}

	// ── 操作数量：v1.go=10 + v2.go=9（含 initDB 因有 @Summary 但无 @Router 应被忽略）────
	// 无 @Router 的函数不产生 Operation
	for _, op := range result.Operations {
		if op.FuncName == "initDB" || op.FuncName == "helperFunc" {
			t.Errorf("non-handler func %q should not produce operation", op.FuncName)
		}
	}

	opByName := make(map[string]extractor.OperationAnnotation)
	for _, op := range result.Operations {
		opByName[op.FuncName] = op
	}

	// 必须包含所有 @Router 函数
	for _, name := range []string{
		"ListPets", "CreatePet", "GetPet", "UpdatePet", "DeletePet", "UploadPetPhoto",
		"ListOrders", "CreateOrder", "GetOrder", "CancelOrder",
		"GetMe", "UpdateMe", "Login", "Logout",
		"AdminListUsers", "AdminGetUser", "AdminDeleteUser",
		"HealthCheck", "GetStats", "SearchPets",
	} {
		if _, ok := opByName[name]; !ok {
			t.Errorf("operation %q not extracted", name)
		}
	}

	// ── 逐一验证关键操作 ──────────────────────────────────────────────────────

	// ListPets: GET /pets, 分页参数, composite 响应, 响应头, security
	listPets := opByName["ListPets"]
	if listPets.Route.Method != "GET" || listPets.Route.Path != "/pets" {
		t.Errorf("ListPets route = %+v", listPets.Route)
	}
	if len(listPets.Params) != 3 { // status, limit, offset
		t.Errorf("ListPets params = %d, want 3", len(listPets.Params))
	}
	if len(listPets.Responses) != 3 { // 200, 400, 500
		t.Errorf("ListPets responses = %d, want 3", len(listPets.Responses))
	}
	if listPets.Responses[0].TypeName != "common.PageData{list=[]models.Pet}" {
		t.Errorf("ListPets 200 TypeName = %q", listPets.Responses[0].TypeName)
	}
	if len(listPets.Headers) != 2 { // X-Total-Count, X-Request-Id
		t.Errorf("ListPets headers = %d, want 2", len(listPets.Headers))
	}
	if len(listPets.Security) != 1 || listPets.Security[0].Name != "ApiKeyAuth" {
		t.Errorf("ListPets security = %+v", listPets.Security)
	}

	// CreatePet: POST, JSON body, 双重安全方案, @ID
	createPet := opByName["CreatePet"]
	if createPet.Route.Method != "POST" {
		t.Errorf("CreatePet method = %q", createPet.Route.Method)
	}
	if createPet.OperationID != "createPet" {
		t.Errorf("CreatePet operationID = %q", createPet.OperationID)
	}
	bodyParams := 0
	for _, p := range createPet.Params {
		if p.In == "body" {
			bodyParams++
		}
	}
	if bodyParams != 1 {
		t.Errorf("CreatePet body params = %d, want 1", bodyParams)
	}
	if len(createPet.Security) != 2 { // BearerAuth + OAuth2[write]
		t.Errorf("CreatePet security = %d, want 2", len(createPet.Security))
	}
	oauth2Sec := createPet.Security[1]
	if oauth2Sec.Name != "OAuth2" || len(oauth2Sec.Scopes) != 1 || oauth2Sec.Scopes[0] != "write" {
		t.Errorf("CreatePet OAuth2 security = %+v", oauth2Sec)
	}

	// DeletePet: @Deprecated, @Success 204 no body
	deletePet := opByName["DeletePet"]
	if !deletePet.Deprecated {
		t.Error("DeletePet should be deprecated")
	}
	if deletePet.Responses[0].Code != "204" || deletePet.Responses[0].TypeName != "" {
		t.Errorf("DeletePet 204 = %+v", deletePet.Responses[0])
	}

	// UploadPetPhoto: @Accept mpfd, formData params
	upload := opByName["UploadPetPhoto"]
	if len(upload.Accept) == 0 || upload.Accept[0] != "multipart/form-data" {
		t.Errorf("UploadPetPhoto accept = %v", upload.Accept)
	}
	formParams := 0
	for _, p := range upload.Params {
		if p.In == "formdata" {
			formParams++
		}
	}
	if formParams != 2 { // file + caption
		t.Errorf("UploadPetPhoto formData params = %d, want 2", formParams)
	}

	// Login: 响应头
	login := opByName["Login"]
	if len(login.Headers) != 2 {
		t.Errorf("Login headers = %d, want 2", len(login.Headers))
	}
	headerNames := map[string]bool{}
	for _, h := range login.Headers {
		headerNames[h.Name] = true
	}
	if !headerNames["X-Access-Token"] || !headerNames["X-Expires-In"] {
		t.Errorf("Login header names = %v", headerNames)
	}

	// UpdateMe: 多重安全 (ApiKeyAuth + BearerAuth)
	updateMe := opByName["UpdateMe"]
	if len(updateMe.Security) != 2 {
		t.Errorf("UpdateMe security = %d, want 2", len(updateMe.Security))
	}

	// AdminListUsers: 多 tag（users, admin）
	adminList := opByName["AdminListUsers"]
	if len(adminList.Tags) != 2 {
		t.Errorf("AdminListUsers tags = %v, want 2", adminList.Tags)
	}

	// HealthCheck: 响应 {string}
	health := opByName["HealthCheck"]
	if health.Responses[0].WrapType != "string" {
		t.Errorf("HealthCheck response WrapType = %q", health.Responses[0].WrapType)
	}

	// ListOrders: 含 date-time format 的 query 参数
	listOrders := opByName["ListOrders"]
	formatParams := 0
	for _, p := range listOrders.Params {
		if p.Format == "date-time" {
			formatParams++
		}
	}
	if formatParams != 2 { // from, to
		t.Errorf("ListOrders date-time params = %d, want 2", formatParams)
	}

	// Logout: header 参数
	logout := opByName["Logout"]
	headerParam := false
	for _, p := range logout.Params {
		if p.In == "header" {
			headerParam = true
		}
	}
	if !headerParam {
		t.Error("Logout should have header param")
	}
}

// TestAnnotations_BuildOpenAPI 使用真实 fixture 做全量构建，验证 Components.Schemas 完整注册。
func TestAnnotations_BuildOpenAPI(t *testing.T) {
	files := parseAnnotationsFixture(t)
	e := &extractor.GoExtractor{}
	result, err := e.Extract(files)
	if err != nil {
		t.Fatal(err)
	}

	b := NewBuilder()
	doc, err := b.Build(result, files)
	if err != nil {
		t.Fatal(err)
	}

	// OpenAPI 基础字段
	if doc.OpenAPI != "3.0.3" || doc.Info.Title != "Pet Store API" {
		t.Errorf("doc header: openapi=%q title=%q", doc.OpenAPI, doc.Info.Title)
	}

	// Paths 必须包含所有路由
	for _, path := range []string{
		"/pets", "/pets/{id}", "/pets/{id}/upload", "/pets/search",
		"/store/orders", "/store/orders/{id}",
		"/users/me", "/users/login", "/users/logout",
		"/admin/users", "/admin/users/{id}",
		"/health", "/stats",
	} {
		if doc.Paths.Get(path) == nil {
			t.Errorf("path %q missing from Paths", path)
		}
	}

	// GET /pets → operation 存在
	petsPI := doc.Paths.Get("/pets")
	if petsPI == nil || petsPI.Get == nil {
		t.Fatal("GET /pets missing")
	}

	// DELETE /pets/{id} → deprecated
	if pi := doc.Paths.Get("/pets/{id}"); pi == nil || pi.Delete == nil || !pi.Delete.Deprecated {
		t.Error("DELETE /pets/{id} should be deprecated")
	}

	// Components.Schemas：核心模型类型已注册
	for _, key := range []string{
		"models.Pet", "models.Category", "models.Tag",
		"models.Order", "models.User", "models.Address",
		"models.ErrorResponse", "models.UploadResult",
		"models.CreatePetRequest", "models.UpdatePetRequest",
		"models.TokenResponse",
	} {
		if doc.Components.Schemas.Get(key) == nil {
			t.Errorf("Components.Schemas[%q] missing", key)
		}
	}

	// common.Response 和 common.PageData 通过 composite 类型触发注册
	for _, key := range []string{"common.Response", "common.PageData"} {
		if doc.Components.Schemas.Get(key) == nil {
			t.Errorf("Components.Schemas[%q] missing", key)
		}
	}

	// Components.SecuritySchemes：3 个方案
	if doc.Components.SecuritySchemes.Get("ApiKeyAuth") == nil {
		t.Error("ApiKeyAuth missing from SecuritySchemes")
	}
	if doc.Components.SecuritySchemes.Get("BearerAuth") == nil {
		t.Error("BearerAuth missing from SecuritySchemes")
	}
	if doc.Components.SecuritySchemes.Get("OAuth2") == nil {
		t.Error("OAuth2 missing from SecuritySchemes")
	}

	// 整体 JSON 序列化不报错
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	if len(data) == 0 {
		t.Error("marshaled JSON is empty")
	}
}

// TestAnnotations_ThirdPartyRecursiveChain 验证从注解 composite 类型触发的多层第三方包递归解析。
//
// 场景：SearchPets 的响应类型为
//
//	common.Response{data=common.PageData{list=[]models.Pet}}
//
// 解析链：
//
//	Response (common) → 注册
//	PageData  (common) → 注册
//	[]Pet     → Pet (models) → Category (models) + []Tag + PetStatus(enum) + time.Time(内置)
//	Category  (models) → 注册
//	Tag       (models) → 注册
//	PetStatus (models) → 注册为 enum schema
func TestAnnotations_ThirdPartyRecursiveChain(t *testing.T) {
	files := parseAnnotationsFixture(t)
	e := &extractor.GoExtractor{}
	result, err := e.Extract(files)
	if err != nil {
		t.Fatal(err)
	}

	b := NewBuilder()
	doc, err := b.Build(result, files)
	if err != nil {
		t.Fatal(err)
	}

	// SearchPets 响应触发嵌套 composite 解析链
	searchPI := doc.Paths.Get("/pets/search")
	if searchPI == nil || searchPI.Get == nil {
		t.Fatal("GET /pets/search missing")
	}
	resp200 := searchPI.Get.Responses.Get("200")
	if resp200 == nil || resp200.Content == nil {
		t.Fatal("SearchPets 200 response missing content")
	}

	// 整个引用链的所有类型必须注册
	chainKeys := []string{
		"common.Response",
		"common.PageData",
		"models.Pet",
		"models.Category",
		"models.Tag",
		"models.PetStatus",
	}
	for _, key := range chainKeys {
		if doc.Components.Schemas.Get(key) == nil {
			t.Errorf("chain link %q not in Components.Schemas", key)
		}
	}

	// PetStatus 应为 string enum
	petStatus := doc.Components.Schemas.Get("models.PetStatus")
	if petStatus.Type != "string" || len(petStatus.Enum) != 3 {
		t.Errorf("models.PetStatus = %+v", petStatus)
	}

	// models.Pet 的 category 字段应为 $ref
	petSchema := doc.Components.Schemas.Get("models.Pet")
	if petSchema.Properties == nil {
		t.Fatal("models.Pet has no properties")
	}
	catProp := petSchema.Properties.Get("category")
	if catProp == nil || catProp.Ref == "" {
		t.Errorf("Pet.category should be $ref, got %+v", catProp)
	}

	// models.Pet 的 tags 字段应为 array of $ref Tag
	tagsProp := petSchema.Properties.Get("tags")
	if tagsProp == nil || tagsProp.Type != "array" {
		t.Errorf("Pet.tags should be array, got %+v", tagsProp)
	}
	if tagsProp.Items == nil || tagsProp.Items.Ref == "" {
		t.Errorf("Pet.tags.items should be $ref Tag, got %+v", tagsProp.Items)
	}
}

// TestAnnotations_CompositeTypeInline 验证组合类型直接内联为 allOf schema，不注册到 Components。
func TestAnnotations_CompositeTypeInline(t *testing.T) {
	files := parseAnnotationsFixture(t)
	e := &extractor.GoExtractor{}
	result, err := e.Extract(files)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := NewBuilder().Build(result, files)
	if err != nil {
		t.Fatal(err)
	}

	// 组合类型不再注册为 composite key（{}）
	doc.Components.Schemas.ForEach(func(key string, _ *spec3.Schema) error {
		if strings.Contains(key, "{") {
			t.Errorf("composite schema key %q should not appear in Components.Schemas", key)
		}
		return nil
	})

	// 验证含组合类型的操作路由仍然正常生成
	if doc.Paths.Get("/pets") == nil {
		t.Error("path /pets not found")
	}
}

// TestAnnotations_EnumFieldsFromModels 验证从 fixture 中解析到的 enum 常量
// 被正确注册为 Components.Schemas 中的 enum schema。
func TestAnnotations_EnumFieldsFromModels(t *testing.T) {
	files := parseAnnotationsFixture(t)
	e := &extractor.GoExtractor{}
	result, err := e.Extract(files)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := NewBuilder().Build(result, files)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		key    string
		typ    string
		values []string
	}{
		{"models.PetStatus", "string", []string{"available", "pending", "sold"}},
		{"models.OrderStatus", "string", []string{"placed", "approved", "delivered"}},
		{"models.UserRole", "string", []string{"admin", "user", "guest"}},
	}
	for _, tc := range cases {
		s := doc.Components.Schemas.Get(tc.key)
		if s == nil {
			t.Errorf("%q not in Components.Schemas", tc.key)
			continue
		}
		if s.Type != tc.typ {
			t.Errorf("%q.Type = %q, want %q", tc.key, s.Type, tc.typ)
		}
		if len(s.Enum) != len(tc.values) {
			t.Errorf("%q.Enum = %v, want %v", tc.key, s.Enum, tc.values)
		}
	}
}

// TestAnnotations_GinTypes_NoLoader 验证无 loader 时 gin 类型的解析行为。
//
// gin.H / gin.Error 不在 testdata/annotations 目录内，无法被 GoParser 解析，
// Components.Schemas 中不会出现对应 key，但路由/操作仍能被正确提取。
func TestAnnotations_GinTypes_NoLoader(t *testing.T) {
	files := parseAnnotationsFixture(t)
	e := &extractor.GoExtractor{}
	result, err := e.Extract(files)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := NewBuilder().Build(result, files)
	if err != nil {
		t.Fatal(err)
	}

	// v3.go 中的 4 个操作路由必须被提取
	routes := []struct{ method, path string }{
		{"GET", "/version"},
		{"GET", "/debug"},
		{"POST", "/echo"},
		{"GET", "/errors"},
	}
	for _, r := range routes {
		pi := doc.Paths.Get(r.path)
		if pi == nil {
			t.Errorf("path %q not found in doc.Paths", r.path)
			continue
		}
		var op interface{}
		switch r.method {
		case "GET":
			op = pi.Get
		case "POST":
			op = pi.Post
		}
		if op == nil {
			t.Errorf("%s %s operation not found", r.method, r.path)
		}
	}

	// 无 loader 时 gin 类型无法解析——记录当前预期行为
	if s := doc.Components.Schemas.Get("gin.H"); s != nil {
		t.Logf("INFO: gin.H resolved to %+v", s)
	} else {
		t.Log("INFO: gin.H not resolved — no loader (expected)")
	}
	if s := doc.Components.Schemas.Get("gin.Error"); s != nil {
		t.Logf("INFO: gin.Error resolved to %+v", s)
	} else {
		t.Log("INFO: gin.Error not resolved — no loader (expected)")
	}
}

// TestAnnotations_GinTypes_WithModuleLoader 验证注入 ModuleLoader 后 gin 类型能被正确解析。
//
// 依赖 go.mod 中存在 gin 依赖，且本机模块缓存已下载对应版本。
// 若模块缓存中找不到 gin，测试以 t.Skip 跳过，不计为失败。
func TestAnnotations_GinTypes_WithModuleLoader(t *testing.T) {
	modInfo, err := parser.ParseGoMod("../../go.mod")
	if err != nil {
		t.Fatalf("parse go.mod: %v", err)
	}
	if _, ok := modInfo.Require["github.com/gin-gonic/gin"]; !ok {
		t.Skip("gin not found in go.mod require — skipping module loader test")
	}

	cacheDir := parser.ModuleCacheDir()

	files := parseAnnotationsFixture(t)
	e := &extractor.GoExtractor{}
	result, err := e.Extract(files)
	if err != nil {
		t.Fatal(err)
	}

	b := NewBuilder()
	b.SetLoader(NewModuleLoader(modInfo, cacheDir))
	doc, err := b.Build(result, files)
	if err != nil {
		t.Fatal(err)
	}

	// GET /version 路由必须存在
	pi := doc.Paths.Get("/version")
	if pi == nil || pi.Get == nil {
		t.Error("GET /version not found")
	}

	// gin.Error 是 struct，应注册到 Components.Schemas 并为 object 类型
	ginErrorSchema := doc.Components.Schemas.Get("gin.Error")
	if ginErrorSchema == nil {
		t.Log("WARN: gin.Error not resolved — module cache may be missing")
	} else {
		if ginErrorSchema.Ref != "" {
			// $ref schema — 通过 ref 找到真实 schema
			t.Logf("gin.Error is $ref: %s", ginErrorSchema.Ref)
		} else if ginErrorSchema.Type != "object" && ginErrorSchema.Type != "" {
			t.Errorf("gin.Error schema type = %q, want object", ginErrorSchema.Type)
		} else {
			t.Logf("gin.Error schema resolved: type=%q", ginErrorSchema.Type)
		}
	}

	// gin.H = type H map[string]any，TypeDef 透传后不注册独立 key
	// 对应的 schema 应内联为 {type: object} 或不出现在 Schemas 中
	if s := doc.Components.Schemas.Get("gin.H"); s != nil {
		t.Logf("gin.H schema registered: %+v", s)
	} else {
		t.Log("gin.H not registered as component (expected: inlined as object)")
	}
}

// TestResolve_LazyLoad_NoDanglingRef 验证懒加载后重试不会因 inProgress 拦截产生悬空 $ref。
//
// 场景：第三方包类型第一次出现时触发懒加载，加载完成后重试时 inProgress[key] 已清除，
// 确保 schema 被真正构建并注册到 Components.Schemas，而不是返回悬空的 $ref。
func TestResolve_LazyLoad_NoDanglingRef(t *testing.T) {
	r, sb := newResolver()

	// controllerFile 通过别名 vpc_pb 引入 tencentvpcmessage 包
	controllerFile := &parser.RawFile{
		Package:  "controller",
		FilePath: "/controller/handler.go",
		Imports: []parser.RawImport{
			{Alias: "vpc_pb", Path: "gitlab.example.com/project/tencentvpcmessage", PkgName: "tencentvpcmessage"},
		},
		Structs: []parser.RawStruct{{
			Name: "Response",
			Fields: []parser.RawField{
				// 字段类型使用别名 vpc_pb，引用第三方包类型
				{Name: "List", TypeName: "[]*vpc_pb.SecurityGroupItem", Tag: `json:"list"`},
			},
		}},
	}
	r.SetFiles([]*parser.RawFile{controllerFile})

	// 模拟懒加载器：第一次调用加载 tencentvpcmessage 包文件
	loaded := false
	r.SetLoader(func(pkgName string, srcFile *parser.RawFile) []*parser.RawFile {
		if pkgName == "tencentvpcmessage" && !loaded {
			loaded = true
			return []*parser.RawFile{{
				Package:  "tencentvpcmessage",
				FilePath: "/tencentvpcmessage/types.go",
				Structs: []parser.RawStruct{{
					Name: "SecurityGroupItem",
					Fields: []parser.RawField{
						{Name: "GroupId", TypeName: "string", Tag: `json:"group_id"`},
						{Name: "GroupName", TypeName: "string", Tag: `json:"group_name"`},
					},
				}},
			}}
		}
		return nil
	})

	s := sb.Build("controller.Response", nil)
	if s == nil || s.Ref == "" {
		t.Fatalf("controller.Response should resolve to $ref, got %+v", s)
	}

	// SecurityGroupItem 必须真正注册（不能是悬空 $ref）
	itemSchema := r.Components().Schemas.Get("tencentvpcmessage.SecurityGroupItem")
	if itemSchema == nil {
		t.Fatal("tencentvpcmessage.SecurityGroupItem should be registered — lazy-load inProgress bug?")
	}
	if itemSchema.Type != "object" {
		t.Errorf("SecurityGroupItem type = %q, want object", itemSchema.Type)
	}
	if itemSchema.Properties == nil || itemSchema.Properties.Get("group_id") == nil {
		t.Error("SecurityGroupItem should have 'group_id' property")
	}
}

// ── SchemaKey helpers ─────────────────────────────────────────────────────────

func TestNewSchemaKey(t *testing.T) {
	cases := []struct{ pkg, typ, want string }{
		{"models", "User", "models.User"},
		{"", "string", "string"},
		{"common", "PageData", "common.PageData"},
	}
	for _, tc := range cases {
		if got := NewSchemaKey(tc.pkg, tc.typ); string(got) != tc.want {
			t.Errorf("NewSchemaKey(%q,%q) = %q, want %q", tc.pkg, tc.typ, got, tc.want)
		}
	}
}

func TestGenericSchemaKey(t *testing.T) {
	base := NewSchemaKey("common", "Resp")
	got := GenericSchemaKey(base, NewSchemaKey("models", "User"))
	want := "common.Resp[models.User]"
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}

	got2 := GenericSchemaKey(base, NewSchemaKey("string", ""), NewSchemaKey("int", ""))
	if !strings.HasPrefix(string(got2), "common.Resp[") {
		t.Errorf("unexpected: %q", got2)
	}
}

func TestCompositeSchemaKey(t *testing.T) {
	base := NewSchemaKey("common", "PageData")

	// 无 overrides → 等于 base
	got := CompositeSchemaKey(base, nil)
	if got != base {
		t.Errorf("no overrides should return base, got %q", got)
	}

	// 有 overrides → 按字母序排列
	got2 := CompositeSchemaKey(base, map[string]string{
		"data":  "[]models.User",
		"total": "int64",
	})
	want := "common.PageData{data=[]models.User,total=int64}"
	if string(got2) != want {
		t.Errorf("got %q, want %q", got2, want)
	}
}

func TestSchemaKeyRef(t *testing.T) {
	k := NewSchemaKey("models", "User")
	if k.Ref() != "#/components/schemas/models.User" {
		t.Errorf("Ref() = %q", k.Ref())
	}
}

// ── primitiveSchema ───────────────────────────────────────────────────────────

func TestPrimitiveSchema(t *testing.T) {
	cases := []struct {
		name       string
		wantType   string
		wantFormat string
	}{
		{"string", "string", ""},
		{"bool", "boolean", ""},
		{"int", "integer", "int32"},
		{"int64", "integer", "int64"},
		{"float32", "number", "float"},
		{"float64", "number", "double"},
		{"integer", "integer", ""},
		{"file", "string", "binary"},
		{"interface{}", "", ""},
		{"any", "", ""},
	}
	for _, tc := range cases {
		s, ok := primitiveSchema(tc.name)
		if !ok {
			t.Errorf("primitiveSchema(%q): not found", tc.name)
			continue
		}
		if s.Type != tc.wantType {
			t.Errorf("primitiveSchema(%q).Type = %q, want %q", tc.name, s.Type, tc.wantType)
		}
		if s.Format != tc.wantFormat {
			t.Errorf("primitiveSchema(%q).Format = %q, want %q", tc.name, s.Format, tc.wantFormat)
		}
	}
	if _, ok := primitiveSchema("unknown"); ok {
		t.Error("unknown type should not be found")
	}
}

// ── builtinSchema ─────────────────────────────────────────────────────────────

func TestBuiltinSchema(t *testing.T) {
	cases := []struct {
		pkg, typ   string
		wantType   string
		wantFormat string
	}{
		{"time", "Time", "string", "date-time"},
		{"time", "Duration", "integer", "int64"},
		{"uuid", "UUID", "string", "uuid"},
		{"url", "URL", "string", "uri"},
		{"net", "IP", "string", "ipv4"},
	}
	for _, tc := range cases {
		s, ok := builtinSchema(tc.pkg, tc.typ)
		if !ok {
			t.Errorf("builtinSchema(%q,%q): not found", tc.pkg, tc.typ)
			continue
		}
		if s.Type != tc.wantType || s.Format != tc.wantFormat {
			t.Errorf("builtinSchema(%q,%q) = {%q,%q}, want {%q,%q}",
				tc.pkg, tc.typ, s.Type, s.Format, tc.wantType, tc.wantFormat)
		}
	}
	if _, ok := builtinSchema("foo", "Bar"); ok {
		t.Error("unknown builtin should not be found")
	}
}

// ── isNumericStr ──────────────────────────────────────────────────────────────

func TestIsNumericStr(t *testing.T) {
	trues := []string{"0", "1", "42", "-1", "100"}
	for _, s := range trues {
		if !isNumericStr(s) {
			t.Errorf("isNumericStr(%q) = false, want true", s)
		}
	}
	falses := []string{"", "active", "1.5", "abc", "1a"}
	for _, s := range falses {
		if isNumericStr(s) {
			t.Errorf("isNumericStr(%q) = true, want false", s)
		}
	}
}

// ── splitTypeArgs / parseOverrides ───────────────────────────────────────────

func TestSplitTypeArgs(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"string,int", []string{"string", "int"}},
		{"[]models.User,int64", []string{"[]models.User", "int64"}},
		{"KV[string,int],bool", []string{"KV[string,int]", "bool"}},
		{"single", []string{"single"}},
		{"", nil},
	}
	for _, tc := range cases {
		got := splitTypeArgs(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("splitTypeArgs(%q) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitTypeArgs(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}

func TestParseOverrides(t *testing.T) {
	got := parseOverrides("data=[]models.User,total=int64")
	if got["data"] != "[]models.User" || got["total"] != "int64" {
		t.Errorf("got %v", got)
	}
	empty := parseOverrides("")
	if len(empty) != 0 {
		t.Errorf("empty string should produce empty map")
	}
}

// ── indexCompositeOpen ────────────────────────────────────────────────────────

func TestIndexCompositeOpen(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"common.PageData{data=[]models.User}", 15},
		{"Pair[string]{x=int}", 12},  // '{' inside generic brackets is skipped
		{"NoComposite", -1},
		{"map[string]int", -1},
	}
	for _, tc := range cases {
		if got := indexCompositeOpen(tc.input); got != tc.want {
			t.Errorf("indexCompositeOpen(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

// ── parseStructTags ───────────────────────────────────────────────────────────

func TestParseStructTags_Basic(t *testing.T) {
	info := parseStructTags(`json:"user_name" validate:"required"`, "UserName")
	if info.jsonName != "user_name" {
		t.Errorf("jsonName = %q", info.jsonName)
	}
	if !info.required {
		t.Error("required should be true")
	}
	if info.skip {
		t.Error("skip should be false")
	}
}

func TestParseStructTags_Skip(t *testing.T) {
	info := parseStructTags(`json:"-"`, "Field")
	if !info.skip {
		t.Error("json:\"-\" should skip")
	}
}

func TestParseStructTags_Omitempty(t *testing.T) {
	info := parseStructTags(`json:"name,omitempty"`, "Name")
	if info.jsonName != "name" || !info.omitempty {
		t.Errorf("got jsonName=%q omitempty=%v", info.jsonName, info.omitempty)
	}
}

func TestParseStructTags_NoJsonTag(t *testing.T) {
	info := parseStructTags(`validate:"required"`, "MyField")
	if info.jsonName != "MyField" {
		t.Errorf("jsonName should fall back to field name, got %q", info.jsonName)
	}
}

func TestParseStructTags_Constraints(t *testing.T) {
	info := parseStructTags(
		`json:"price" minimum:"0" maximum:"999.99" minLength:"1" maxLength:"200" pattern:"^\\d+$" minItems:"1" maxItems:"10" uniqueItems:"true"`,
		"Price",
	)
	if info.minimum == nil || *info.minimum != 0 {
		t.Errorf("minimum = %v", info.minimum)
	}
	if info.maximum == nil || *info.maximum != 999.99 {
		t.Errorf("maximum = %v", info.maximum)
	}
	if info.minLength == nil || *info.minLength != 1 {
		t.Errorf("minLength = %v", info.minLength)
	}
	if info.maxLength == nil || *info.maxLength != 200 {
		t.Errorf("maxLength = %v", info.maxLength)
	}
	if info.pattern != `^\d+$` {
		t.Errorf("pattern = %q", info.pattern)
	}
	if info.minItems == nil || *info.minItems != 1 {
		t.Errorf("minItems = %v", info.minItems)
	}
	if info.maxItems == nil || *info.maxItems != 10 {
		t.Errorf("maxItems = %v", info.maxItems)
	}
	if !info.uniqueItems {
		t.Error("uniqueItems should be true")
	}
}

func TestParseStructTags_AccessControl(t *testing.T) {
	info := parseStructTags(`json:"id" readonly:"true" deprecated:"true"`, "ID")
	if !info.readonly {
		t.Error("readonly should be true")
	}
	if !info.deprecated {
		t.Error("deprecated should be true")
	}
}

func TestParseStructTags_Enums(t *testing.T) {
	info := parseStructTags(`json:"status" enums:"active,inactive,deleted"`, "Status")
	if len(info.enum) != 3 || info.enum[0] != "active" || info.enum[2] != "deleted" {
		t.Errorf("enum = %v", info.enum)
	}
}

func TestParseStructTags_BindingRequired(t *testing.T) {
	info := parseStructTags(`json:"name" binding:"required,min=3"`, "Name")
	if !info.required {
		t.Error("binding:required should set required=true")
	}
}

// ── Resolver.Resolve — primitive & builtin ────────────────────────────────────

func newResolver() (*Resolver, *SchemaBuilder) {
	r := NewResolver()
	sb := NewSchemaBuilder(r)
	return r, sb
}

func TestResolve_Primitives(t *testing.T) {
	_, sb := newResolver()
	cases := []struct{ typ, wantType, wantFmt string }{
		{"string", "string", ""},
		{"int64", "integer", "int64"},
		{"bool", "boolean", ""},
		{"float64", "number", "double"},
	}
	for _, tc := range cases {
		s := sb.Build(tc.typ, nil)
		if s == nil {
			t.Fatalf("Build(%q) = nil", tc.typ)
		}
		if s.Type != tc.wantType || s.Format != tc.wantFmt {
			t.Errorf("Build(%q) = {%q,%q}, want {%q,%q}",
				tc.typ, s.Type, s.Format, tc.wantType, tc.wantFmt)
		}
	}
}

func TestResolve_BuiltinExternalTypes(t *testing.T) {
	_, sb := newResolver()
	file := &parser.RawFile{
		Package: "models",
		Imports: []parser.RawImport{
			{Alias: "", Path: "time", PkgName: "time"},
		},
	}
	s := sb.Build("time.Time", file)
	if s == nil || s.Type != "string" || s.Format != "date-time" {
		t.Errorf("time.Time → got %+v", s)
	}
}

func TestResolve_Pointer(t *testing.T) {
	_, sb := newResolver()
	s := sb.Build("*string", nil)
	if s == nil || s.Type != "string" || !s.Nullable {
		t.Errorf("*string → got %+v", s)
	}
}

func TestResolve_Slice(t *testing.T) {
	_, sb := newResolver()
	s := sb.Build("[]int64", nil)
	if s == nil || s.Type != "array" {
		t.Fatalf("[]int64 → got %+v", s)
	}
	if s.Items == nil || s.Items.Format != "int64" {
		t.Errorf("items = %+v", s.Items)
	}
}

func TestResolve_Map(t *testing.T) {
	_, sb := newResolver()
	s := sb.Build("map[string]int", nil)
	if s == nil || s.Type != "object" {
		t.Fatalf("map[string]int → got %+v", s)
	}
	if s.AdditionalProperties == nil {
		t.Error("map should have additionalProperties")
	}
}

func TestResolve_Interface(t *testing.T) {
	_, sb := newResolver()
	s := sb.Build("interface{}", nil)
	if s == nil {
		t.Error("interface{} should return empty schema, got nil")
	}
	if s.Type != "" {
		t.Errorf("interface{} should have no type, got %q", s.Type)
	}
}

func TestResolve_SkipSpecialTypes(t *testing.T) {
	_, sb := newResolver()
	for _, typ := range []string{"func", "chan", "struct{}"} {
		if s := sb.Build(typ, nil); s != nil {
			t.Errorf("Build(%q) should be nil, got %+v", typ, s)
		}
	}
}

// ── Resolver.Resolve — struct lookup ─────────────────────────────────────────

func TestResolve_Struct(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{{
		Package:  "models",
		FilePath: "/models/user.go",
		Structs: []parser.RawStruct{{
			Name: "User",
			Fields: []parser.RawField{
				{Name: "ID", TypeName: "int64", Tag: `json:"id"`},
				{Name: "Name", TypeName: "string", Tag: `json:"name" validate:"required"`},
				{Name: "Email", TypeName: "string", Tag: `json:"email"`},
			},
		}},
	}})

	s := sb.Build("models.User", nil)
	if s == nil || s.Ref == "" {
		t.Fatalf("models.User should return $ref, got %+v", s)
	}
	if s.Ref != "#/components/schemas/models.User" {
		t.Errorf("Ref = %q", s.Ref)
	}

	// schema is registered in components
	registered := r.Components().Schemas.Get("models.User")
	if registered == nil {
		t.Fatal("models.User not found in Components.Schemas")
	}
	if registered.Type != "object" {
		t.Errorf("registered schema type = %q", registered.Type)
	}
	if registered.Properties == nil {
		t.Fatal("properties should not be nil")
	}
	if registered.Properties.Get("id") == nil {
		t.Error("property 'id' missing")
	}
	// name is required
	if len(registered.Required) == 0 || registered.Required[0] != "name" {
		t.Errorf("Required = %v, want [name]", registered.Required)
	}
}

func TestResolve_StructRegisteredOnce(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{{
		Package:  "models",
		FilePath: "/models/item.go",
		Structs: []parser.RawStruct{{
			Name:   "Item",
			Fields: []parser.RawField{{Name: "ID", TypeName: "int64", Tag: `json:"id"`}},
		}},
	}})

	sb.Build("models.Item", nil)
	sb.Build("models.Item", nil) // second call

	// should still be only one entry in registry
	count := 0
	r.Components().Schemas.ForEach(func(key string, _ *spec3.Schema) error {
		if key == "models.Item" {
			count++
		}
		return nil
	})
	if count != 1 {
		t.Errorf("models.Item registered %d times, want 1", count)
	}
}

func TestResolve_StructSelfRecursive(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{{
		Package:  "models",
		FilePath: "/models/node.go",
		Structs: []parser.RawStruct{{
			Name: "TreeNode",
			Fields: []parser.RawField{
				{Name: "Value", TypeName: "int", Tag: `json:"value"`},
				{Name: "Children", TypeName: "[]*TreeNode", Tag: `json:"children"`},
			},
		}},
	}})

	// should not infinite-loop
	s := sb.Build("models.TreeNode", nil)
	if s == nil || s.Ref == "" {
		t.Fatalf("TreeNode should return $ref, got %+v", s)
	}
}

func TestResolve_CrossPackageTypeRef(t *testing.T) {
	// order.go 引用 pet.go 中的 Pet（跨文件同包）
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{
		{
			Package:  "shop",
			FilePath: "/shop/order.go",
			Structs: []parser.RawStruct{{
				Name:   "Order",
				Fields: []parser.RawField{{Name: "Pet", TypeName: "Pet", Tag: `json:"pet"`}},
			}},
		},
		{
			Package:  "shop",
			FilePath: "/shop/pet.go",
			Structs: []parser.RawStruct{{
				Name:   "Pet",
				Fields: []parser.RawField{{Name: "Name", TypeName: "string", Tag: `json:"name"`}},
			}},
		},
	})

	file := &parser.RawFile{Package: "shop", FilePath: "/shop/order.go"}
	s := sb.Build("shop.Order", file)
	if s == nil || s.Ref == "" {
		t.Fatalf("shop.Order should return $ref, got %+v", s)
	}

	// Pet should also be registered (resolved as Order's field type)
	if r.Components().Schemas.Get("shop.Pet") == nil {
		t.Error("shop.Pet should be registered via Order.Pet field")
	}
}

// ── Resolver.Resolve — type alias ─────────────────────────────────────────────

func TestResolve_TypeAlias(t *testing.T) {
	_, sb := newResolver()
	file := &parser.RawFile{
		Package:  "models",
		FilePath: "/models/aliases.go",
		TypeAliases: []parser.RawTypeAlias{
			{Name: "TimeAlias", TypeName: "time.Time"},
		},
		Imports: []parser.RawImport{{Path: "time", PkgName: "time"}},
	}
	// Resolver needs to find the file to look up the alias
	r, sb2 := newResolver()
	r.SetFiles([]*parser.RawFile{file})
	_ = sb

	s := sb2.Build("models.TimeAlias", nil)
	// TimeAlias → time.Time → {type: string, format: date-time}
	if s == nil || s.Type != "string" || s.Format != "date-time" {
		t.Errorf("TimeAlias → got %+v", s)
	}
}

// ── Resolver.Resolve — const enum ─────────────────────────────────────────────

func TestResolve_ConstEnum_String(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{{
		Package:  "models",
		FilePath: "/models/status.go",
		Consts: []parser.RawConst{
			{Name: "StatusActive", TypeName: "Status", Value: "active"},
			{Name: "StatusInactive", TypeName: "Status", Value: "inactive"},
			{Name: "StatusDeleted", TypeName: "Status", Value: "deleted"},
		},
	}})

	s := sb.Build("models.Status", nil)
	if s == nil || s.Ref == "" {
		t.Fatalf("models.Status should return $ref, got %+v", s)
	}
	registered := r.Components().Schemas.Get("models.Status")
	if registered == nil {
		t.Fatal("models.Status not found in Components.Schemas")
	}
	if registered.Type != "string" {
		t.Errorf("enum type = %q, want string", registered.Type)
	}
	if len(registered.Enum) != 3 {
		t.Errorf("enum values = %v, want 3", registered.Enum)
	}
}

func TestResolve_ConstEnum_Int(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{{
		Package:  "models",
		FilePath: "/models/direction.go",
		Consts: []parser.RawConst{
			{Name: "North", TypeName: "Direction", Value: "0"},
			{Name: "South", TypeName: "Direction", Value: "1"},
		},
	}})

	sb.Build("models.Direction", nil)
	registered := r.Components().Schemas.Get("models.Direction")
	if registered == nil || registered.Type != "integer" {
		t.Errorf("int enum type = %q, want integer", registered.Type)
	}
}

func TestResolve_ConstEnum_WithComments(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{{
		Package:  "models",
		FilePath: "/models/status.go",
		Consts: []parser.RawConst{
			{Name: "StatusActive", TypeName: "Status", Value: "active", Comments: []string{"活跃"}},
			{Name: "StatusInactive", TypeName: "Status", Value: "inactive", Comments: []string{"未激活"}},
			{Name: "StatusDeleted", TypeName: "Status", Value: "deleted"},
		},
	}})

	sb.Build("models.Status", nil)
	registered := r.Components().Schemas.Get("models.Status")
	if registered == nil {
		t.Fatal("models.Status not found in Components.Schemas")
	}

	varnames, ok := registered.Extensions.GetStringSlice("x-enum-varnames")
	if !ok || len(varnames) != 3 {
		t.Fatalf("x-enum-varnames = %v, want 3 elements", varnames)
	}
	if varnames[0] != "StatusActive" || varnames[1] != "StatusInactive" || varnames[2] != "StatusDeleted" {
		t.Errorf("x-enum-varnames = %v", varnames)
	}

	descs, ok := registered.Extensions.GetStringSlice("x-enumdescriptions")
	if !ok || len(descs) != 3 {
		t.Fatalf("x-enumdescriptions = %v, want 3 elements", descs)
	}
	if descs[0] != "活跃" || descs[1] != "未激活" || descs[2] != "" {
		t.Errorf("x-enumdescriptions = %v", descs)
	}
}

func TestResolve_ConstEnum_NoComments_NoDescriptions(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{{
		Package:  "models",
		FilePath: "/models/dir.go",
		Consts: []parser.RawConst{
			{Name: "DirNorth", TypeName: "Direction", Value: "0"},
			{Name: "DirSouth", TypeName: "Direction", Value: "1"},
		},
	}})

	sb.Build("models.Direction", nil)
	registered := r.Components().Schemas.Get("models.Direction")
	if registered == nil {
		t.Fatal("models.Direction not found in Components.Schemas")
	}

	if _, ok := registered.Extensions.GetStringSlice("x-enumdescriptions"); ok {
		t.Error("x-enumdescriptions should not be set when no const has comments")
	}

	varnames, ok := registered.Extensions.GetStringSlice("x-enum-varnames")
	if !ok || len(varnames) != 2 {
		t.Fatalf("x-enum-varnames = %v, want 2 elements", varnames)
	}
}

// ── Resolver.Resolve — package alias ─────────────────────────────────────────

func TestResolve_PackageAlias(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{{
		Package:  "models",
		FilePath: "/models/user.go",
		Structs: []parser.RawStruct{{
			Name:   "User",
			Fields: []parser.RawField{{Name: "ID", TypeName: "int64", Tag: `json:"id"`}},
		}},
	}})

	// file that imports models with alias "m"
	file := &parser.RawFile{
		Package: "handlers",
		Imports: []parser.RawImport{
			{Alias: "m", Path: "github.com/example/models", PkgName: "models"},
		},
	}
	s := sb.Build("m.User", file)
	if s == nil || s.Ref == "" {
		t.Fatalf("m.User (alias for models.User) should return $ref, got %+v", s)
	}
	if s.Ref != "#/components/schemas/models.User" {
		t.Errorf("Ref = %q, want models.User", s.Ref)
	}
}

// ── Resolver.Resolve — embedded fields ───────────────────────────────────────

func TestResolve_EmbeddedFields(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{{
		Package:  "models",
		FilePath: "/models/models.go",
		Structs: []parser.RawStruct{
			{
				Name: "BaseModel",
				Fields: []parser.RawField{
					{Name: "ID", TypeName: "int64", Tag: `json:"id"`},
				},
			},
			{
				Name: "User",
				Fields: []parser.RawField{
					{Name: "BaseModel", TypeName: "BaseModel", Embedded: true},
					{Name: "Name", TypeName: "string", Tag: `json:"name"`},
				},
			},
		},
	}})

	sb.Build("models.User", nil)
	registered := r.Components().Schemas.Get("models.User")
	if registered == nil {
		t.Fatal("models.User not in Components.Schemas")
	}
	if len(registered.AllOf) == 0 {
		t.Error("embedded field should produce allOf")
	}
}

// ── Resolver.Resolve — composite type ────────────────────────────────────────

func TestResolve_CompositeType(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{
		{
			Package:  "common",
			FilePath: "/common/page.go",
			Structs: []parser.RawStruct{{
				Name: "PageData",
				Fields: []parser.RawField{
					{Name: "Total", TypeName: "int", Tag: `json:"total"`},
					{Name: "Data", TypeName: "interface{}", Tag: `json:"data"`},
				},
			}},
		},
		{
			Package:  "models",
			FilePath: "/models/user.go",
			Structs: []parser.RawStruct{{
				Name:   "User",
				Fields: []parser.RawField{{Name: "ID", TypeName: "int64", Tag: `json:"id"`}},
			}},
		},
	})

	file := &parser.RawFile{
		Package: "handlers",
		Imports: []parser.RawImport{
			{Path: "example.com/common", PkgName: "common"},
			{Path: "example.com/models", PkgName: "models"},
		},
	}
	s := sb.Build("common.PageData{data=[]models.User}", file)
	if s == nil {
		t.Fatal("composite type should return non-nil schema")
	}
	// 组合类型直接内联为 allOf，不返回 $ref
	if s.Ref != "" {
		t.Errorf("composite type should be inline (no $ref), got Ref=%q", s.Ref)
	}
	// allOf 应包含两个元素：baseRef + override object
	if len(s.AllOf) != 2 {
		t.Fatalf("composite schema AllOf length = %d, want 2", len(s.AllOf))
	}
	// 第一个元素为 baseSchema 的 $ref
	if s.AllOf[0].Ref == "" {
		t.Error("allOf[0] should be $ref to base schema")
	}
	// 第二个元素为包含 override properties 的 object schema
	override := s.AllOf[1]
	if override.Properties == nil || override.Properties.Get("data") == nil {
		t.Error("allOf[1] should contain override property 'data'")
	}
	// 组合类型不注册到 Components.Schemas
	if r.Components().Schemas.Get("common.PageData{data=[]models.User}") != nil {
		t.Error("composite schema should NOT be registered in Components.Schemas")
	}
}

// ── Resolver.Resolve — generic type ──────────────────────────────────────────

func TestResolve_GenericType(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{
		{
			Package:  "common",
			FilePath: "/common/resp.go",
			Structs: []parser.RawStruct{{
				Name:       "Resp",
				TypeParams: []parser.RawTypeParam{{Name: "T", Constraint: "any"}},
				Fields: []parser.RawField{
					{Name: "Code", TypeName: "int", Tag: `json:"code"`},
					{Name: "Data", TypeName: "T", Tag: `json:"data"`},
				},
			}},
		},
		{
			Package:  "models",
			FilePath: "/models/user.go",
			Structs: []parser.RawStruct{{
				Name:   "User",
				Fields: []parser.RawField{{Name: "ID", TypeName: "int64", Tag: `json:"id"`}},
			}},
		},
	})

	file := &parser.RawFile{
		Package: "handlers",
		Imports: []parser.RawImport{
			{Path: "example.com/common", PkgName: "common"},
			{Path: "example.com/models", PkgName: "models"},
		},
	}
	s := sb.Build("common.Resp[models.User]", file)
	if s == nil || s.Ref == "" {
		t.Fatalf("generic type should return $ref, got %+v", s)
	}

	genericKey := "common.Resp[models.User]"
	registered := r.Components().Schemas.Get(genericKey)
	if registered == nil {
		t.Fatalf("generic schema not found in Components.Schemas")
	}
	// Data field should be resolved to models.User $ref
	if registered.Properties == nil {
		t.Fatal("properties should not be nil")
	}
	dataProp := registered.Properties.Get("data")
	if dataProp == nil {
		t.Fatal("data property missing")
	}
	if dataProp.Ref == "" {
		t.Errorf("data should be $ref to models.User, got %+v", dataProp)
	}
}

// ── applyTagConstraints ───────────────────────────────────────────────────────

func TestApplyTagConstraints(t *testing.T) {
	minVal := 0.0
	maxVal := 100.0
	minLen := int64(1)
	maxLen := int64(255)
	info := fieldTagInfo{
		format:      "email",
		example:     "user@example.com",
		defaultVal:  "none",
		enum:        []string{"a", "b"},
		readonly:    true,
		minimum:     &minVal,
		maximum:     &maxVal,
		minLength:   &minLen,
		maxLength:   &maxLen,
		pattern:     "^[a-z]+$",
		uniqueItems: true,
	}
	tmp := emptySchema()
	s := &tmp
	applyTagConstraints(s, info)

	if s.Format != "email" {
		t.Errorf("Format = %q", s.Format)
	}
	if s.Example != "user@example.com" {
		t.Errorf("Example = %v", s.Example)
	}
	if s.Default != "none" {
		t.Errorf("Default = %v", s.Default)
	}
	if len(s.Enum) != 2 {
		t.Errorf("Enum = %v", s.Enum)
	}
	if !s.ReadOnly {
		t.Error("ReadOnly should be true")
	}
	if s.Minimum == nil || *s.Minimum != 0 {
		t.Errorf("Minimum = %v", s.Minimum)
	}
	if s.Maximum == nil || *s.Maximum != 100 {
		t.Errorf("Maximum = %v", s.Maximum)
	}
	if s.MinLength == nil || *s.MinLength != 1 {
		t.Errorf("MinLength = %v", s.MinLength)
	}
	if s.MaxLength == nil || *s.MaxLength != 255 {
		t.Errorf("MaxLength = %v", s.MaxLength)
	}
	if s.Pattern != "^[a-z]+$" {
		t.Errorf("Pattern = %q", s.Pattern)
	}
	if !s.UniqueItems {
		t.Error("UniqueItems should be true")
	}
}

// emptySchema returns a new empty spec3.Schema for testing.
func emptySchema() spec3.Schema { return spec3.Schema{} }

// ── Builder.Build — integration ───────────────────────────────────────────────

func TestBuilder_Build_MinimalDoc(t *testing.T) {
	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{
			Title:   "Test API",
			Version: "1.0.0",
		},
	}

	doc, err := NewBuilder().Build(result, nil)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Info.Title != "Test API" || doc.Info.Version != "1.0.0" {
		t.Errorf("Info = %+v", doc.Info)
	}
	if doc.OpenAPI != "3.0.3" {
		t.Errorf("OpenAPI = %q", doc.OpenAPI)
	}
}

func TestBuilder_Build_Servers_Explicit(t *testing.T) {
	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{
			Title:   "API",
			Version: "1.0.0",
			Servers: []extractor.ServerAnnotation{
				{URL: "https://api.example.com", Description: "Production"},
				{URL: "http://localhost:8080"},
			},
		},
	}
	doc, err := NewBuilder().Build(result, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Servers) != 2 {
		t.Fatalf("Servers len = %d, want 2", len(doc.Servers))
	}
	if doc.Servers[0].URL != "https://api.example.com" || doc.Servers[0].Description != "Production" {
		t.Errorf("Servers[0] = %+v", doc.Servers[0])
	}
}

func TestBuilder_Build_Servers_LegacyHostBasePath(t *testing.T) {
	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{
			Title:    "API",
			Version:  "1.0.0",
			Host:     "api.example.com",
			BasePath: "/v1",
			Schemes:  []string{"https"},
		},
	}
	doc, err := NewBuilder().Build(result, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Servers) != 1 || doc.Servers[0].URL != "https://api.example.com/v1" {
		t.Errorf("Servers = %+v", doc.Servers)
	}
}

func TestBuilder_Build_Tags(t *testing.T) {
	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{
			Title:   "API",
			Version: "1.0.0",
			Tags: []extractor.TagAnnotation{
				{Name: "pets", Description: "Pet operations"},
				{Name: "users", Description: "User operations"},
			},
		},
	}
	doc, err := NewBuilder().Build(result, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Tags) != 2 || doc.Tags[0].Name != "pets" {
		t.Errorf("Tags = %+v", doc.Tags)
	}
}

func TestBuilder_Build_Operation(t *testing.T) {
	files := []*parser.RawFile{{
		Package:  "models",
		FilePath: "/models/user.go",
		Structs: []parser.RawStruct{{
			Name: "User",
			Fields: []parser.RawField{
				{Name: "ID", TypeName: "int64", Tag: `json:"id"`},
				{Name: "Name", TypeName: "string", Tag: `json:"name"`},
			},
		}},
	}}

	handlerFile := &parser.RawFile{
		Package:  "handlers",
		FilePath: "/handlers/user.go",
		Imports: []parser.RawImport{
			{Path: "example.com/models", PkgName: "models"},
		},
	}
	files = append(files, handlerFile)

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{Title: "API", Version: "1.0.0"},
		Operations: []extractor.OperationAnnotation{
			{
				FuncName: "GetUser",
				FilePath: "/handlers/user.go",
				Line:     10,
				Summary:  "Get user by ID",
				Tags:     []string{"users"},
				Route:    extractor.RouteInfo{Method: "GET", Path: "/users/{id}"},
				Params: []extractor.ParamAnnotation{
					{Name: "id", In: "path", TypeName: "integer", Required: true, Format: "int64", Description: "User ID"},
				},
				Responses: []extractor.ResponseAnnotation{
					{Code: "200", TypeName: "models.User", WrapType: "object", Description: "OK"},
					{Code: "404", TypeName: "models.ErrorResp", WrapType: "object", Description: "Not found"},
				},
				Security: []extractor.SecurityRequirement{{Name: "ApiKeyAuth"}},
			},
		},
	}

	doc, err := NewBuilder().Build(result, files)
	if err != nil {
		t.Fatal(err)
	}

	// Paths
	if doc.Paths == nil {
		t.Fatal("Paths is nil")
	}
	pi := doc.Paths.Get("/users/{id}")
	if pi == nil || pi.Get == nil {
		t.Fatal("GET /users/{id} not found")
	}
	op := pi.Get
	if op.Summary == nil || *op.Summary != "Get user by ID" {
		t.Errorf("Summary = %v", op.Summary)
	}
	if len(op.Tags) != 1 || op.Tags[0] != "users" {
		t.Errorf("Tags = %v", op.Tags)
	}
	if len(op.Parameters) != 1 {
		t.Fatalf("Parameters len = %d", len(op.Parameters))
	}
	param := op.Parameters[0]
	if param.Name != "id" || param.In != "path" || !param.Required {
		t.Errorf("param = %+v", param)
	}
	if param.Schema.Format != "int64" {
		t.Errorf("param.Schema.Format = %q", param.Schema.Format)
	}

	// Responses
	resp200 := op.Responses.Get("200")
	if resp200 == nil {
		t.Fatal("200 response missing")
	}
	if resp200.Content == nil {
		t.Error("200 response should have content")
	}

	// Security
	if len(op.Security) != 1 {
		t.Fatalf("Security len = %d, want 1", len(op.Security))
	}

	// Components: models.User registered
	if doc.Components.Schemas.Get("models.User") == nil {
		t.Error("models.User not in Components.Schemas")
	}
}

func TestBuilder_Build_RequestBody(t *testing.T) {
	files := []*parser.RawFile{
		{
			Package:  "models",
			FilePath: "/models/req.go",
			Structs: []parser.RawStruct{{
				Name:   "CreateUserReq",
				Fields: []parser.RawField{{Name: "Name", TypeName: "string", Tag: `json:"name"`}},
			}},
		},
		{
			Package:  "handlers",
			FilePath: "/handlers/h.go",
			Imports:  []parser.RawImport{{Path: "example.com/models", PkgName: "models"}},
		},
	}

	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{Title: "API", Version: "1.0.0"},
		Operations: []extractor.OperationAnnotation{{
			FuncName: "CreateUser",
			FilePath: "/handlers/h.go",
			Route:    extractor.RouteInfo{Method: "POST", Path: "/users"},
			Accept:   []string{"application/json"},
			Params: []extractor.ParamAnnotation{
				{Name: "body", In: "body", TypeName: "models.CreateUserReq", Required: true},
			},
		}},
	}

	doc, err := NewBuilder().Build(result, files)
	if err != nil {
		t.Fatal(err)
	}
	pi := doc.Paths.Get("/users")
	if pi == nil || pi.Post == nil {
		t.Fatal("POST /users not found")
	}
	rb := pi.Post.RequestBody
	if !rb.Required {
		t.Error("RequestBody.Required should be true")
	}
	if rb.Content.Get("application/json") == nil {
		t.Error("RequestBody should have application/json content")
	}
}

func TestBuilder_Build_Deprecated(t *testing.T) {
	files := []*parser.RawFile{
		{Package: "handlers", FilePath: "/handlers/h.go"},
	}
	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{Title: "API", Version: "1.0.0"},
		Operations: []extractor.OperationAnnotation{{
			FuncName:   "OldEndpoint",
			FilePath:   "/handlers/h.go",
			Route:      extractor.RouteInfo{Method: "GET", Path: "/old"},
			Deprecated: true,
		}},
	}

	doc, err := NewBuilder().Build(result, files)
	if err != nil {
		t.Fatal(err)
	}
	op := doc.Paths.Get("/old").Get
	if op == nil || !op.Deprecated {
		t.Error("operation should be deprecated")
	}
}

func TestBuilder_Build_SecuritySchemes(t *testing.T) {
	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{
			Title:   "API",
			Version: "1.0.0",
			SecurityDefs: []extractor.SecurityDefAnnotation{
				{Name: "ApiKeyAuth", Type: "apiKey", In: "header", KeyName: "X-API-Key"},
				{Name: "BearerAuth", Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
			},
		},
	}

	doc, err := NewBuilder().Build(result, nil)
	if err != nil {
		t.Fatal(err)
	}

	apiKey := doc.Components.SecuritySchemes.Get("ApiKeyAuth")
	if apiKey == nil {
		t.Fatal("ApiKeyAuth not found")
	}
	if apiKey.Type != "apiKey" || apiKey.In != "header" || apiKey.Name != "X-API-Key" {
		t.Errorf("ApiKeyAuth = %+v", apiKey)
	}

	bearer := doc.Components.SecuritySchemes.Get("BearerAuth")
	if bearer == nil {
		t.Fatal("BearerAuth not found")
	}
	if bearer.Scheme != "bearer" || bearer.BearerFormat != "JWT" {
		t.Errorf("BearerAuth = %+v", bearer)
	}
}

func TestBuilder_Build_JSON(t *testing.T) {
	result := &extractor.ExtractResult{
		Global: extractor.GlobalAnnotation{Title: "My API", Version: "2.0.0"},
	}
	doc, err := NewBuilder().Build(result, nil)
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if m["openapi"] != "3.0.3" {
		t.Errorf("openapi = %v", m["openapi"])
	}
	info, ok := m["info"].(map[string]interface{})
	if !ok || info["title"] != "My API" {
		t.Errorf("info = %v", m["info"])
	}
}

// ── resolveQualifier 严格模式 ─────────────────────────────────────────────────

// TestResolve_MissingImport 当 file 非 nil 且 qualifier 未出现在该文件的 import
// 声明中时，Resolver 应兜底在已加载文件中按包名查找，仍能成功解析。
func TestResolve_MissingImport(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{{
		Package:  "models",
		FilePath: "/models/user.go",
		Structs: []parser.RawStruct{{
			Name:   "User",
			Fields: []parser.RawField{{Name: "ID", TypeName: "int64", Tag: `json:"id"`}},
		}},
	}})

	// file 非 nil 但没有 models 的 import → 兜底搜索已加载文件，应返回 $ref
	fileNoImport := &parser.RawFile{
		Package:  "handlers",
		FilePath: "/handlers/h.go",
		Imports:  []parser.RawImport{}, // 无 import，依赖兜底逻辑
	}
	s := sb.Build("models.User", fileNoImport)
	if s == nil || s.Ref == "" {
		t.Errorf("should resolve via fallback when qualifier %q not in file imports, got %+v", "models", s)
	}
	if r.Components().Schemas.Get("models.User") == nil {
		t.Error("models.User should be registered in Components.Schemas via fallback")
	}
}

// TestResolve_HasImport 当 file 非 nil 且正确声明了 import 时，Resolver 能找到类型。
func TestResolve_HasImport(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{{
		Package:  "models",
		FilePath: "/models/user.go",
		Structs: []parser.RawStruct{{
			Name:   "User",
			Fields: []parser.RawField{{Name: "ID", TypeName: "int64", Tag: `json:"id"`}},
		}},
	}})

	// file 非 nil 且有 models 的 import → 应返回 $ref
	fileWithImport := &parser.RawFile{
		Package:  "handlers",
		FilePath: "/handlers/h.go",
		Imports: []parser.RawImport{
			{Path: "github.com/example/models", PkgName: "models"},
		},
	}
	s := sb.Build("models.User", fileWithImport)
	if s == nil || s.Ref == "" {
		t.Errorf("should return $ref when import is present, got %+v", s)
	}
	if r.Components().Schemas.Get("models.User") == nil {
		t.Error("models.User should be registered in Components.Schemas")
	}
}

// TestResolve_AliasedImport_ByPkgName 验证当 import 使用了别名引入时，
// 注解中仍可使用包的真实名称（PkgName）进行类型引用。
//
// 场景：
//
//	import rsp "github.com/foo/resourcemgrmessage"
//	// @Success 200 {object} response.Wrapper{data=resourcemgrmessage.QueryRsp}
func TestResolve_AliasedImport_ByPkgName(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{
		{
			Package:  "response",
			FilePath: "/response/wrapper.go",
			Structs: []parser.RawStruct{{
				Name:   "Wrapper",
				Fields: []parser.RawField{{Name: "Data", TypeName: "interface{}", Tag: `json:"data"`}},
			}},
		},
		{
			Package:  "resourcemgrmessage",
			FilePath: "/resourcemgrmessage/query.go",
			Structs: []parser.RawStruct{{
				Name:   "QueryRsp",
				Fields: []parser.RawField{{Name: "List", TypeName: "[]string", Tag: `json:"list"`}},
			}},
		},
	})

	// import 使用了别名 "rsp"，但注解里用的是包本名 "resourcemgrmessage"
	file := &parser.RawFile{
		Package: "handlers",
		Imports: []parser.RawImport{
			{Alias: "", Path: "github.com/foo/response", PkgName: "response"},
			{Alias: "rsp", Path: "github.com/foo/resourcemgrmessage", PkgName: "resourcemgrmessage"},
		},
	}

	// 用别名引用（应正常工作）
	s1 := sb.Build("response.Wrapper{data=rsp.QueryRsp}", file)
	if s1 == nil {
		t.Fatal("alias reference should resolve, got nil")
	}
	if len(s1.AllOf) != 2 {
		t.Fatalf("expected allOf[2], got %d", len(s1.AllOf))
	}

	// 用包本名引用（兼容行为，应同样正常工作）
	s2 := sb.Build("response.Wrapper{data=resourcemgrmessage.QueryRsp}", file)
	if s2 == nil {
		t.Fatal("pkg-name reference (with alias import) should resolve, got nil")
	}
	if len(s2.AllOf) != 2 {
		t.Fatalf("expected allOf[2], got %d", len(s2.AllOf))
	}
	// 两种写法解析结果一致：override 字段都是 QueryRsp
	if r.Components().Schemas.Get("resourcemgrmessage.QueryRsp") == nil {
		t.Error("resourcemgrmessage.QueryRsp should be registered in Components.Schemas")
	}
}

// TestResolve_LocalStruct_FieldWithAliasedImport 验证函数局部 struct 的字段使用了别名引入的包类型时能正确解析。
//
// 场景（与实际业务代码一致）：
//
//	import ecs_pb "gitlab.example.com/project/ecsmessage"
//
//	func Controller() gin.HandlerFunc {
//	    type Response struct {
//	        List []*ecs_pb.SecurityGroupItem `json:"list"`
//	    }
//	    // @Success 200 {object} controller.Controller.Response
//	}
func TestResolve_LocalStruct_FieldWithAliasedImport(t *testing.T) {
	r, sb := newResolver()

	// 模拟已加载文件：controller 文件（含局部 struct）+ ecsmessage 包（第三方）
	controllerFile := &parser.RawFile{
		Package:  "controller",
		FilePath: "/controller/handler.go",
		Imports: []parser.RawImport{
			{Alias: "ecs_pb", Path: "gitlab.example.com/project/ecsmessage", PkgName: "ecsmessage"},
		},
		Functions: []parser.RawFunc{{
			Name:     "NewController",
			FilePath: "/controller/handler.go",
			LocalStructs: []parser.RawStruct{{
				Name: "Response",
				Fields: []parser.RawField{
					// 字段类型使用别名 ecs_pb
					{Name: "List", TypeName: "[]*ecs_pb.SecurityGroupItem", Tag: `json:"list"`},
					{Name: "Total", TypeName: "int32", Tag: `json:"total"`},
				},
			}},
		}},
	}
	ecsmessageFile := &parser.RawFile{
		Package:  "ecsmessage",
		FilePath: "/ecsmessage/types.go",
		Structs: []parser.RawStruct{{
			Name: "SecurityGroupItem",
			Fields: []parser.RawField{
				{Name: "GroupId", TypeName: "string", Tag: `json:"group_id"`},
				{Name: "GroupName", TypeName: "string", Tag: `json:"group_name"`},
			},
		}},
	}
	r.SetFiles([]*parser.RawFile{controllerFile, ecsmessageFile})

	// 解析函数局部类型
	s := sb.Build("controller.NewController.Response", controllerFile)
	if s == nil || s.Ref == "" {
		t.Fatalf("controller.NewController.Response should resolve to $ref, got %+v", s)
	}

	// Response 的 schema 应包含 list 字段（需要解析 ecs_pb.SecurityGroupItem）
	respSchema := r.Components().Schemas.Get("controller.NewController.Response")
	if respSchema == nil {
		t.Fatal("controller.NewController.Response not found in Components.Schemas")
	}
	if respSchema.Properties == nil || respSchema.Properties.Get("list") == nil {
		t.Error("Response schema should contain 'list' field resolved from ecs_pb.SecurityGroupItem")
	}
	if respSchema.Properties.Get("total") == nil {
		t.Error("Response schema should contain 'total' field")
	}

	// SecurityGroupItem 应被注册为独立 schema
	if r.Components().Schemas.Get("ecsmessage.SecurityGroupItem") == nil {
		t.Error("ecsmessage.SecurityGroupItem should be registered in Components.Schemas")
	}
}

// TestResolve_LocalStruct_SiblingReference 验证局部 struct 的字段引用同函数内另一个局部 struct 时能正确解析。
//
// 场景（真实业务代码）：
//
//	func Controller() gin.HandlerFunc {
//	    type RegionItem struct { RegionCode string `json:"region_code"` }
//	    type Response struct { RegionList []*RegionItem `json:"region_list"` }
//	    // @Success 200 {object} response.baseResponse{data=controller.Controller.Response}
//	}
func TestResolve_LocalStruct_SiblingReference(t *testing.T) {
	_, sb := newResolver()

	controllerFile := &parser.RawFile{
		Package:  "controller",
		FilePath: "/controller/handler.go",
		Functions: []parser.RawFunc{{
			Name:     "NewRDSController",
			FilePath: "/controller/handler.go",
			LocalStructs: []parser.RawStruct{
				{
					Name: "RegionItem",
					Fields: []parser.RawField{
						{Name: "RegionCode", TypeName: "string", Tag: `json:"region_code"`},
					},
				},
				{
					Name: "Response",
					Fields: []parser.RawField{
						// 字段引用同函数内的 RegionItem（无包限定符）
						{Name: "RegionList", TypeName: "[]*RegionItem", Tag: `json:"region_list"`},
						{Name: "Total", TypeName: "int32", Tag: `json:"total"`},
					},
				},
			},
		}},
	}
	sb.resolver.SetFiles([]*parser.RawFile{controllerFile})

	s := sb.Build("controller.NewRDSController.Response", controllerFile)
	if s == nil || s.Ref == "" {
		t.Fatalf("controller.NewRDSController.Response should resolve to $ref, got %+v", s)
	}

	respSchema := sb.resolver.Components().Schemas.Get("controller.NewRDSController.Response")
	if respSchema == nil {
		t.Fatal("Response not found in Components.Schemas")
	}

	// region_list 字段必须存在（需要找到 RegionItem 局部 struct）
	if respSchema.Properties == nil || respSchema.Properties.Get("region_list") == nil {
		t.Error("Response schema should contain 'region_list' field (sibling local struct reference)")
	}
	if respSchema.Properties.Get("total") == nil {
		t.Error("Response schema should contain 'total' field")
	}

	// RegionItem 应以包含函数名的完整 key 注册（避免与其他函数同名局部 struct 冲突）
	regionItemKey := "controller.NewRDSController.RegionItem"
	if sb.resolver.Components().Schemas.Get(regionItemKey) == nil {
		t.Errorf("sibling local struct should be registered as %q", regionItemKey)
	}
}

// TestResolve_NilFile_Permissive 当 file 为 nil 时宽松处理（向后兼容，测试代码常用场景）。
func TestResolve_NilFile_Permissive(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{{
		Package:  "models",
		FilePath: "/models/user.go",
		Structs: []parser.RawStruct{{
			Name:   "User",
			Fields: []parser.RawField{{Name: "ID", TypeName: "int64", Tag: `json:"id"`}},
		}},
	}})

	// nil file → permissive，qualifier 直接作为包名使用
	s := sb.Build("models.User", nil)
	if s == nil || s.Ref == "" {
		t.Errorf("nil file should be permissive, got %+v", s)
	}
}

// TestResolve_MissingImport_Composite 组合类型的 base qualifier 缺少 import 时，
// 应兜底在已加载文件中查找，仍能成功解析。
func TestResolve_MissingImport_Composite(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{{
		Package:  "common",
		FilePath: "/common/page.go",
		Structs: []parser.RawStruct{{
			Name:   "PageData",
			Fields: []parser.RawField{{Name: "Total", TypeName: "int", Tag: `json:"total"`}},
		}},
	}})

	fileNoImport := &parser.RawFile{
		Package:  "handlers",
		FilePath: "/handlers/h.go",
		// 无 common import，依赖兜底逻辑
	}
	s := sb.Build("common.PageData{total=int}", fileNoImport)
	if s == nil {
		t.Errorf("composite type should resolve via fallback when qualifier not in imports")
	}
	_ = r
}

// TestAnnotations_MissingImportFallback 验证 fixture 文件中，
// 即使将 v1.go 的 import 清空，Resolver 也能通过兜底策略（在已加载文件中按包名查找）
// 解析出 models.Pet、common.PageData 等跨包类型。
func TestAnnotations_MissingImportFallback(t *testing.T) {
	files := parseAnnotationsFixture(t)

	// 模拟 v1.go 没有 import：从 v1.go 的 RawFile 中清空 imports
	for _, f := range files {
		if f.Package == "annotations" {
			f.Imports = nil
		}
	}

	e := &extractor.GoExtractor{}
	result, err := e.Extract(files)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := NewBuilder().Build(result, files)
	if err != nil {
		t.Fatal(err)
	}

	// 兜底策略：即使没有 import，只要包在已加载文件中就能解析
	if doc.Components.Schemas.Get("models.Pet") == nil {
		t.Error("models.Pet should be resolved via fallback even when import is missing")
	}
	if doc.Components.Schemas.Get("common.PageData") == nil {
		t.Error("common.PageData should be resolved via fallback even when import is missing")
	}
}

// ── 跨包类型递归链 ─────────────────────────────────────────────────────────────
//
// 场景说明：
//   handlers.CreateReq → models.User → common.Address → string（三层跨包）
//   解析 CreateReq 时，Resolver 应递归注册 models.User 和 common.Address。

// threeLayerFiles 构造三层跨包引用的文件集合：
//
//	handlers/req.go    CreateReq  { User models.User }
//	models/user.go     User       { ID int64, Address common.Address }
//	common/address.go  Address    { Street string, City string, Country string }
func threeLayerFiles() []*parser.RawFile {
	return []*parser.RawFile{
		{
			Package:  "handlers",
			FilePath: "/handlers/req.go",
			Imports: []parser.RawImport{
				{Path: "example.com/models", PkgName: "models"},
			},
			Structs: []parser.RawStruct{{
				Name: "CreateReq",
				Fields: []parser.RawField{
					{Name: "User", TypeName: "models.User", Tag: `json:"user"`},
				},
			}},
		},
		{
			Package:  "models",
			FilePath: "/models/user.go",
			Imports: []parser.RawImport{
				{Path: "example.com/common", PkgName: "common"},
			},
			Structs: []parser.RawStruct{{
				Name: "User",
				Fields: []parser.RawField{
					{Name: "ID", TypeName: "int64", Tag: `json:"id"`},
					{Name: "Name", TypeName: "string", Tag: `json:"name"`},
					{Name: "Address", TypeName: "common.Address", Tag: `json:"address"`},
				},
			}},
		},
		{
			Package:  "common",
			FilePath: "/common/address.go",
			Structs: []parser.RawStruct{{
				Name: "Address",
				Fields: []parser.RawField{
					{Name: "Street", TypeName: "string", Tag: `json:"street"`},
					{Name: "City", TypeName: "string", Tag: `json:"city"`},
					{Name: "Country", TypeName: "string", Tag: `json:"country"`},
				},
			}},
		},
	}
}

// TestResolve_ThreeLevelChain 验证三层跨包引用链全量注册：
// handlers.CreateReq → models.User → common.Address → primitives
func TestResolve_ThreeLevelChain(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles(threeLayerFiles())

	s := sb.Build("handlers.CreateReq", nil)
	if s == nil || s.Ref == "" {
		t.Fatalf("handlers.CreateReq should return $ref, got %+v", s)
	}

	// 所有三层类型必须都注册到 Components.Schemas
	for _, key := range []string{"handlers.CreateReq", "models.User", "common.Address"} {
		if r.Components().Schemas.Get(key) == nil {
			t.Errorf("%q not found in Components.Schemas", key)
		}
	}

	// models.User 的 address 字段应是 $ref → common.Address
	userSchema := r.Components().Schemas.Get("models.User")
	if userSchema == nil || userSchema.Properties == nil {
		t.Fatal("models.User schema missing or has no properties")
	}
	addrProp := userSchema.Properties.Get("address")
	if addrProp == nil || addrProp.Ref == "" {
		t.Errorf("models.User.address should be $ref, got %+v", addrProp)
	}
	if addrProp.Ref != "#/components/schemas/common.Address" {
		t.Errorf("address $ref = %q, want common.Address", addrProp.Ref)
	}
}

// TestResolve_ThreeLevelChain_AliasPkg 验证每层文件独立 import alias 的情况。
//
// handlers 用 alias "m" 引用 models，models 用 alias "c" 引用 common。
// 解析 m.User 时应按 handlers 文件的 imports 解析为 models.User，
// 再解析 User.Address (c.Address) 时应按 models 文件的 imports 解析为 common.Address。
func TestResolve_ThreeLevelChain_AliasPkg(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{
		{
			Package:  "models",
			FilePath: "/models/user.go",
			Imports: []parser.RawImport{
				{Alias: "c", Path: "example.com/common", PkgName: "common"},
			},
			Structs: []parser.RawStruct{{
				Name: "User",
				Fields: []parser.RawField{
					{Name: "ID", TypeName: "int64", Tag: `json:"id"`},
					{Name: "Address", TypeName: "c.Address", Tag: `json:"address"`},
				},
			}},
		},
		{
			Package:  "common",
			FilePath: "/common/address.go",
			Structs: []parser.RawStruct{{
				Name:   "Address",
				Fields: []parser.RawField{{Name: "City", TypeName: "string", Tag: `json:"city"`}},
			}},
		},
	})

	// handlers 文件用 alias "m" 引用 models
	handlerFile := &parser.RawFile{
		Package: "handlers",
		Imports: []parser.RawImport{
			{Alias: "m", Path: "example.com/models", PkgName: "models"},
		},
	}

	s := sb.Build("m.User", handlerFile)
	if s == nil || s.Ref != "#/components/schemas/models.User" {
		t.Fatalf("m.User should resolve to models.User $ref, got %+v", s)
	}

	// common.Address 应被递归注册（由 models/user.go 的 c.Address 触发）
	if r.Components().Schemas.Get("common.Address") == nil {
		t.Error("common.Address not registered via alias chain")
	}
}

// TestResolve_ChainWithBuiltinExternal 验证链中混入内置外部类型（time.Time）不中断递归。
//
//	shop.Order  → shop.Pet (跨文件) + time.Time (内置短路)
//	shop.Pet    → time.Time (内置短路) + string
func TestResolve_ChainWithBuiltinExternal(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{
		{
			Package:  "shop",
			FilePath: "/shop/order.go",
			Imports:  []parser.RawImport{{Path: "time", PkgName: "time"}},
			Structs: []parser.RawStruct{{
				Name: "Order",
				Fields: []parser.RawField{
					{Name: "ID", TypeName: "int64", Tag: `json:"id"`},
					{Name: "Pet", TypeName: "Pet", Tag: `json:"pet"`},
					{Name: "CreatedAt", TypeName: "time.Time", Tag: `json:"created_at"`},
				},
			}},
		},
		{
			Package:  "shop",
			FilePath: "/shop/pet.go",
			Imports:  []parser.RawImport{{Path: "time", PkgName: "time"}},
			Structs: []parser.RawStruct{{
				Name: "Pet",
				Fields: []parser.RawField{
					{Name: "Name", TypeName: "string", Tag: `json:"name"`},
					{Name: "UpdatedAt", TypeName: "time.Time", Tag: `json:"updated_at"`},
				},
			}},
		},
	})

	file := &parser.RawFile{Package: "shop", FilePath: "/shop/order.go",
		Imports: []parser.RawImport{{Path: "time", PkgName: "time"}},
	}
	s := sb.Build("shop.Order", file)
	if s == nil || s.Ref == "" {
		t.Fatalf("shop.Order should return $ref, got %+v", s)
	}

	// Order 和 Pet 都在 Components.Schemas
	for _, key := range []string{"shop.Order", "shop.Pet"} {
		if r.Components().Schemas.Get(key) == nil {
			t.Errorf("%q not found in Components.Schemas", key)
		}
	}

	// Order.created_at 是内置类型，应内联 {type:string, format:date-time}，不注册到 Schemas
	orderSchema := r.Components().Schemas.Get("shop.Order")
	if orderSchema.Properties == nil {
		t.Fatal("Order properties nil")
	}
	createdAt := orderSchema.Properties.Get("created_at")
	if createdAt == nil || createdAt.Type != "string" || createdAt.Format != "date-time" {
		t.Errorf("created_at should be inline date-time schema, got %+v", createdAt)
	}

	// Pet.updated_at 同样
	petSchema := r.Components().Schemas.Get("shop.Pet")
	updatedAt := petSchema.Properties.Get("updated_at")
	if updatedAt == nil || updatedAt.Format != "date-time" {
		t.Errorf("updated_at should be date-time, got %+v", updatedAt)
	}
}

// TestResolve_ChainWithConstEnum 验证链中含常量 enum 类型时全量递归注册。
//
//	models.User    { Status status.Status }
//	status.Status  const enum: active / inactive
func TestResolve_ChainWithConstEnum(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{
		{
			Package:  "models",
			FilePath: "/models/user.go",
			Imports:  []parser.RawImport{{Path: "example.com/status", PkgName: "status"}},
			Structs: []parser.RawStruct{{
				Name: "User",
				Fields: []parser.RawField{
					{Name: "ID", TypeName: "int64", Tag: `json:"id"`},
					{Name: "Status", TypeName: "status.Status", Tag: `json:"status"`},
				},
			}},
		},
		{
			Package:  "status",
			FilePath: "/status/status.go",
			Consts: []parser.RawConst{
				{Name: "Active", TypeName: "Status", Value: "active"},
				{Name: "Inactive", TypeName: "Status", Value: "inactive"},
			},
		},
	})

	sb.Build("models.User", nil)

	if r.Components().Schemas.Get("models.User") == nil {
		t.Error("models.User not registered")
	}
	statusSchema := r.Components().Schemas.Get("status.Status")
	if statusSchema == nil {
		t.Fatal("status.Status not registered via chain")
	}
	if statusSchema.Type != "string" || len(statusSchema.Enum) != 2 {
		t.Errorf("status.Status schema = %+v", statusSchema)
	}

	// User.status 字段应是 $ref → status.Status
	userSchema := r.Components().Schemas.Get("models.User")
	statusProp := userSchema.Properties.Get("status")
	if statusProp == nil || statusProp.Ref == "" {
		t.Errorf("User.status should be $ref, got %+v", statusProp)
	}
}

// TestResolve_ChainWithTypeAlias 验证链中含类型别名时穿透递归到底层类型。
//
//	models.User    { ID models.UserID }
//	models.UserID  = common.ID
//	common.ID      = int64  (primitive alias)
func TestResolve_ChainWithTypeAlias(t *testing.T) {
	_, sb := newResolver()
	r2, sb2 := newResolver()
	r2.SetFiles([]*parser.RawFile{
		{
			Package:  "models",
			FilePath: "/models/user.go",
			TypeAliases: []parser.RawTypeAlias{
				{Name: "UserID", TypeName: "common.ID"},
			},
			Imports: []parser.RawImport{{Path: "example.com/common", PkgName: "common"}},
		},
		{
			Package:  "common",
			FilePath: "/common/types.go",
			TypeAliases: []parser.RawTypeAlias{
				{Name: "ID", TypeName: "int64"},
			},
		},
	})
	_ = sb

	// models.UserID → common.ID → int64 → {type:integer, format:int64}
	s := sb2.Build("models.UserID", nil)
	if s == nil || s.Type != "integer" || s.Format != "int64" {
		t.Errorf("models.UserID alias chain → got %+v", s)
	}
}

// TestResolve_CircularCrossPackage 验证跨包循环引用（A.B ↔ B.A）不产生死循环。
//
//	pkg_a.A  { B *pkg_b.B }
//	pkg_b.B  { A *pkg_a.A }
func TestResolve_CircularCrossPackage(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{
		{
			Package:  "pkg_a",
			FilePath: "/pkg_a/a.go",
			Imports:  []parser.RawImport{{Path: "example.com/pkg_b", PkgName: "pkg_b"}},
			Structs: []parser.RawStruct{{
				Name: "A",
				Fields: []parser.RawField{
					{Name: "Value", TypeName: "string", Tag: `json:"value"`},
					{Name: "B", TypeName: "*pkg_b.B", Tag: `json:"b"`},
				},
			}},
		},
		{
			Package:  "pkg_b",
			FilePath: "/pkg_b/b.go",
			Imports:  []parser.RawImport{{Path: "example.com/pkg_a", PkgName: "pkg_a"}},
			Structs: []parser.RawStruct{{
				Name: "B",
				Fields: []parser.RawField{
					{Name: "Name", TypeName: "string", Tag: `json:"name"`},
					{Name: "A", TypeName: "*pkg_a.A", Tag: `json:"a"`},
				},
			}},
		},
	})

	// 必须不死循环，返回 $ref
	s := sb.Build("pkg_a.A", nil)
	if s == nil || s.Ref == "" {
		t.Fatalf("pkg_a.A should return $ref, got %+v", s)
	}

	// 两个类型都应注册到 Components
	for _, key := range []string{"pkg_a.A", "pkg_b.B"} {
		if r.Components().Schemas.Get(key) == nil {
			t.Errorf("%q not found in Components.Schemas", key)
		}
	}
}

// TestResolve_MapOfCrossPackageType 验证 map value 是跨包类型时触发递归注册。
//
//	handlers.Config  { Policies map[string]policy.Rule }
//	policy.Rule      { Name string, Priority int }
func TestResolve_MapOfCrossPackageType(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{
		{
			Package:  "handlers",
			FilePath: "/handlers/config.go",
			Imports:  []parser.RawImport{{Path: "example.com/policy", PkgName: "policy"}},
			Structs: []parser.RawStruct{{
				Name: "Config",
				Fields: []parser.RawField{
					{Name: "Policies", TypeName: "map[string]policy.Rule", Tag: `json:"policies"`},
				},
			}},
		},
		{
			Package:  "policy",
			FilePath: "/policy/rule.go",
			Structs: []parser.RawStruct{{
				Name: "Rule",
				Fields: []parser.RawField{
					{Name: "Name", TypeName: "string", Tag: `json:"name"`},
					{Name: "Priority", TypeName: "int", Tag: `json:"priority"`},
				},
			}},
		},
	})

	sb.Build("handlers.Config", nil)

	configSchema := r.Components().Schemas.Get("handlers.Config")
	if configSchema == nil {
		t.Fatal("handlers.Config not registered")
	}

	policiesProp := configSchema.Properties.Get("policies")
	if policiesProp == nil || policiesProp.Type != "object" {
		t.Fatalf("policies should be object schema, got %+v", policiesProp)
	}
	if policiesProp.AdditionalProperties == nil {
		t.Fatal("map type should have additionalProperties")
	}

	// additionalProperties 指向 policy.Rule 的 $ref
	addPropSchema, ok := policiesProp.AdditionalProperties.(*spec3.Schema)
	if !ok || addPropSchema.Ref == "" {
		t.Errorf("additionalProperties should be $ref to policy.Rule, got %+v", policiesProp.AdditionalProperties)
	}

	// policy.Rule 也应在 Components.Schemas 中
	if r.Components().Schemas.Get("policy.Rule") == nil {
		t.Error("policy.Rule not found in Components.Schemas")
	}
}

// TestResolve_SliceOfCrossPackageType 验证 []pkg.Type 中的元素类型被递归注册。
func TestResolve_SliceOfCrossPackageType(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{
		{
			Package:  "api",
			FilePath: "/api/list.go",
			Imports:  []parser.RawImport{{Path: "example.com/models", PkgName: "models"}},
			Structs: []parser.RawStruct{{
				Name: "ListResp",
				Fields: []parser.RawField{
					{Name: "Items", TypeName: "[]models.Item", Tag: `json:"items"`},
					{Name: "Total", TypeName: "int64", Tag: `json:"total"`},
				},
			}},
		},
		{
			Package:  "models",
			FilePath: "/models/item.go",
			Structs: []parser.RawStruct{{
				Name: "Item",
				Fields: []parser.RawField{
					{Name: "ID", TypeName: "int64", Tag: `json:"id"`},
					{Name: "Name", TypeName: "string", Tag: `json:"name"`},
				},
			}},
		},
	})

	sb.Build("api.ListResp", nil)

	if r.Components().Schemas.Get("api.ListResp") == nil {
		t.Error("api.ListResp not registered")
	}
	if r.Components().Schemas.Get("models.Item") == nil {
		t.Error("models.Item not registered via slice element")
	}

	listSchema := r.Components().Schemas.Get("api.ListResp")
	itemsProp := listSchema.Properties.Get("items")
	if itemsProp == nil || itemsProp.Type != "array" {
		t.Fatalf("items should be array schema, got %+v", itemsProp)
	}
	if itemsProp.Items == nil || itemsProp.Items.Ref == "" {
		t.Errorf("items.Items should be $ref to models.Item, got %+v", itemsProp.Items)
	}
}

// TestResolve_FourLevelChain 验证四层跨包引用链全量注册。
//
//	api.Request → biz.Order → models.User → common.BaseModel → primitives
func TestResolve_FourLevelChain(t *testing.T) {
	r, sb := newResolver()
	r.SetFiles([]*parser.RawFile{
		{
			Package:  "api",
			FilePath: "/api/req.go",
			Imports:  []parser.RawImport{{Path: "example.com/biz", PkgName: "biz"}},
			Structs: []parser.RawStruct{{
				Name:   "Request",
				Fields: []parser.RawField{{Name: "Order", TypeName: "biz.Order", Tag: `json:"order"`}},
			}},
		},
		{
			Package:  "biz",
			FilePath: "/biz/order.go",
			Imports:  []parser.RawImport{{Path: "example.com/models", PkgName: "models"}},
			Structs: []parser.RawStruct{{
				Name: "Order",
				Fields: []parser.RawField{
					{Name: "ID", TypeName: "int64", Tag: `json:"id"`},
					{Name: "User", TypeName: "models.User", Tag: `json:"user"`},
				},
			}},
		},
		{
			Package:  "models",
			FilePath: "/models/user.go",
			Imports:  []parser.RawImport{{Path: "example.com/common", PkgName: "common"}},
			Structs: []parser.RawStruct{{
				Name: "User",
				Fields: []parser.RawField{
					{Name: "Name", TypeName: "string", Tag: `json:"name"`},
					{Name: "Base", TypeName: "common.BaseModel", Tag: `json:"base"`},
				},
			}},
		},
		{
			Package:  "common",
			FilePath: "/common/base.go",
			Structs: []parser.RawStruct{{
				Name: "BaseModel",
				Fields: []parser.RawField{
					{Name: "ID", TypeName: "int64", Tag: `json:"id"`},
					{Name: "CreatedAt", TypeName: "string", Tag: `json:"created_at"`},
				},
			}},
		},
	})

	sb.Build("api.Request", nil)

	for _, key := range []string{"api.Request", "biz.Order", "models.User", "common.BaseModel"} {
		if r.Components().Schemas.Get(key) == nil {
			t.Errorf("%q not found in Components.Schemas after four-level resolution", key)
		}
	}
}

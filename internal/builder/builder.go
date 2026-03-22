package builder

import (
	"strings"

	spec3 "github.com/shuttlefy/go-openapi3-spec"
	"github.com/shuttlefy/go-openapi3-swag/internal/extractor"
	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

// Builder 编排 SchemaBuilder + OperationBuilder，产出 *spec3.OpenAPI。
type Builder struct {
	schema   *SchemaBuilder
	op       *OperationBuilder
	resolver *Resolver
}

func NewBuilder() *Builder {
	resolver := NewResolver()
	schema := NewSchemaBuilder(resolver)
	return &Builder{
		schema:   schema,
		op:       NewOperationBuilder(schema),
		resolver: resolver,
	}
}

// SetLoader 注入第三方包懒加载器，用于解析模块缓存中的类型。
// 需在 Build 调用之前设置。
func (b *Builder) SetLoader(loader PackageLoader) {
	b.resolver.SetLoader(loader)
}

// SetQueryStructExplode 控制 query 注解类型为 struct 时是否自动打散字段。
// 默认 false（不打散）；设为 true 后，所有 in=query 的 struct 类型参数将自动展开。
func (b *Builder) SetQueryStructExplode(v bool) {
	b.op.queryStructExplode = v
}

// NewModuleLoader 创建基于 go.mod 模块缓存的 PackageLoader。
// modInfo 通过 parser.ParseGoMod 获得，cacheDir 通过 parser.ModuleCacheDir 获得。
func NewModuleLoader(modInfo *parser.ModuleInfo, cacheDir string) PackageLoader {
	p := &parser.GoParser{}
	loaded := make(map[string]bool)
	return func(pkgName string, srcFile *parser.RawFile) []*parser.RawFile {
		importPath := findImportPathFromFile(pkgName, srcFile)
		if importPath == "" || loaded[importPath] {
			return nil
		}
		loaded[importPath] = true
		dir, ok := parser.ResolvePackageDir(importPath, modInfo, cacheDir)
		if !ok {
			return nil
		}
		files, _ := p.ParseDir(dir)
		return files
	}
}

// findImportPathFromFile 从文件 import 声明中查找包名对应的完整 import path。
func findImportPathFromFile(pkgName string, file *parser.RawFile) string {
	if file == nil {
		return ""
	}
	return findImportPath(pkgName, file)
}

// Build 接收 ExtractResult 和所有解析文件，返回完整的 *spec3.OpenAPI。
func (b *Builder) Build(result *extractor.ExtractResult, files []*parser.RawFile) (*spec3.OpenAPI, error) {
	b.resolver.SetFiles(files)

	// 构建 filePath → *RawFile 索引，供 OperationBuilder 查找当前文件
	fileIndex := make(map[string]*parser.RawFile, len(files))
	for _, f := range files {
		fileIndex[f.FilePath] = f
	}

	g := result.Global

	doc := &spec3.OpenAPI{
		OpenAPI: "3.0.3",
		Info: &spec3.Info{
			Title:          g.Title,
			Version:        g.Version,
			Description:    g.Description,
			TermsOfService: g.TermsOfService,
		},
	}

	// Contact / License
	if g.Contact.Name != "" || g.Contact.Email != "" || g.Contact.URL != "" {
		doc.Info.Contact = &spec3.Contact{
			Name:  g.Contact.Name,
			Email: g.Contact.Email,
			URL:   g.Contact.URL,
		}
	}
	if g.License.Name != "" {
		doc.Info.License = &spec3.License{
			Name: g.License.Name,
			URL:  g.License.URL,
		}
	}

	// ExternalDocs
	if g.ExternalDocs.URL != "" {
		doc.ExternalDocs = &spec3.ExternalDocumentation{
			URL:         g.ExternalDocs.URL,
			Description: g.ExternalDocs.Description,
		}
	}

	// Servers
	doc.Servers = buildServers(g)

	// Tags
	for _, t := range g.Tags {
		doc.Tags = append(doc.Tags, spec3.Tag{
			Name:        t.Name,
			Description: t.Description,
		})
	}

	// Paths（所有 @Router 操作）
	var paths spec3.Paths
	for _, opAnno := range result.Operations {
		operation, err := b.op.Build(opAnno, fileIndex)
		if err != nil {
			return nil, err
		}
		setOperation(&paths, opAnno.Route.Path, opAnno.Route.Method, operation)
	}
	doc.Paths = &paths

	// Components（Schemas + SecuritySchemes）
	components := b.schema.Components()
	buildSecuritySchemes(components, g.SecurityDefs)
	doc.Components = components

	return doc, nil
}

// ── Servers ───────────────────────────────────────────────────────────────────

func buildServers(g extractor.GlobalAnnotation) []spec3.Server {
	var servers []spec3.Server
	for _, s := range g.Servers {
		servers = append(servers, spec3.Server{URL: s.URL, Description: s.Description})
	}
	// swaggo 兼容：@host + @BasePath + @schemes
	if len(servers) == 0 && g.Host != "" {
		servers = append(servers, spec3.Server{URL: buildServerURL(g)})
	}
	return servers
}

func buildServerURL(g extractor.GlobalAnnotation) string {
	scheme := "https"
	if len(g.Schemes) > 0 {
		scheme = g.Schemes[0]
	}
	base := g.BasePath
	if base == "" {
		base = "/"
	}
	return scheme + "://" + g.Host + base
}

// ── Paths ─────────────────────────────────────────────────────────────────────

// setOperation 将 operation 设置到 PathItem 的对应 HTTP 方法字段。
func setOperation(paths *spec3.Paths, path, method string, op *spec3.Operation) {
	pi := paths.Get(path)
	if pi == nil {
		pi = &spec3.PathItem{}
		paths.Set(path, pi)
	}
	switch strings.ToUpper(method) {
	case "GET":
		pi.Get = op
	case "POST":
		pi.Post = op
	case "PUT":
		pi.Put = op
	case "DELETE":
		pi.Delete = op
	case "PATCH":
		pi.Patch = op
	case "HEAD":
		pi.Head = op
	case "OPTIONS":
		pi.Options = op
	case "TRACE":
		pi.Trace = op
	}
}

// ── SecuritySchemes ───────────────────────────────────────────────────────────

func buildSecuritySchemes(components *spec3.Components, defs []extractor.SecurityDefAnnotation) {
	for _, def := range defs {
		scheme := &spec3.SecurityScheme{
			Type:        def.Type,
			Description: def.Description,
		}
		switch def.Type {
		case "apiKey":
			scheme.Name = def.KeyName
			scheme.In = def.In
		case "http":
			scheme.Scheme = def.Scheme
			scheme.BearerFormat = def.BearerFormat
		case "oauth2":
			flows := spec3.OAuthFlows{}
			for _, f := range def.Flows {
				flow := spec3.OAuthFlow{
					AuthorizationURL: f.AuthorizationURL,
					TokenURL:         f.TokenURL,
				}
				for scopeName, scopeDesc := range f.Scopes {
					desc := scopeDesc
					flow.Scopes.Set(scopeName, &desc)
				}
				switch f.Type {
				case "implicit":
					flows.Implicit = flow
				case "password":
					flows.Password = flow
				case "clientCredentials":
					flows.ClientCredentials = flow
				case "authorizationCode":
					flows.AuthorizationCode = flow
				}
			}
			scheme.Flows = flows
		case "openIdConnect":
			scheme.OpenIDConnectURL = def.OpenIDConnectURL
		}
		components.SecuritySchemes.Set(def.Name, scheme)
	}
}

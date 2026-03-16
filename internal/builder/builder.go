package builder

import (
	"fmt"
	"sort"
	"strings"

	spec "github.com/shuttlefy/go-openapi3-spec"
	"github.com/shuttlefy/go-openapi3-swag/internal/extractor"
	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

type Builder struct {
	warnings []string
}

func NewBuilder() *Builder {
	return &Builder{}
}

// Warnings returns non-fatal diagnostic messages collected during the last Build call.
func (b *Builder) Warnings() []string {
	return b.warnings
}

// Build converts extracted annotations and raw AST into a complete OpenAPI 3 document.
func (b *Builder) Build(result *extractor.ExtractResult, rawAST *parser.RawAST) (*spec.OpenAPI, error) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)
	ob := NewOperationBuilder(sb)

	b.warnings = nil

	// Phase 0: set primary package and register all scanned package names.
	// The primary package is used to decide whether a type's schema name is
	// kept as a short name ("Pet") or qualified ("bo.StockItem").
	sb.SetPrimaryPackage(rawAST.Package)
	for _, pkg := range rawAST.Packages {
		sb.RegisterPackage(pkg)
	}

	// Phase 1a: register top-level (non-scoped) structs.
	for i := range rawAST.Structs {
		if rawAST.Structs[i].FuncScope == "" {
			sb.RegisterStruct(&rawAST.Structs[i])
		}
	}

	// Phase 1b: register function-local structs that belong to an annotated
	// operation function. This allows annotations such as
	//   @Param body body SearchFilter true "..."
	// to reference a struct defined inside the same handler function body.
	opFuncs := make(map[string]bool, len(result.Operations))
	for _, op := range result.Operations {
		if op.FuncName != "" {
			opFuncs[op.FuncName] = true
		}
	}
	for i := range rawAST.Structs {
		s := &rawAST.Structs[i]
		if s.FuncScope != "" && opFuncs[s.FuncScope] {
			sb.RegisterStruct(s)
		}
	}

	// Phase 1c: register top-level type aliases (e.g. type Status string).
	// These produce named schemas in components and allow $ref usage in fields.
	for i := range rawAST.TypeAliases {
		if rawAST.TypeAliases[i].FuncScope == "" {
			sb.RegisterTypeAlias(&rawAST.TypeAliases[i])
		}
	}

	// Phase 1d: register typed constants for enum generation on alias schemas.
	for i := range rawAST.Consts {
		if rawAST.Consts[i].FuncScope == "" {
			sb.RegisterConst(rawAST.Consts[i])
		}
	}

	// Phase 2: build all schemas
	sb.BuildAll()

	// Collect unknown-type warnings with candidate suggestions.
	for _, unknown := range sb.UnknownTypeNames() {
		candidates := b.findCandidates(resolver, unknown)
		msg := fmt.Sprintf("schema references unknown type %q", unknown)
		if len(candidates) > 0 {
			msg += fmt.Sprintf("; possible matches: %s", strings.Join(candidates, ", "))
		}
		b.warnings = append(b.warnings, msg)
	}

	doc := &spec.OpenAPI{
		OpenAPI: "3.2.0",
	}
	// Phase 3: Info, Servers, ExternalDocs
	doc.Info = buildInfo(result.Global)
	doc.Servers = buildServers(result.Global)
	if result.Global.ExternalDocs != nil {
		doc.ExternalDocs = &spec.ExternalDocumentation{
			URL:         result.Global.ExternalDocs.URL,
			Description: result.Global.ExternalDocs.Description,
		}
	}

	// Phase 4: build Operations → Paths
	paths := &spec.Paths{}
	for _, opAnno := range result.Operations {
		operation := ob.Build(opAnno)
		routePath := opAnno.Route.Path
		method := strings.ToUpper(opAnno.Route.Method)

		item := paths.Get(routePath)
		if item == nil {
			item = &spec.PathItem{}
		}
		setOperationOnPathItem(item, method, operation)
		paths.Set(routePath, item)
	}
	doc.Paths = paths

	// Phase 5: Components
	doc.Components = buildComponents(sb, result.Global.SecurityDefs)

	// Phase 6: Tags
	doc.Tags = buildTags(result.Global.Tags)

	// Phase 7: Global security
	if len(result.Global.SecurityDefs) > 0 {
		// OpenAPI 3 global security is typically set at operation level;
		// we don't auto-apply global security here unless explicitly annotated.
	}

	return doc, nil
}

func buildInfo(g extractor.GlobalAnnotation) *spec.Info {
	info := &spec.Info{
		Title:          g.Title,
		Description:    g.Description,
		Version:        g.Version,
		TermsOfService: g.TermsOfService,
	}
	if g.Contact.Name != "" || g.Contact.Email != "" || g.Contact.URL != "" {
		info.Contact = &spec.Contact{
			Name:  g.Contact.Name,
			URL:   g.Contact.URL,
			Email: g.Contact.Email,
		}
	}
	if g.License.Name != "" {
		info.License = &spec.License{
			Name: g.License.Name,
			URL:  g.License.URL,
		}
	}
	return info
}

func buildServers(g extractor.GlobalAnnotation) []spec.Server {
	if len(g.Servers) > 0 {
		servers := make([]spec.Server, len(g.Servers))
		for i, s := range g.Servers {
			servers[i] = spec.Server{
				URL:         s.URL,
				Description: s.Description,
			}
		}
		return servers
	}

	// Fallback: synthesize from Host/BasePath/Schemes
	if g.Host != "" {
		scheme := "https"
		if len(g.Schemes) > 0 {
			scheme = g.Schemes[0]
		}
		url := scheme + "://" + g.Host
		if g.BasePath != "" && g.BasePath != "/" {
			url += g.BasePath
		}
		return []spec.Server{{URL: url}}
	}

	return nil
}

func buildComponents(sb *SchemaBuilder, secDefs []extractor.SecurityDefAnnotation) *spec.Components {
	comp := &spec.Components{
		Schemas: *sb.Schemas(),
	}

	if len(secDefs) > 0 {
		schemes := spec.NewOrderedSecuritySchemes()
		for _, sd := range secDefs {
			scheme := buildSecurityScheme(sd)
			schemes.Set(sd.Name, scheme)
		}
		comp.SecuritySchemes = schemes
	}

	return comp
}

func buildSecurityScheme(sd extractor.SecurityDefAnnotation) *spec.SecurityScheme {
	scheme := &spec.SecurityScheme{
		Type:        sd.Type,
		Description: sd.Description,
	}

	switch sd.Type {
	case "apiKey":
		scheme.Name = sd.FieldName
		scheme.In = sd.In
	case "http":
		scheme.Scheme = sd.Scheme
		if sd.BearerFormat != "" {
			scheme.BearerFormat = sd.BearerFormat
		}
	case "oauth2":
		scopes := spec.NewOrderedStrings()
		for name, desc := range sd.Scopes {
			d := desc
			scopes.Set(name, &d)
		}
		flow := spec.OAuthFlow{
			AuthorizationURL: sd.AuthorizationURL,
			TokenURL:         sd.TokenURL,
			Scopes:           scopes,
		}
		switch sd.OAuthFlowType {
		case "implicit":
			scheme.Flows.Implicit = flow
		case "password":
			scheme.Flows.Password = flow
		case "clientCredentials":
			scheme.Flows.ClientCredentials = flow
		case "authorizationCode":
			scheme.Flows.AuthorizationCode = flow
		}
	case "openIdConnect":
		scheme.OpenIDConnectURL = sd.OpenIDConnectURL
	}

	return scheme
}

func buildTags(tags []extractor.TagAnnotation) []spec.Tag {
	if len(tags) == 0 {
		return nil
	}
	out := make([]spec.Tag, len(tags))
	for i, t := range tags {
		out[i] = spec.Tag{
			Name:        t.Name,
			Description: t.Description,
		}
	}
	return out
}

// findCandidates returns registered schema names that partially match the unknown type name.
func (b *Builder) findCandidates(r *Resolver, unknown string) []string {
	lowerUnknown := strings.ToLower(unknown)
	var candidates []string
	for name := range r.registered {
		lower := strings.ToLower(name)
		if strings.Contains(lower, lowerUnknown) || strings.Contains(lowerUnknown, lower) {
			candidates = append(candidates, name)
		}
	}
	sort.Strings(candidates)
	return candidates
}

func setOperationOnPathItem(item *spec.PathItem, method string, op *spec.Operation) {
	switch method {
	case "GET":
		item.Get = op
	case "POST":
		item.Post = op
	case "PUT":
		item.Put = op
	case "DELETE":
		item.Delete = op
	case "PATCH":
		item.Patch = op
	case "HEAD":
		item.Head = op
	case "OPTIONS":
		item.Options = op
	case "TRACE":
		item.Trace = op
	}
}

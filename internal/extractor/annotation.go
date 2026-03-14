package extractor

// DiagLevel is the severity of a diagnostic message.
type DiagLevel string

const (
	DiagWarn  DiagLevel = "warn"
	DiagError DiagLevel = "error"
)

// Diagnostic is a warning or error produced during annotation extraction or building.
type Diagnostic struct {
	Level    DiagLevel
	FilePath string
	Line     int
	Message  string
}

type ExtractResult struct {
	Global      GlobalAnnotation
	Operations  []OperationAnnotation
	Diagnostics []Diagnostic
}

type GlobalAnnotation struct {
	Title          string
	Description    string
	Version        string
	TermsOfService string
	Contact        ContactAnnotation
	License        LicenseAnnotation

	// OpenAPI 3: servers replace host/basePath/schemes.
	// If Servers is empty but Host is set, the builder synthesizes a Server from Host+BasePath+Schemes.
	Host     string
	BasePath string
	Schemes  []string
	Servers  []ServerAnnotation

	ExternalDocs *ExternalDocsAnnotation
	Tags         []TagAnnotation
	SecurityDefs []SecurityDefAnnotation
}

type ContactAnnotation struct {
	Name  string
	URL   string
	Email string
}

type LicenseAnnotation struct {
	Name string
	URL  string
}

type ServerAnnotation struct {
	URL         string
	Description string
}

type ExternalDocsAnnotation struct {
	URL         string
	Description string
}

type TagAnnotation struct {
	Name        string
	Description string
}

// SecurityDefAnnotation models a security scheme definition.
// Supports: apiKey, http (basic/bearer), oauth2, openIdConnect.
type SecurityDefAnnotation struct {
	Name             string
	Type             string // "apiKey", "http", "oauth2", "openIdConnect"
	In               string // apiKey: "header", "query", "cookie"
	FieldName        string // apiKey: header/query param name
	Scheme           string // http: "basic", "bearer"
	BearerFormat     string // http bearer: e.g. "JWT"
	Description      string
	OAuthFlowType    string            // oauth2: "implicit", "password", "clientCredentials", "authorizationCode"
	AuthorizationURL string            // oauth2
	TokenURL         string            // oauth2
	Scopes           map[string]string // oauth2: scope name → description
	OpenIDConnectURL string            // openIdConnect
}

type OperationAnnotation struct {
	FuncName    string
	FilePath    string
	Line        int
	Tags        []string
	Summary     string
	Description string
	OperationID string
	Route       RouteInfo
	Accept      []string // request content types (e.g. "application/json")
	Produce     []string // response content types
	Params      []ParamAnnotation
	RequestBody *RequestBodyAnnotation
	Responses   []ResponseAnnotation
	Security    []SecurityRequirement
	Deprecated  bool
}

type RouteInfo struct {
	Method string
	Path   string
}

type ParamAnnotation struct {
	Name        string
	In          string // "path", "query", "header", "cookie"
	TypeName    string
	Required    bool
	Description string
	Format      string
	Default     string
	Enums       []string
}

// TypeExpr represents a possibly-composite type expression.
// Simple:    TypeName="User", Overrides=nil
// Composite: TypeName="PageData", Overrides=[{Field:"data", TypeExpr:"[]User"}]
type TypeExpr struct {
	Name      string
	Overrides []FieldOverride
}

// FieldOverride maps a struct field name to a replacement type expression.
// e.g. PageData{data=[]User} → Field="data", TypeExpr="[]User"
type FieldOverride struct {
	Field    string
	TypeExpr string
}

// RequestBodyAnnotation represents @Param with in=body or in=formData.
type RequestBodyAnnotation struct {
	TypeName    string
	Type        TypeExpr
	Required    bool
	Description string
	IsForm      bool // true if formData, false if JSON body
	Fields      []FormFieldAnnotation
}

type FormFieldAnnotation struct {
	Name        string
	TypeName    string
	Required    bool
	Description string
}

// SecurityRequirement is a scheme name with optional scopes.
type SecurityRequirement struct {
	Name   string
	Scopes []string
}

type ResponseAnnotation struct {
	Code        string
	TypeName    string
	Type        TypeExpr
	Description string
	IsArray     bool
	IsPrimitive bool // true for {string}, {integer}, {boolean}, {number}
	Headers     []ResponseHeaderAnnotation
}

type ResponseHeaderAnnotation struct {
	Code        string
	Name        string
	TypeName    string
	Description string
}

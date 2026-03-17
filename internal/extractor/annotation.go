package extractor

// ExtractResult 是 GoExtractor.Extract 的输出：全局注解 + 所有操作注解。
type ExtractResult struct {
	Global     GlobalAnnotation
	Operations []OperationAnnotation
}

// GlobalAnnotation 对应入口函数（通常是 main）上的全局注解。
type GlobalAnnotation struct {
	Title          string
	Description    string
	Version        string
	TermsOfService string

	Contact      ContactAnnotation
	License      LicenseAnnotation
	ExternalDocs ExternalDocsAnnotation

	// swaggo 兼容：@host / @BasePath / @schemes → builder 阶段合成 servers[]
	Host     string
	BasePath string
	Schemes  []string

	Servers      []ServerAnnotation
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

type ExternalDocsAnnotation struct {
	URL         string
	Description string
}

type ServerAnnotation struct {
	URL         string
	Description string
}

type TagAnnotation struct {
	Name        string
	Description string
}

// SecurityDefAnnotation 描述一条安全方案定义（来自 @securityDefinitions.*）。
type SecurityDefAnnotation struct {
	Name        string
	Type        string // "apiKey" | "http" | "oauth2" | "openIdConnect"
	Description string

	// apiKey
	In      string // "header" | "query" | "cookie"
	KeyName string // 参数名

	// http
	Scheme       string // "basic" | "bearer"
	BearerFormat string

	// oauth2
	Flows []OAuthFlowAnnotation

	// openIdConnect
	OpenIDConnectURL string
}

// OAuthFlowAnnotation 对应 OAuth2 的一种授权流。
type OAuthFlowAnnotation struct {
	Type             string            // "implicit" | "password" | "clientCredentials" | "authorizationCode"
	AuthorizationURL string
	TokenURL         string
	Scopes           map[string]string // scope name → description
}

// OperationAnnotation 对应一个带 @Router 注解的处理函数。
type OperationAnnotation struct {
	FuncName    string
	FilePath    string
	Line        int
	Tags        []string
	Summary     string
	Description string
	OperationID string
	Accept      []string
	Produce     []string
	Route       RouteInfo
	Params      []ParamAnnotation
	Responses   []ResponseAnnotation
	Headers     []HeaderAnnotation
	Security    []SecurityRequirement
	Deprecated  bool
}

type RouteInfo struct {
	Method string
	Path   string
}

// ParamAnnotation 对应一行 @Param 注解。
type ParamAnnotation struct {
	Name        string
	In          string // "path" | "query" | "header" | "cookie" | "body" | "formData"
	TypeName    string
	Required    bool
	Description string
	Format      string
}

// ResponseAnnotation 对应一行 @Success / @Failure 注解。
type ResponseAnnotation struct {
	Code        string
	TypeName    string
	Description string
	IsArray     bool
	WrapType    string // "object" | "array" | "string" | "integer" | "number" | "boolean"
}

// HeaderAnnotation 对应一行 @Header 注解。
type HeaderAnnotation struct {
	Code        string
	TypeName    string
	Name        string
	Description string
}

// SecurityRequirement 对应操作级别的 @Security 注解。
type SecurityRequirement struct {
	Name   string
	Scopes []string
}

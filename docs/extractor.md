# extractor 包（Stage 2）

**代码位置**：`internal/extractor/`

| 文件 | 职责 |
|------|------|
| `extractor.go` | `GoExtractor`：遍历 `RawFunc.Comments`，分发各标签的解析 |
| `annotation.go` | 注解数据结构：`ExtractResult`、`OperationAnnotation` 等 |
| `tag_parser.go` | 各 `@Tag` 的结构化解析逻辑（`@Param`、`@Success`、`@Router` 等） |

## 职责

从 `[]*parser.RawFile` 的注释中提取结构化 API 注解，产出 `*ExtractResult`。不做类型解析，不操作 spec3。

## 核心类型

### `GoExtractor`

```go
type GoExtractor struct{}

func (e *GoExtractor) Extract(files []*parser.RawFile) (*ExtractResult, error)
```

**过滤规则**：只处理包含 `@Router` 的函数注释，无 `@Router` 的函数忽略。

### `ExtractResult`

```go
type ExtractResult struct {
    Global     GlobalAnnotation
    Operations []OperationAnnotation
}
```

### `GlobalAnnotation`

从入口函数（通常是 `main()`）的注释中提取：

```go
type GlobalAnnotation struct {
    Title          string
    Description    string
    Version        string
    TermsOfService string
    Contact        ContactAnnotation
    License        LicenseAnnotation
    ExternalDocs   ExternalDocsAnnotation
    Host           string          // 兼容 swaggo：合成 servers[]
    BasePath       string          // 兼容 swaggo
    Schemes        []string        // 兼容 swaggo
    Servers        []ServerAnnotation
    Tags           []TagAnnotation
    SecurityDefs   []SecurityDefAnnotation
}
```

### `OperationAnnotation`

每个带 `@Router` 的函数对应一个 `OperationAnnotation`：

```go
type OperationAnnotation struct {
    FuncName    string
    FilePath    string
    Line        int
    Tags        []string
    Summary     string
    Description string    // 无 @Description 时由非标签注释行自动拼接
    OperationID string
    Accept      []string  // MIME 类型
    Produce     []string
    Route       RouteInfo
    Params      []ParamAnnotation
    Responses   []ResponseAnnotation
    Headers     []HeaderAnnotation
    Security    []SecurityRequirement
    Deprecated  bool
}

type RouteInfo struct {
    Method string // 大写，如 "GET"
    Path   string // 如 "/users/{id}"
}
```

### `ParamAnnotation`

对应一条 `@Param` 注解：

```go
type ParamAnnotation struct {
    Name        string // "" 表示打散 struct
    In          string // path / query / header / cookie / body / formData
    TypeName    string
    Required    bool
    Description string
    Format      string
}
```

### `ResponseAnnotation`

对应一条 `@Success` 或 `@Failure` 注解：

```go
type ResponseAnnotation struct {
    Code        string // 状态码字符串，如 "200"
    WrapType    string // "object" | "array" | "string" | "integer" | "number" | "boolean"
    TypeName    string
    Description string
    IsArray     bool
}
```

### `HeaderAnnotation`

```go
type HeaderAnnotation struct {
    Code        string // 状态码字符串，如 "200"
    TypeName    string
    Name        string
    Description string
}
```

### `SecurityDefAnnotation`

不同安全类型使用不同字段组合：

```go
type SecurityDefAnnotation struct {
    Name        string
    Type        string // "apiKey" | "http" | "oauth2" | "openIdConnect"
    Description string

    // apiKey 专用
    In      string // "header" | "query" | "cookie"
    KeyName string // 参数名

    // http 专用
    Scheme       string // "basic" | "bearer"
    BearerFormat string

    // oauth2 专用
    Flows []OAuthFlowAnnotation

    // openIdConnect 专用
    OpenIDConnectURL string
}
```

### `OAuthFlowAnnotation`

```go
type OAuthFlowAnnotation struct {
    Type             string            // "implicit" | "password" | "clientCredentials" | "authorizationCode"
    AuthorizationURL string
    TokenURL         string
    Scopes           map[string]string // scope name → description
}
```

### `SecurityRequirement`

```go
type SecurityRequirement struct {
    Name   string
    Scopes []string // OAuth2 scope 列表；非 OAuth2 方案为空
}
```

## 标签解析规则（tag_parser.go）

标签名**不区分大小写**。

| 标签 | 解析格式 |
|------|---------|
| `@Router` | `路径 [方法]` |
| `@Param` | `名称 位置 类型 必填 [格式] "描述"` |
| `@Success` / `@Failure` | `状态码 {包装类型} 类型名 "描述"` |
| `@Header` | `状态码 {类型} 头名称 "描述"` |
| `@Security` | `方案名` 或 `方案名[scope1, scope2]` |
| `@Tags` | 逗号分隔的标签名列表 |
| `@Accept` / `@Produce` | 逗号分隔的 MIME 别名或完整 MIME 类型 |

# swaggin 包

**代码位置**：`pkg/swaggin/`

## 职责

将 swag3 生成的 OpenAPI 规范挂载到 Gin 路由，提供 Swagger UI、Redoc、Fdoc 三种文档界面。是本项目唯一的公开 API 包，不依赖任何 `internal/` 包。

## 类型

### `Renderer`

```go
type Renderer string

const (
    SwaggerUI Renderer = "swagger-ui" // 默认，渲染 Swagger UI
    Redoc     Renderer = "redoc"      // 渲染 Redoc
    Fdoc      Renderer = "fdoc"       // 渲染 Fdoc（@braydenyang/fdoc）
)
```

### `Options`

```go
type Options struct {
    SpecFile    string   // openapi.json 文件路径（每次请求时从磁盘读取，支持热重载）
    SpecContent []byte   // 内联 spec 内容（优先级高于 SpecFile）
    JSONPath    string   // 原始 spec 的 URL 路径，默认 "/openapi.json"
    UIPath      string   // UI 的 URL 路径，默认 "/docs"；设为 "-" 可禁用
    Renderer    Renderer // UI 类型，默认 SwaggerUI
    RedocPath   string   // 额外挂载 Redoc 的路径；留空不注册，设为 "-" 显式禁用
    FdocPath    string   // 额外挂载 Fdoc 的路径；留空不注册，设为 "-" 显式禁用
    FaviconPath string   // favicon 路由路径，默认 "/favicon.ico"；设为 "-" 禁用
    Title       string   // HTML 页面标题，默认 "API Documentation"
    AllowCORS   bool     // 是否启用 CORS
    CORSOrigin  string   // CORS 允许的 Origin，默认 "*"
}
```

## 函数

### `Register`

```go
func Register(r gin.IRouter, opts Options)
```

一次性注册所有路由：

| 路由 | 说明 |
|------|------|
| `GET {JSONPath}` | 返回原始 OpenAPI JSON |
| `GET {UIPath}` | 返回 Swagger UI 或 Redoc HTML（`UIPath != "-"` 时注册） |
| `GET {RedocPath}` | 返回 Redoc HTML（`RedocPath` 非空且不为 `"-"` 时注册） |
| `GET {FdocPath}` | 返回 Fdoc HTML（`FdocPath` 非空且不为 `"-"` 时注册） |
| `GET {FaviconPath}` | 返回内嵌的 `favicon.svg`，默认路径 `/favicon.ico` |
| `OPTIONS *` | CORS 预检，每条 GET 路由均注册（仅 `AllowCORS=true` 时） |

### Handler 单独使用

```go
func SpecHandler(opts Options) gin.HandlerFunc  // 原始 spec
func UIHandler(opts Options) gin.HandlerFunc    // Swagger UI / Redoc / Fdoc（由 Renderer 决定）
func RedocHandler(opts Options) gin.HandlerFunc // 始终渲染 Redoc
func FdocHandler(opts Options) gin.HandlerFunc  // 始终渲染 Fdoc
```

## 快速集成示例

```go
import "github.com/shuttlefy/go-openapi3-swag/pkg/swaggin"

func main() {
    r := gin.Default()

    // 业务路由 ...

    swaggin.Register(r, swaggin.Options{
        SpecFile:  "docs/openapi.json",
        Title:     "My API",
        JSONPath:  "/openapi.json",
        UIPath:    "/docs",
        RedocPath: "/redoc",
        AllowCORS: true,
    })

    r.Run(":8080")
}
```

启动后访问：
- Swagger UI：`http://localhost:8080/docs`
- Redoc：`http://localhost:8080/redoc`
- 原始 JSON：`http://localhost:8080/openapi.json`

## 注意事项

- `SpecFile` 在每次请求时从磁盘读取，无需重启即可热重载（重新运行 `swag3` 生成后立即生效）
- `SpecContent` 优先级高于 `SpecFile`，适合将 spec 内嵌到二进制中
- `AllowCORS=true` 时，每条路由的响应中自动写入以下 4 个 CORS 响应头，并为每条路由额外注册 `OPTIONS` 预检处理器：
  ```
  Access-Control-Allow-Origin:  <CORSOrigin>（默认 "*"）
  Access-Control-Allow-Methods: GET, OPTIONS
  Access-Control-Allow-Headers: Origin, Accept, Content-Type, Authorization
  Access-Control-Max-Age:       86400
  ```

# swag3

Go 源码 → OpenAPI 3.x 规范生成器，通过解析函数注释中的标签自动生成 OpenAPI JSON / YAML 文档，并可通过 `swaggin` 包将文档界面挂载到 Gin 路由。

---

## 目录

- [安装](#安装)
- [快速开始](#快速开始)
- [CLI 参数](#cli-参数)
- [编写注解](#编写注解)
  - [全局注解](#全局注解)
  - [操作注解](#操作注解)
  - [Struct Tag 约束](#struct-tag-约束)
- [集成到 Gin（swaggin）](#集成到-ginswaggin)
- [完整示例](#完整示例)

---

## 安装

```bash
go install github.com/shuttlefy/go-openapi3-swag/cmd/swag3@latest
```

验证安装：

```bash
swag3 -help
```

---

## 快速开始

**第一步**：在项目入口（通常是 `main()` 函数）的注释中写全局注解：

```go
// @title           My API
// @version         1.0.0
// @description     A simple API.
// @server          http://localhost:8080 "Local"
func main() { ... }
```

**第二步**：在每个 HTTP 处理函数的注释中写操作注解：

```go
// @Summary  获取用户
// @Tags     users
// @Produce  json
// @Param    id   path  integer  true  int64 "用户 ID"
// @Success  200 {object} models.User "用户信息"
// @Failure  404 {object} models.ErrorResponse "用户不存在"
// @Router   /users/{id} [get]
func GetUser(c *gin.Context) { ... }
```

**第三步**：运行 swag3 生成文档：

```bash
swag3 -dirs . -output docs/openapi.json
```

---

## CLI 参数

```bash
swag3 -dirs <目录> [选项]
```

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-dirs` | **必填** | 要扫描的源码目录，逗号分隔或多次传入 |
| `-output` | `./docs/openapi.json` | 输出路径，扩展名决定格式（`.json` / `.yaml`） |
| `-openapi-ver` | `3.2.0` | OpenAPI 版本，支持 `3.0.3` / `3.1.0` / `3.2.0` |
| `-depth` | `-1` | 目录递归深度：`-1` 无限，`0` 仅当前目录，`N` 最多 N 层 |
| `-gomod` | `go.mod` | go.mod 路径，用于跨模块类型解析 |

**示例：**

```bash
# 扫描当前目录，输出 JSON
swag3 -dirs .

# 多目录扫描
swag3 -dirs ./cmd,./internal -output docs/openapi.json

# 输出 YAML，指定 OpenAPI 版本
swag3 -dirs . -output docs/openapi.yaml -openapi-ver 3.1.0

# 仅扫描 ./api 目录，不递归子目录
swag3 -dirs ./api -depth 0
```

---

## 编写注解

注解以 `// @标签名 值` 格式写在 Go 函数注释中，**标签名不区分大小写**。

> 完整标签参考见 [annotations.md](annotations.md)。

### 全局注解

写在 `main()` 函数的注释中，声明 API 的基本信息、服务器地址和安全方案。

```go
// @title           Bookstore API
// @version         1.0.0
// @description     书店管理 API，支持书籍的增删改查。
// @termsOfService  https://example.com/terms
//
// @contact.name    API Support
// @contact.email   support@example.com
//
// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT
//
// @server https://api.example.com "Production"
// @server http://localhost:8080   "Development"
//
// @tag books "书籍管理"
// @tag users "用户管理"
//
// @securityDefinitions.bearer BearerAuth
// @securityDefinitions.bearer.bearerFormat JWT
// @securityDefinitions.bearer.description  JWT Bearer Token
func main() { ... }
```

**安全方案一览：**

| 类型 | 示例 |
|------|------|
| API Key | `@securityDefinitions.apikey ApiKeyAuth` |
| HTTP Basic | `@securityDefinitions.basic BasicAuth` |
| HTTP Bearer | `@securityDefinitions.bearer BearerAuth` |
| OAuth2 授权码 | `@securityDefinitions.oauth2.authorizationCode OAuth2` |
| OpenID Connect | `@securityDefinitions.openIdConnect MyOIDC` |

---

### 操作注解

写在每个 HTTP 处理函数的注释中。**必须包含 `@Router`**，否则该函数被忽略。

```go
// @Summary     创建书籍
// @Description 创建一条新书记录，title 和 author 为必填字段。
// @ID          createBook
// @Tags        books
// @Accept      json
// @Produce     json
// @Param       body  body  models.CreateBookRequest  true  "书籍信息"
// @Success     201 {object} models.Book "创建成功"
// @Failure     400 {object} models.ErrorResponse "参数校验失败"
// @Failure     401 {object} models.ErrorResponse "未授权"
// @Security    BearerAuth
// @Router      /books [post]
func CreateBook(c *gin.Context) { ... }
```

**常用标签：**

| 标签 | 说明 |
|------|------|
| `@Summary` | 操作摘要（一行） |
| `@Description` | 详细描述 |
| `@Tags` | 所属标签组（逗号分隔） |
| `@Accept` | 请求 Content-Type（`json`, `mpfd` 等） |
| `@Produce` | 响应 Content-Type |
| `@Param` | 参数声明（见下） |
| `@Success` | 成功响应 |
| `@Failure` | 失败响应 |
| `@Security` | 安全要求 |
| `@Router` | 路由路径和 HTTP 方法（**必填**） |
| `@Deprecated` | 标记为已弃用 |

**`@Param` 格式：**

```
// @Param 名称 位置 类型 必填 "描述"
```

- **位置**：`path` / `query` / `header` / `cookie` / `body` / `formData`
- **类型**：原始类型（`string` / `integer` / `number` / `boolean` / `file`）或模型引用（`包名.类型名`）

```go
// @Param id       path   integer              true  int64 "用户 ID"
// @Param status   query  string               false "状态过滤"
// @Param body     body   models.CreateRequest true  "请求体"
// @Param file     formData file              true  "上传文件"
```

**Struct 打散参数**（将一个 struct 的字段展开为独立的 query 参数）：

```go
// @Param "" query models.ListQuery false "查询条件"
```

**`@Success` / `@Failure` 格式：**

```
// @Success 状态码 {类型} 模型名 "描述"
```

```go
// @Success 200 {object}  models.User       "用户对象"
// @Success 200 {array}   []models.User     "用户列表"
// @Success 204           "No Content"
// @Failure 400 {object}  models.ErrorResp  "参数错误"
```

**组合类型**（用于泛型包装，如分页）：

```go
// @Success 200 {object} common.PageData{data=[]models.Book} "书籍分页列表"
```

**模型引用规则：**

注解中引用 Go 类型时，格式为 `包名.类型名`（不可省略包名）：

```go
// 正确
// @Success 200 {object} models.User "OK"
// @Param   body body    models.CreateRequest true "body"

// 错误 — 缺少包名
// @Success 200 {object} User "OK"
```

---

### Struct Tag 约束

在 Go struct 字段的 tag 中声明 OpenAPI Schema 约束，swag3 自动提取并映射：

```go
type Product struct {
    Name      string   `json:"name"      description:"商品名称" minLength:"1" maxLength:"200" example:"Widget"`
    Price     float64  `json:"price"     minimum:"0"    maximum:"99999.99" format:"double"`
    SKU       string   `json:"sku"       pattern:"^[A-Z]{2}-\\d{6}$"`
    Quantity  int      `json:"quantity"  minimum:"0"    default:"1"`
    Tags      []string `json:"tags"      minItems:"1"   maxItems:"10"    uniqueItems:"true"`
    State     string   `json:"state"     enums:"active,inactive" default:"active"`
    CreatedAt string   `json:"created_at" format:"date-time" readonly:"true"`
    Password  string   `json:"password"  writeonly:"true" minLength:"8"`
    Internal  string   `json:"-"`                   // 自动排除
}
```

**常用 Tag：**

| Tag | 映射到 | 说明 |
|-----|--------|------|
| `json:"name"` | 属性名 | JSON 序列化名称 |
| `json:"-"` | — | 排除该字段 |
| `binding:"required"` / `validate:"required"` | `required` | 必填字段 |
| `description:"text"` | `description` | 字段描述 |
| `example:"value"` | `example` | 示例值 |
| `enums:"a,b,c"` | `enum` | 枚举值 |
| `format:"fmt"` | `format` | 数据格式（`date-time`, `email`, `uuid` 等） |
| `default:"value"` | `default` | 默认值 |
| `minimum:"n"` / `maximum:"n"` | `minimum` / `maximum` | 数值范围 |
| `minLength:"n"` / `maxLength:"n"` | `minLength` / `maxLength` | 字符串长度 |
| `pattern:"regex"` | `pattern` | 正则校验 |
| `minItems:"n"` / `maxItems:"n"` | `minItems` / `maxItems` | 数组长度 |
| `uniqueItems:"true"` | `uniqueItems` | 数组元素唯一 |
| `readonly:"true"` | `readOnly` | 仅出现在响应中 |
| `writeonly:"true"` | `writeOnly` | 仅出现在请求中 |
| `deprecated:"true"` | `deprecated` | 字段已废弃 |

> 指针类型（`*string`、`*int` 等）自动映射为 `nullable: true`。

---

## 集成到 Gin（swaggin）

`pkg/swaggin` 将生成的 OpenAPI 规范挂载到 Gin 路由，提供 Swagger UI、Redoc、Fdoc 三种文档界面。

**安装：**

```bash
go get github.com/shuttlefy/go-openapi3-swag/pkg/swaggin
```

**注册路由：**

```go
import "github.com/shuttlefy/go-openapi3-swag/pkg/swaggin"

func main() {
    r := gin.Default()

    // 业务路由 ...

    swaggin.Register(r, swaggin.Options{
        SpecFile:  "docs/openapi.json", // 每次请求时从磁盘读取，支持热重载
        Title:     "My API",
        JSONPath:  "/openapi.json",
        UIPath:    "/docs",             // Swagger UI
        RedocPath: "/redoc",            // Redoc
        FdocPath:  "/fdoc",             // Fdoc
        AllowCORS: true,
    })

    r.Run(":8080")
}
```

启动后访问：

| 地址 | 内容 |
|------|------|
| `http://localhost:8080/docs` | Swagger UI |
| `http://localhost:8080/redoc` | Redoc |
| `http://localhost:8080/fdoc` | Fdoc |
| `http://localhost:8080/openapi.json` | 原始 JSON Spec |

**`Options` 配置项：**

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `SpecFile` | — | OpenAPI JSON 文件路径（热重载，每次请求时读取） |
| `SpecContent` | — | 内联 Spec 内容，优先级高于 `SpecFile` |
| `JSONPath` | `/openapi.json` | 原始 Spec 的 URL 路径 |
| `UIPath` | `/docs` | 文档 UI 的 URL 路径；`"-"` 禁用 |
| `Renderer` | `SwaggerUI` | UI 类型：`SwaggerUI` / `Redoc` / `Fdoc` |
| `RedocPath` | — | 额外挂载 Redoc 的路径 |
| `FdocPath` | — | 额外挂载 Fdoc 的路径 |
| `FaviconPath` | `/favicon.ico` | Favicon 路径；`"-"` 禁用 |
| `Title` | `API Documentation` | HTML 页面标题 |
| `AllowCORS` | `false` | 是否启用 CORS |
| `CORSOrigin` | `*` | CORS 允许的 Origin |

---

## 完整示例

以下是一个书店 API 的完整示例，位于 [`examples/bookstore/`](examples/bookstore/)。

**项目结构：**

```
bookstore/
├── main.go          # 全局注解 + Gin 路由 + swaggin 注册
├── handler.go       # 操作注解
├── models/
│   └── models.go    # 数据模型（含 Struct Tag 约束）
├── common/
│   └── response.go  # 通用响应结构
└── docs/
    └── openapi.json # 由 swag3 生成
```

**生成文档：**

```bash
cd examples/bookstore
swag3 -dirs . -output docs/openapi.json
```

**启动服务：**

```bash
go run .
```

**注解示例摘录：**

```go
// main.go — 全局注解
//
// @title           Bookstore API
// @version         1.0.0
// @server          http://localhost:9999 "Local"
//
// @securityDefinitions.bearer BearerAuth
// @securityDefinitions.bearer.bearerFormat JWT
func main() { ... }

// handler.go — 操作注解
//
// @Summary  列出书籍
// @Tags     books
// @Produce  json
// @Param    category  query   string   false "分类过滤"
// @Param    page      query   integer  false int32 "页码"
// @Success  200 {object} common.PageData{list=[]models.Book} "分页列表"
// @Security ApiKeyAuth
// @Router   /books [get]
func listBooks(c *gin.Context) { ... }
```

---

## 参考文档

| 文档 | 说明 |
|------|------|
| [annotations.md](annotations.md) | 所有注解标签完整参考（含安全方案、组合类型等） |
| [docs/](docs/) | 内部架构文档（CLI、Parser、Builder 等各模块） |

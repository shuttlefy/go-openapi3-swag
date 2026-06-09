# swag3 注释标签参考

本文档定义了 swag3 支持的所有注释标签。标签写在 Go 函数或包级别的注释中，格式为 `// @TagName value`。

标签名**不区分大小写**。

---

## 一、全局注解（General API Info）

写在入口函数（通常是 `main()`）的注释中。

### 基本信息

| 标签 | 说明 | 示例 |
|------|------|------|
| `@title` | API 标题（**必填**） | `// @title Pet Store API` |
| `@version` | API 版本（**必填**） | `// @version 1.0.0` |
| `@description` | API 描述 | `// @description A sample pet store server.` |
| `@termsOfService` | 服务条款 URL | `// @termsOfService https://example.com/terms` |

### 联系人信息

| 标签 | 说明 | 示例 |
|------|------|------|
| `@contact.name` | 联系人姓名 | `// @contact.name API Support` |
| `@contact.url` | 联系人 URL | `// @contact.url https://www.example.com/support` |
| `@contact.email` | 联系人邮箱 | `// @contact.email support@example.com` |

### 许可证

| 标签 | 说明 | 示例 |
|------|------|------|
| `@license.name` | 许可证名称 | `// @license.name Apache 2.0` |
| `@license.url` | 许可证 URL | `// @license.url https://www.apache.org/licenses/LICENSE-2.0.html` |

### 服务器（OpenAPI 3）

使用 `@server` 定义 API 服务器列表。可多次声明。

```
// @server URL "描述"
```

| 示例 |
|------|
| `// @server https://api.example.com "Production"` |
| `// @server https://staging.example.com "Staging"` |
| `// @server http://localhost:8080` |

> **兼容模式**：也支持旧版 `@host` / `@BasePath` / `@schemes`，builder 会自动合成为 `servers[]`。
>
> | 标签 | 示例 |
> |------|------|
> | `@host` | `// @host localhost:8080` |
> | `@BasePath` | `// @BasePath /api/v1` |
> | `@schemes` | `// @schemes http https` |

### 外部文档

| 标签 | 说明 | 示例 |
|------|------|------|
| `@externalDocs.url` | 外部文档 URL | `// @externalDocs.url https://example.com/docs` |
| `@externalDocs.description` | 外部文档描述 | `// @externalDocs.description Find out more` |

### 标签定义

声明全局 API 标签。可多次声明。

```
// @tag 名称 "描述"
```

| 示例 |
|------|
| `// @tag users "User management"` |
| `// @tag pets "Pet operations"` |

---

## 二、安全方案定义（Security Definitions）

写在全局注解（`main` 函数）的注释中，采用多行声明模式。每种方案固定两段式：

```
// @securityDefinitions.{类型}           方案名称   ← 第一行：声明方案，值为方案名
// @securityDefinitions.{类型}.{子属性}  值          ← 后续行：设置子属性
```

标签名**不区分大小写**。支持的类型汇总：

| 类型关键字 | OpenAPI type | 说明 |
|------------|-------------|------|
| `apikey` | `apiKey` | API Key |
| `basic` | `http` (scheme=basic) | HTTP Basic 认证 |
| `bearer` | `http` (scheme=bearer) | HTTP Bearer Token |
| `oauth2.implicit` | `oauth2` | OAuth2 隐式授权 |
| `oauth2.password` | `oauth2` | OAuth2 密码授权 |
| `oauth2.clientCredentials` | `oauth2` | OAuth2 客户端凭证 |
| `oauth2.authorizationCode` | `oauth2` | OAuth2 授权码 |
| `openIdConnect` | `openIdConnect` | OpenID Connect |

---

### API Key

```go
// @securityDefinitions.apikey ApiKeyAuth
// @securityDefinitions.apikey.in     header
// @securityDefinitions.apikey.name   X-API-Key
// @securityDefinitions.apikey.description API key 认证，写入请求头 X-API-Key
```

| 子属性 | 必填 | 说明 | 可选值 |
|--------|------|------|--------|
| `.in` | ✓ | 传递位置 | `header`, `query`, `cookie` |
| `.name` | ✓ | Header / 参数名称 | 任意字符串 |
| `.description` | — | 描述 | 任意字符串 |

---

### HTTP Basic

```go
// @securityDefinitions.basic BasicAuth
// @securityDefinitions.basic.description HTTP Basic 认证
```

自动设置 `type=http`、`scheme=basic`。

| 子属性 | 必填 | 说明 |
|--------|------|------|
| `.description` | — | 描述 |

---

### HTTP Bearer

```go
// @securityDefinitions.bearer BearerAuth
// @securityDefinitions.bearer.bearerFormat JWT
// @securityDefinitions.bearer.description JWT Bearer Token 认证
```

| 子属性 | 必填 | 说明 |
|--------|------|------|
| `.bearerFormat` | — | Token 格式，如 `JWT` |
| `.description` | — | 描述 |

---

### OAuth2

OAuth2 需要在类型关键字中指定流类型，每种流类型所需 URL 不同：

| 流类型 | 需要 `authorizationUrl` | 需要 `tokenUrl` |
|--------|------------------------|----------------|
| `oauth2.implicit` | ✓ | — |
| `oauth2.password` | — | ✓ |
| `oauth2.clientCredentials` | — | ✓ |
| `oauth2.authorizationCode` | ✓ | ✓ |

```go
// @securityDefinitions.oauth2.authorizationCode OAuth2
// @securityDefinitions.oauth2.authorizationCode.authorizationUrl https://example.com/oauth/authorize
// @securityDefinitions.oauth2.authorizationCode.tokenUrl         https://example.com/oauth/token
// @securityDefinitions.oauth2.authorizationCode.scope.read       "只读权限"
// @securityDefinitions.oauth2.authorizationCode.scope.write      "写权限"
// @securityDefinitions.oauth2.authorizationCode.description      OAuth2 授权码模式
```

| 子属性 | 必填 | 说明 |
|--------|------|------|
| `.authorizationUrl` | 视流类型 | 授权端点 URL |
| `.tokenUrl` | 视流类型 | Token 端点 URL |
| `.scope.{名称}` | — | 作用域，值为描述字符串（可加引号） |
| `.description` | — | 描述 |

---

### OpenID Connect

```go
// @securityDefinitions.openIdConnect MyOIDC
// @securityDefinitions.openIdConnect.openIdConnectUrl https://example.com/.well-known/openid-configuration
// @securityDefinitions.openIdConnect.description OpenID Connect 认证
```

| 子属性 | 必填 | 说明 |
|--------|------|------|
| `.openIdConnectUrl` | ✓ | Discovery 文档 URL |
| `.description` | — | 描述 |

---

## 三、操作注解（Operation Annotations）

写在 API 处理函数的注释中。**必须包含 `@Router`**，否则该函数被忽略。

### 模型名引用规则

注解中凡是需要引用 Go 类型（非原始类型）的地方，**必须**使用以下格式：

```
包名(别名).[函数名.]类型名
```

| 部分 | 说明 |
|------|------|
| `包名` | Go 源文件中 `package` 声明的名称（如 `models`），**或**该包的 import alias；两者均可，最终 schema 名称统一使用真实包名 |
| `[函数名.]` | 可选；引用定义在**某函数体内部**的局部 struct 时使用 |
| `类型名` | Go 导出类型名 |

**原始类型**（`string` / `integer` / `int` / `number` / `boolean` / `file`）直接写，不加包名。

**数组**在类型名前加 `[]`：`[]models.User`

```go
// 正确 ✓ — 使用包名
// @Success 200 {object} models.User "OK"
// @Success 200 {array}  []models.User "OK"
// @Param   body body    models.CreateUserRequest true "body"
// @Success 200 {object} handlers.CreateUser.Request "OK"   // 函数局部类型

// 正确 ✓ — 使用 import alias（最终 schema 名称仍为真实包名）
// import m "github.com/example/models"
// @Success 200 {object} m.User "OK"   // → $ref: models.User

// 错误 ✗ — 缺少包名限定
// @Success 200 {object} User "OK"
// @Param   body body    CreateUserRequest true "body"
```

> 工具解析时优先按 alias 匹配，其次按包名匹配；**无论注解中写的是 alias 还是包名，最终注册到 `components/schemas` 的 key 均使用真实包名**。

**同名包冲突时必须使用别名**

若同一文件引入了两个 `package` 声明名称相同的包（如 v1、v2 多版本），在 Go 代码中必须为其中至少一个设置 import alias，注解中也使用该 alias 加以区分：

```go
import (
    modelsv1 "github.com/example/api/v1/models"  // package models
    modelsv2 "github.com/example/api/v2/models"  // package models（同名）
)

// @Param body body modelsv1.CreateRequest true "v1 请求体"
// @Param body body modelsv2.CreateRequest true "v2 请求体"
```

> 注意：上例两个包的真实包名均为 `models`，最终 schema 分别注册为 `models.CreateRequest`（v1）和 `models.CreateRequest`（v2）——**若两个包存在同名类型，会产生 key 冲突**，详见下方「Components/Schemas 命名规则」。

> **同包类型也须加包名**。注解由工具跨文件解析，无法依赖"当前包"上下文推断。

> **无需显式 import（仅限真实包名）**。当注解中使用的是真实包名（`package` 声明名称）时，即使当前文件没有对应的 `import`，工具也会在所有已扫描文件中按包名兜底查找，引用仍能正常解析。若注解中使用的是 import alias，则必须在当前文件中显式声明该 import，否则解析失败。

### Components/Schemas 命名规则

工具将所有复杂类型注册到 `components/schemas`，命名规则如下：

| 类型 | Schema 名称格式 | 示例 |
|------|----------------|------|
| 包级 struct / enum | `pkg.TypeName` | `models.User` |
| 函数内局部 struct | `pkg.FuncName.TypeName` | `handlers.CreateUser.Request` |
| 泛型实例化 | `pkg.TypeName[ArgKey,...]` | `common.Resp[models.User]` |
| 组合类型 `Base{field=T}` | **不注册**（内联 `allOf`） | — |
| 透明类型别名 `type A = B` | 穿透为 B 的名称 | — |
| 非 struct 类型定义（map 等） | 穿透，不注册自身 | — |
| 原始类型（string/int/...） | **不注册**（内联 schema） | — |

`$ref` 路径直接以 schema 名称拼接：`#/components/schemas/models.User`。

**同名包（import alias）的 schema 冲突**

Schema 名称始终使用 Go 源文件中 `package` 声明的真实包名，**import alias 不会体现在 schema 名称中**。因此，如果两个包的 `package` 声明名称相同（如都是 `models`），它们的类型会共享同一个命名空间：

```
modelsv1 "github.com/example/api/v1/models"  // package models
modelsv2 "github.com/example/api/v2/models"  // package models

modelsv1.Order  →  components/schemas/models.Order
modelsv2.Order  →  components/schemas/models.Order  ← 与上面冲突！
```

发生 key 冲突时，**先被解析到的类型胜出**，后者的 `$ref` 指向的是错误的 schema。

> **规避方法**：确保同一次扫描中不存在两个 `package` 声明名称相同且含有同名类型的包。如必须同时引用，考虑在源码层面为其中一个包通过 `package` 声明改名（如 `modelsv2`），而非仅设置 import alias。

### 基本信息

| 标签 | 说明 | 示例 |
|------|------|------|
| `@Summary` | 操作摘要（一行） | `// @Summary Get user by ID` |
| `@Description` | 操作详细描述 | `// @Description Returns a single user` |
| `@ID` | operationId（全局唯一） | `// @ID getUserById` |
| `@Tags` | 标签（逗号分隔） | `// @Tags users, admin` |
| `@Deprecated` | 标记为已弃用 | `// @Deprecated` |

> **描述回退**：如果没有 `@Description`，函数注释中的非标签行会自动拼接为描述。

### 路由

```
// @Router 路径 [方法]
```

路径使用 `{param}` 表示路径参数。方法不区分大小写。

| 示例　　　　　　　　　　　　　　　　　　　 |
| --------------------------------------------|
| `// @Router /users/{id} [get]`　　　　　　 |
| `// @Router /users [post]`　　　　　　　　 |
| `// @Router /pets/{petId}/photos [delete]` |

### 内容类型

| 标签 | 说明 | 示例 |
|------|------|------|
| `@Accept` | 请求内容类型 | `// @Accept json, xml` |
| `@Produce` | 响应内容类型 | `// @Produce json` |

支持的 MIME 别名：

| 别名 | 完整 MIME 类型 |
|------|----------------|
| `json` | `application/json` |
| `xml` | `application/xml` |
| `plain` | `text/plain` |
| `html` | `text/html` |
| `mpfd` | `multipart/form-data` |
| `x-www-form-urlencoded` | `application/x-www-form-urlencoded` |
| `json-api` | `application/vnd.api+json` |
| `json-stream` | `application/x-json-stream` |
| `octet-stream` | `application/octet-stream` |
| `png` | `image/png` |
| `jpeg` | `image/jpeg` |
| `gif` | `image/gif` |

也可以直接写完整的 MIME 类型：`// @Accept application/json`

### 参数（Parameters）

```
// @Param 名称 位置 类型 必填 "描述"
// @Param 名称 位置 类型 必填 格式 "描述"
```

| 字段 | 说明 | 可选值 |
|------|------|--------|
| 名称 | 参数名 | 任意字符串；写 `""` 表示打散 struct（见下） |
| 位置 | 参数位置 | `path`, `query`, `header`, `cookie`, `body`, `formData` |
| 类型 | 数据类型 | 原始类型：`string`, `integer`, `int`, `number`, `boolean`, `file`；或模型引用：`包名.类型名` |
| 必填 | 是否必填 | `true`, `false` |
| 格式 | 类型格式（可选） | `int32`, `int64`, `float`, `double`, `date`, `date-time`, `email`, `uri` 等 |
| 描述 | 参数描述（引号包裹） | `"User ID"` |

**示例：**

```go
// @Param id         path   int           true  "User ID"
// @Param status     query  string        false "Filter status"
// @Param X-Token    header string        true  "Auth token"
// @Param limit      query  integer       false int32 "Page size"
// @Param created_at query  string        false date-time "Creation date"
// @Param filter     query  models.Filter false "Filter object"
```

### Struct 打散参数

当名称写为 `""` 时，工具会将 struct 的每个字段展开为独立的同位置参数，适用于将请求过滤条件、分页参数等集中定义在一个 struct 中的场景。

```
// @Param "" 位置 类型引用 必填 "描述"
```

类型引用支持三种格式（与模型名引用规则一致）：

| 格式 | 说明 |
|------|------|
| `包名.TypeName` | 包级 struct |
| `包名.FuncName.TypeName` | 函数内局部 struct |
| `TypeName` | 当前包的 struct（同包引用） |

**规则：**
- 名称必须为 `""`，类型必须是 struct 引用；原始类型（`string`、`integer` 等）使用 `""` 名称属于错误。
- 仅支持 `query`、`header`、`cookie` 位置（非 `body` / `formData`）。
- struct 的字段约束（`binding:"required"`、`format`、`enums`、`example`、`minimum`/`maximum` 等 struct tag）全部保留。
- 字段名按位置选择命名 tag，与 gin 等框架的绑定行为一致：
  - `query` → `form` 优先，缺失时回退 `json`
  - `path` → `uri` 优先，缺失时回退 `json`
  - `header` → `header` 优先，缺失时回退 `json`
  - `body` → 仅 `json`
- 首个出现的 tag 决定字段名与 `-` 跳过语义。例如 query 位置上 `form:"-"` 直接跳过该字段。
- 嵌入字段（embedded struct）递归展开。
- 类型所在包无需在当前文件中 `import`，工具会在已扫描的所有文件中兜底查找。

**示例一：包级 struct**

```go
// models/query.go
type BookQuery struct {
    Category string `json:"category" enums:"fiction,science,history"`
    InStock  bool   `json:"in_stock"`
    Page     int    `json:"page"  minimum:"1" default:"1"`
    Size     int    `json:"size"  minimum:"1" maximum:"100" default:"20"`
}
```

```go
// @Param "" query models.BookQuery false "查询条件"
// @Router /books [get]
```

**示例二：函数内局部 struct**

```go
// GetSubnet 查询子网列表。
//
// @Param "" query controller.GetSubnet.Request false "查询条件"
// @Router /subnet/list [get]
func GetSubnet(c *gin.Context) {
    type Request struct {
        VpcID  string `json:"vpc_id"`
        Region string `json:"region"`
        Page   int    `json:"page" minimum:"1" default:"1"`
    }
    // ...
}
```

**示例三：使用 `form` tag（推荐用于 query / formData）**

字段名按 OpenAPI 位置选择命名 tag。query 位置应直接使用 gin 绑定用的 `form` tag：

```go
// GetAccountList 查询账号列表。
//
// @Param "" query controller.GetAccountList.Request true "请求参数"
// @Router /account/list [get]
func GetAccountList(c *gin.Context) {
    type Request struct {
        Provider *string `form:"provider_code"` // 云厂商
        Tenant   *string `form:"tenant_code"`   // 主体
        Page     int32   `form:"page"      binding:"required"`
        PageSize int32   `form:"page_size" binding:"required"`
    }
    // ...
}
```

展开后会得到 `provider_code` / `tenant_code` / `page` / `page_size` 四个 query 参数。`form:"-"` 会跳过对应字段。

以上写法等价于逐一声明各字段为独立的 `query` 参数。

### 请求体（Request Body）

在 OpenAPI 3 中，`body` 和 `formData` 参数被转换为 `requestBody`。

**JSON Body：**

```go
// @Param body body models.CreateUserRequest true "User to create"
```

**Form Data（多字段）：**

```go
// @Accept mpfd
// @Param file formData file true "Avatar file"
// @Param name formData string true "User name"
```

> body 和 formData 参数**不会**出现在 OpenAPI 3 的 `parameters` 中，而是自动映射到 `requestBody`。

### 响应（Responses）

```
// @Success 状态码 {类型} 模型名 "描述"
// @Failure 状态码 {类型} 模型名 "描述"
```

`模型名` 须遵循**模型名引用规则**：`包名(别名).[函数名.]类型名`

| 类型包装器 | 模型名要求 | 说明 |
|------------|-----------|------|
| `{object}` | `包名.类型名` | 对象类型，引用模型 Schema |
| `{array}` | `[]包名.类型名` | 数组类型，items 引用模型 Schema |
| `{string}` | `string` | 原始 string 类型 |
| `{integer}` | `integer` | 原始 integer 类型 |
| `{number}` | `number` | 原始 number 类型 |
| `{boolean}` | `boolean` | 原始 boolean 类型 |

**示例：**

```go
// @Success 200 {object} models.UserResponse "OK"
// @Success 200 {array}  []models.UserResponse "Users list"
// @Success 200 {string} string "A raw string"
// @Failure 400 {object} models.ErrorResponse "Bad request"
// @Failure 404 {object} models.ErrorResponse "Not found"
// @Failure 500 {object} models.ErrorResponse "Internal error"
```

**无 Body 响应（如 204）：**

```go
// @Success 204 "No Content"
```

### 组合类型（Composite Types）

当响应或请求体需要使用通用包装类型（如分页），并指定其中某个字段的具体类型时，使用 `{field=Type}` 语法：

```
包名.模型名{字段名=具体类型}
包名.模型名{字段1=类型1,字段2=类型2}
```

**基础模型名**和**字段值中的类型名**均须遵循`包名(别名).[函数名.]类型名`规则；原始类型（`int64` 等）直接写。

**典型场景 — 分页包装：**

假设 `common` 包中有通用分页结构：

```go
// common/page.go
type PageData struct {
    Total int         `json:"total"`
    Page  int         `json:"page"`
    Data  interface{} `json:"data"`
}
```

在注解中指定 `data` 字段的实际类型：

```go
// @Success 200 {object} common.PageData{data=[]models.User} "Paginated users"
// @Success 200 {object} common.PageData{data=[]models.Order} "Paginated orders"
```

**多字段覆盖：**

```go
// @Success 200 {object} common.PageData{data=[]models.User,total=int64} "Paginated users"
```

**嵌套组合：**

```go
// @Success 200 {object} common.Response{data=common.PageData{items=[]models.User},meta=common.Meta} "Wrapped response"
```

**请求体同样支持：**

```go
// @Param body body models.BatchRequest{items=[]models.CreateUserRequest} true "Batch create"
```

> Builder 阶段会将组合类型展开为内联 Schema：基于基础模型的 `$ref`，用 `allOf` 覆盖指定字段的类型。

### 响应头（Response Headers）

```
// @Header 状态码 {类型} 头名称 "描述"
```

Header 会按状态码自动关联到对应的 Response。

```go
// @Success 200 {object} UserResponse "OK"
// @Header 200 {string} X-Request-Id "Request tracking ID"
// @Header 200 {integer} X-RateLimit-Remaining "Remaining requests"
```

### 安全要求（Security）

```
// @Security 方案名
// @Security 方案名[作用域1, 作用域2]
```

可多次声明，表示多个安全要求（OR 关系）。

```go
// @Security ApiKeyAuth
// @Security OAuth2[read, write]
```

---

## 四、Struct Tag 属性（Schema 约束）

在 Go struct 的字段 tag 中声明 OpenAPI Schema 属性。Parser 自动提取这些 tag，Builder 阶段映射为 Schema 字段。

### 基础属性

| Tag | OpenAPI Schema | 说明 | 示例 |
|-----|---------------|------|------|
| `json:"name"` | 属性名 | JSON 序列化名称 | `json:"user_name"` |
| `json:"name,omitempty"` | — | 省略零值 | `json:"remark,omitempty"` |
| `json:"-"` | — | 排除该字段 | `json:"-"` |
| `binding:"required"` | `required` | 标记必填（gin） | `binding:"required"` |
| `validate:"required"` | `required` | 标记必填（validator） | `validate:"required"` |
| `example:"value"` | `example` | 示例值 | `example:"brayden"` |
| `enums:"a,b,c"` | `enum` | 枚举值列表 | `enums:"active,inactive"` |
| `format:"fmt"` | `format` | 数据格式 | `format:"date-time"` |
| `default:"value"` | `default` | 默认值 | `default:"active"` |
| `description:"text"` | `description` | 字段描述 | `description:"用户状态"` |

### 访问控制

| Tag | OpenAPI Schema | 说明 | 示例 |
|-----|---------------|------|------|
| `readonly:"true"` | `readOnly` | 只读（仅在响应中出现） | `readonly:"true"` |
| `writeonly:"true"` | `writeOnly` | 只写（仅在请求中出现，如密码） | `writeonly:"true"` |
| `deprecated:"true"` | `deprecated` | 字段已废弃 | `deprecated:"true"` |

### 数值约束

| Tag | OpenAPI Schema | 说明 | 示例 |
|-----|---------------|------|------|
| `minimum:"n"` | `minimum` | 最小值（含） | `minimum:"0"` |
| `maximum:"n"` | `maximum` | 最大值（含） | `maximum:"100"` |

支持整数和浮点数，包括负数（如 `minimum:"-1.5"`）。

### 字符串约束

| Tag | OpenAPI Schema | 说明 | 示例 |
|-----|---------------|------|------|
| `minLength:"n"` | `minLength` | 最小长度 | `minLength:"1"` |
| `maxLength:"n"` | `maxLength` | 最大长度 | `maxLength:"255"` |
| `pattern:"regex"` | `pattern` | 正则表达式校验 | `pattern:"^[a-z]+$"` |

### 数组约束

| Tag | OpenAPI Schema | 说明 | 示例 |
|-----|---------------|------|------|
| `minItems:"n"` | `minItems` | 数组最少元素 | `minItems:"1"` |
| `maxItems:"n"` | `maxItems` | 数组最多元素 | `maxItems:"50"` |
| `uniqueItems:"true"` | `uniqueItems` | 数组元素须唯一 | `uniqueItems:"true"` |

### 常用 Format 值

| `format` | 适用类型 | 说明 |
|----------|---------|------|
| `int32` | integer | 32 位整数 |
| `int64` | integer | 64 位整数 |
| `float` | number | 单精度浮点 |
| `double` | number | 双精度浮点 |
| `date` | string | 日期 `2006-01-02` |
| `date-time` | string | 日期时间 RFC 3339 |
| `email` | string | 邮箱地址 |
| `uri` | string | URI |
| `uuid` | string | UUID |
| `byte` | string | Base64 编码 |
| `binary` | string | 二进制数据 |
| `password` | string | 密码（UI 中隐藏） |

### 综合示例

```go
type Product struct {
    Name       string   `json:"name" description:"Product name" minLength:"1" maxLength:"200" example:"Widget"`
    Price      float64  `json:"price" minimum:"0" maximum:"99999.99" default:"0" format:"double"`
    SKU        string   `json:"sku" pattern:"^[A-Z]{2}-\\d{6}$" example:"AB-123456"`
    Quantity   int      `json:"quantity" minimum:"0" maximum:"10000" default:"1"`
    Tags       []string `json:"tags" minItems:"1" maxItems:"10" uniqueItems:"true"`
    InternalID string   `json:"internal_id" readonly:"true" format:"uuid"`
    Password   string   `json:"password" writeonly:"true" minLength:"8"`
    OldField   string   `json:"old_field" deprecated:"true"`
    CreatedAt  string   `json:"created_at" format:"date-time" readonly:"true"`
    Email      string   `json:"email" format:"email" description:"Contact email"`
    State      string   `json:"state" enums:"active,inactive" default:"active" example:"active"`
}
```

> **指针类型**：`*string`、`*int` 等指针类型在 Builder 阶段自动映射为 `nullable: true`（OpenAPI 3.0）。  
> **Required 判定**：字段是否 required 由 `binding:"required"` 或 `validate:"required"` 决定，而非指针/非指针类型。

---

## 五、完整示例

```go
package main

// @title Pet Store API
// @version 2.0.0
// @description A sample pet store server.
// @termsOfService https://example.com/terms
//
// @contact.name API Support
// @contact.email support@example.com
// @contact.url https://www.example.com/support
//
// @license.name Apache 2.0
// @license.url https://www.apache.org/licenses/LICENSE-2.0.html
//
// @server https://api.petstore.com "Production"
// @server http://localhost:8080 "Development"
//
// @externalDocs.url https://example.com/docs
// @externalDocs.description Find out more
//
// @tag pets "Everything about your Pets"
// @tag users "User operations"
//
// @securityDefinitions.apikey ApiKeyAuth
// @securityDefinitions.apikey.in header
// @securityDefinitions.apikey.name X-API-Key
//
// @securityDefinitions.oauth2.authorizationCode OAuth2
// @securityDefinitions.oauth2.authorizationCode.authorizationUrl https://example.com/oauth/authorize
// @securityDefinitions.oauth2.authorizationCode.tokenUrl https://example.com/oauth/token
// @securityDefinitions.oauth2.authorizationCode.scope.read "Read access"
// @securityDefinitions.oauth2.authorizationCode.scope.write "Write access"
func main() {}

// ListPets returns a list of all pets in the store.
//
// @Summary List all pets
// @Description Get a paginated list of all pets
// @ID listPets
// @Tags pets
// @Accept json
// @Produce json, xml
// @Param limit  query integer false int32 "Max items per page"
// @Param offset query integer false int32 "Offset"
// @Success 200 {object} common.PageData{data=[]models.Pet} "Paginated pets"
// @Header 200 {string} X-Total-Count "Total number of pets"
// @Failure 500 {object} models.Error "Server error"
// @Security ApiKeyAuth
// @Router /pets [get]
func ListPets() {}

// CreatePet creates a new pet.
//
// @Summary Create a pet
// @Tags pets
// @Accept json
// @Produce json
// @Param body body models.CreatePetRequest true "Pet to create"
// @Success 201 {object} models.Pet "Created"
// @Failure 400 {object} models.Error "Validation error"
// @Security OAuth2[write]
// @Router /pets [post]
func CreatePet() {}

// DeletePet deletes a pet by ID.
//
// @Summary Delete a pet
// @Tags pets
// @Param id path integer true int64 "Pet ID"
// @Success 204 "No Content"
// @Failure 404 {object} models.Error "Not found"
// @Security ApiKeyAuth
// @Deprecated
// @Router /pets/{id} [delete]
func DeletePet() {}

// UploadPhoto uploads a pet photo.
//
// @Summary Upload pet photo
// @Tags pets
// @Accept mpfd
// @Produce json
// @Param id      path     integer         true  "Pet ID"
// @Param file    formData file            true  "Photo file"
// @Param caption formData string          false "Photo caption"
// @Success 200 {object} models.UploadResult "OK"
// @Router /pets/{id}/photos [post]
func UploadPhoto() {}
```

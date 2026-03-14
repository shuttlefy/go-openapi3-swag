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

写在全局注解同一函数中。采用多行声明模式。

### API Key

```go
// @securityDefinitions.apikey ApiKeyAuth
// @securityDefinitions.apikey.in header
// @securityDefinitions.apikey.name X-API-Key
// @securityDefinitions.apikey.description API key authentication
```

| 子属性 | 说明 | 可选值 |
|--------|------|--------|
| `.in` | 传递位置 | `header`, `query`, `cookie` |
| `.name` | 参数名称 | 任意字符串 |
| `.description` | 描述 | 任意字符串 |

### HTTP Basic

```go
// @securityDefinitions.basic BasicAuth
```

自动设置 `type=http`, `scheme=basic`。

### HTTP Bearer

```go
// @securityDefinitions.bearer BearerAuth
// @securityDefinitions.bearer.bearerFormat JWT
```

| 子属性 | 说明 |
|--------|------|
| `.bearerFormat` | Token 格式，如 `JWT` |
| `.description` | 描述 |

### OAuth2

OAuth2 需要指定流类型。支持的流类型：

| 流类型 | 说明 |
|--------|------|
| `implicit` | 隐式授权 |
| `password` | 密码授权 |
| `clientCredentials` | 客户端凭证 |
| `authorizationCode` | 授权码 |

```go
// @securityDefinitions.oauth2.authorizationCode OAuth2
// @securityDefinitions.oauth2.authorizationCode.authorizationUrl https://example.com/oauth/authorize
// @securityDefinitions.oauth2.authorizationCode.tokenUrl https://example.com/oauth/token
// @securityDefinitions.oauth2.authorizationCode.scope.read "Read access"
// @securityDefinitions.oauth2.authorizationCode.scope.write "Write access"
```

| 子属性 | 说明 |
|--------|------|
| `.authorizationUrl` | 授权端点 URL |
| `.tokenUrl` | Token 端点 URL |
| `.scope.{名称}` | 作用域定义，值为描述 |
| `.description` | 描述 |

### OpenID Connect

```go
// @securityDefinitions.openIdConnect MyOIDC
// @securityDefinitions.openIdConnect.openIdConnectUrl https://example.com/.well-known/openid-configuration
```

---

## 三、操作注解（Operation Annotations）

写在 API 处理函数的注释中。**必须包含 `@Router`**，否则该函数被忽略。

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

| 示例 |
|------|
| `// @Router /users/{id} [get]` |
| `// @Router /users [post]` |
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
| 名称 | 参数名 | 任意字符串 |
| 位置 | 参数位置 | `path`, `query`, `header`, `cookie`, `body`, `formData` |
| 类型 | 数据类型 | `string`, `integer`, `int`, `number`, `boolean`, `file`, 或模型名 |
| 必填 | 是否必填 | `true`, `false` |
| 格式 | 类型格式（可选） | `int32`, `int64`, `float`, `double`, `date`, `date-time`, `email`, `uri` 等 |
| 描述 | 参数描述（引号包裹） | `"User ID"` |

**示例：**

```go
// @Param id path int true "User ID"
// @Param status query string false "Filter status"
// @Param X-Token header string true "Auth token"
// @Param limit query integer false int32 "Page size"
// @Param created_at query string false date-time "Creation date"
```

### 请求体（Request Body）

在 OpenAPI 3 中，`body` 和 `formData` 参数被转换为 `requestBody`。

**JSON Body：**

```go
// @Param body body CreateUserRequest true "User to create"
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

| 类型包装器 | 说明 |
|------------|------|
| `{object}` | 对象类型，引用模型 Schema |
| `{array}` | 数组类型，items 引用模型 Schema |
| `{string}` | 原始 string 类型 |
| `{integer}` | 原始 integer 类型 |
| `{number}` | 原始 number 类型 |
| `{boolean}` | 原始 boolean 类型 |

**示例：**

```go
// @Success 200 {object} UserResponse "OK"
// @Success 200 {array} UserResponse "Users list"
// @Success 200 {string} string "A raw string"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 404 {object} ErrorResponse "Not found"
// @Failure 500 {object} ErrorResponse "Internal error"
```

**无 Body 响应（如 204）：**

```go
// @Success 204 "No Content"
```

### 组合类型（Composite Types）

当响应或请求体需要使用通用包装类型（如分页），并指定其中某个字段的具体类型时，使用 `{field=Type}` 语法：

```
模型名{字段名=具体类型}
模型名{字段1=类型1,字段2=类型2}
```

**典型场景 — 分页包装：**

假设项目中有通用分页结构：

```go
type PageData struct {
    Total int         `json:"total"`
    Page  int         `json:"page"`
    Data  interface{} `json:"data"`
}
```

在注解中指定 `data` 字段的实际类型：

```go
// @Success 200 {object} PageData{data=[]User} "Paginated users"
// @Success 200 {object} PageData{data=[]Order} "Paginated orders"
```

**多字段覆盖：**

```go
// @Success 200 {object} PageData{data=[]User,total=int64} "Paginated users"
```

**嵌套组合：**

```go
// @Success 200 {object} Response{data=PageData{items=[]User},meta=Meta} "Wrapped response"
```

**请求体同样支持：**

```go
// @Param body body BatchRequest{items=[]CreateUserRequest} true "Batch create"
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
// @Param limit query integer false int32 "Max items per page"
// @Param offset query integer false int32 "Offset"
// @Success 200 {object} PageData{data=[]Pet} "Paginated pets"
// @Header 200 {string} X-Total-Count "Total number of pets"
// @Failure 500 {object} Error "Server error"
// @Security ApiKeyAuth
// @Router /pets [get]
func ListPets() {}

// CreatePet creates a new pet.
//
// @Summary Create a pet
// @Tags pets
// @Accept json
// @Produce json
// @Param body body CreatePetRequest true "Pet to create"
// @Success 201 {object} Pet "Created"
// @Failure 400 {object} Error "Validation error"
// @Security OAuth2[write]
// @Router /pets [post]
func CreatePet() {}

// DeletePet deletes a pet by ID.
//
// @Summary Delete a pet
// @Tags pets
// @Param id path integer true int64 "Pet ID"
// @Success 204 "No Content"
// @Failure 404 {object} Error "Not found"
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
// @Param id path integer true "Pet ID"
// @Param file formData file true "Photo file"
// @Param caption formData string false "Photo caption"
// @Success 200 {object} UploadResult "OK"
// @Router /pets/{id}/photos [post]
func UploadPhoto() {}
```

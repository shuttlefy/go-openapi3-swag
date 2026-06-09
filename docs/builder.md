# builder 包（Stage 3）

**代码位置**：`internal/builder/`

| 文件 | 职责 |
|------|------|
| `builder.go` | `Builder`：编排各子组件，组装 `*spec3.OpenAPI` |
| `resolver.go` | `Resolver`：类型查找、`$ref` 管理、循环检测、懒加载 |
| `schema_builder.go` | `SchemaBuilder`：Go struct/类型 → `spec3.Schema` |
| `operation_builder.go` | `OperationBuilder`：注解 → `spec3.Operation` |

## 职责

将 `*extractor.ExtractResult`（注解）+ `[]*parser.RawFile`（类型信息）直接构建为 `*spec3.OpenAPI`，无自定义 IR。

---

## Builder

```go
func NewBuilder() *Builder
func (b *Builder) SetLoader(loader PackageLoader)
func (b *Builder) SetQueryStructExplode(v bool)
func (b *Builder) Build(result *extractor.ExtractResult, files []*parser.RawFile) (*spec3.OpenAPI, error)
```

`Builder` 持有 `SchemaBuilder`、`OperationBuilder`、`Resolver` 的引用，负责：

1. 用 `extractor.GlobalAnnotation` 填充 `spec3.Info`、`spec3.Server`、`spec3.Tag`、`spec3.SecurityScheme`
2. 遍历 `Operations`，调用 `OperationBuilder.Build()` 得到每条路由的 `spec3.Operation`
3. 将 `SchemaBuilder.Components()` 注入 `doc.Components`

---

## Resolver

类型解析的核心，负责将字符串形式的类型名（如 `"models.User"`）解析为 `*spec3.Schema` 或 `$ref`。

### SchemaKey

```go
type SchemaKey string

func NewSchemaKey(pkg, typeName string) SchemaKey        // "pkg.TypeName"
func GenericSchemaKey(base SchemaKey, args ...SchemaKey) SchemaKey // "pkg.T[A,B]"
func CompositeSchemaKey(base SchemaKey, overrides map[string]string) SchemaKey // "pkg.T{field=Type,...}"
func (k SchemaKey) Ref() string                          // "#/components/schemas/<key>"
```

### PackageLoader（第三方包懒加载）

```go
// pkgName：短包名（如 "gin"）；srcFile：引用该包的源文件（用于查 import path）
type PackageLoader func(pkgName string, srcFile *parser.RawFile) []*parser.RawFile

func NewModuleLoader(modInfo *parser.ModuleInfo, cacheDir string) PackageLoader
```

当 `Resolver` 在已扫描文件中找不到某类型时，自动触发懒加载：定位模块缓存目录 → 调用 `parser.ParseDir()` → 追加到 `r.files` → 重试查找。每个 import path 只加载一次（`loadedPkgs` 去重）。

### 类型查找算法（`Resolve`）

**入口**：`Resolve(typeStr string, currentFile *parser.RawFile) *spec3.Schema`

#### Step 1：解析前缀和结构

| 前缀/结构 | 处理 |
|---------|------|
| `[]` | `isArray=true`，递归解析元素类型 |
| `*` | `nullable=true`，递归解析元素类型 |
| 含 `{…}` | 组合类型（见 Step 6） |
| 含 `[…]` | 泛型实例化（见 Step 5） |
| 其余 | 简单类型，按 `.` 分段 |

#### Step 2：qualifier → pkgName

| 优先级 | 条件 | 结果 |
|--------|------|------|
| 1 | `qualifier == currentFile.Package` | 当前包 |
| 2 | `Imports` 中 `Alias == qualifier` | 别名精确匹配，取 `PkgName` |
| 3 | `Imports` 中 `PkgName == qualifier` | 包本名兼容匹配 |
| — | 均未匹配 | 返回 nil |

#### Step 3：原始类型短路

| Go 类型 | OpenAPI schema |
|---------|---------------|
| `string` | `{type: string}` |
| `int`, `int32` ... | `{type: integer, format: ...}` |
| `float32`, `float64` | `{type: number, format: ...}` |
| `bool` | `{type: boolean}` |
| `interface{}`, `any`, `error` | `{}` |

#### Step 4：已知外部类型短路

| 类型 | OpenAPI schema |
|------|---------------|
| `time.Time` | `{type: string, format: date-time}` |
| `time.Duration` | `{type: integer, format: int64}` |
| `uuid.UUID` | `{type: string, format: uuid}` |
| `decimal.Decimal` | `{type: string, format: decimal}` |
| `json.RawMessage` | `{type: object}` |
| `net.IP` | `{type: string, format: ipv4}` |

#### Step 5：在 RawFile 中查找（按优先级）

```
for each file where file.Package == pkgName:
    1. 函数局部 struct（仅 funcName 非空时）→ LocalStructs
    2. 包级 Struct                         → 构建 schema，注册到 Components
    3. 类型别名 type A = B                 → 穿透，递归 Resolve(B)
    4. 常量 enum（const 块同类型常量）      → 推断基础类型 + enum 数组
    5. 非 struct 类型定义 type H map[...]  → 穿透底层类型，不注册自身
```

未命中时触发懒加载并重试（见 PackageLoader）。

#### Step 6：泛型实例化展开

找到泛型 struct 定义后，将 `typeArgs` 代入字段的类型参数：

```
对每个 RawField：
    若 field.TypeName 是类型参数名 → 替换为具体 typeArg，递归 Resolve
    否则                           → 直接 Resolve(field.TypeName, structFile)
```

以 `GenericSchemaKey(baseKey, argKeys...)` 为 key 注册，相同 key 直接复用（去重）。

#### Step 7：组合类型展开（`Base{field=Type,...}`）

```
1. 递归 Resolve(baseType)              → baseSchema（$ref）
2. 对每个 fieldOverride：递归 Resolve(TypeExpr) → overrideSchema
3. 生成内联 allOf schema（不注册到 Components）：
   allOf:
     - $ref: baseKey
     - type: object
       properties:
         field: overrideSchema
```

组合类型**不注册** `Components.Schemas`，每处使用点持有独立内联 schema。

---

## SchemaBuilder

```go
func NewSchemaBuilder(resolver *Resolver) *SchemaBuilder
func (sb *SchemaBuilder) Build(typeStr string, file *parser.RawFile) *spec3.Schema
func (sb *SchemaBuilder) Components() *spec3.Components
```

### struct → Schema

1. 遍历 `RawStruct.Fields`，按位置（kind）选择命名 tag，遇到 `tag:"-"` 跳过字段
2. 对嵌入字段（`Embedded=true`）递归展开，合并属性
3. 对每个字段解析 struct tag（见下），再调用 `Resolver.Resolve` 得到字段 schema
4. 收集 `binding:"required"` / `validate:"required"` 字段名到 `schema.Required`

### struct tag 解析（`parseStructTags`）

`parseStructTags(rawTag, fieldName, kind)` 接受参数位置（`kind`），按 OpenAPI 位置选择**命名 tag**优先级，与 gin 等框架的绑定行为一致：

| 参数位置 (`in`) | 命名 tag 优先级 |
|-----------------|----------------|
| `body`          | `json` |
| `query`         | `form` → `json` |
| `formData`      | `form` → `json` |
| `path`          | `uri` → `json` |
| `header`        | `header` → `json` |
| `cookie`        | `json` |

规则：
- **首个出现的 tag** 决定字段名与 `-` 跳过语义。例如 query 位置上 `form:"-"` 直接跳过该字段，不再回退到 `json`。
- 若优先级第一的 tag 完全缺失，回退到下一个（`json` 兜底，兼容旧代码仅写 `json` tag 的 struct）。
- `body` 位置只看 `json`，不会被 `form` tag 干扰（确保同一个 struct 既可作为 query 也可作为 body 时不串名）。

约束 tag（与命名无关，所有位置一致）：

| Tag | 对应 OpenAPI |
|-----|-------------|
| `binding/validate:"required"` | `required` |
| `description:"..."` | `description` |
| `example:"..."` | `example` |
| `enums:"a,b,c"` | `enum` |
| `format:"..."` | `format` |
| `default:"..."` | `default` |
| `readonly:"true"` | `readOnly` |
| `writeonly:"true"` | `writeOnly` |
| `deprecated:"true"` | `deprecated` |
| `minimum/maximum:"n"` | `minimum` / `maximum` |
| `minLength/maxLength:"n"` | `minLength` / `maxLength` |
| `pattern:"regex"` | `pattern` |
| `minItems/maxItems:"n"` | `minItems` / `maxItems` |
| `uniqueItems:"true"` | `uniqueItems` |

### 常量 enum schema（`buildEnumSchema`）

`const` 块内同类型常量构成 enum：

```go
func buildEnumSchema(consts []parser.RawConst) *spec3.Schema
```

- 首个值为纯数字 → `type: integer`，否则 → `type: string`
- 始终生成 `x-enum-varnames`（常量名列表）
- 至少一个常量有注释时，生成 `x-enumdescriptions`（每个常量的注释，无注释的填 `""`）

---

## OperationBuilder

```go
func NewOperationBuilder(schema *SchemaBuilder) *OperationBuilder
func (ob *OperationBuilder) Build(op extractor.OperationAnnotation, fileIndex map[string]*parser.RawFile) (*spec3.Operation, error)
```

处理逻辑：

1. **`@Param`（非 body/formData）** → `spec3.Parameter`，类型调用 `SchemaBuilder.Build`
2. **Struct 打散（显式）**（`name=""`）→ 展开 struct 所有字段为独立 `Parameter`
3. **Struct 打散（自动）** → 当 `queryStructExplode=true` 且 `in=query` 时，若类型解析为 struct，自动展开字段；类型不是 struct 时退回为普通 `Parameter`
3. **`@Param body`** → `spec3.RequestBody`，content-type 由 `@Accept` 决定（默认 `application/json`）
4. **`@Param formData`** → `spec3.RequestBody`，content-type 为 `multipart/form-data`
5. **`@Success` / `@Failure`** → `spec3.Response`，schema 由 `SchemaBuilder.Build` 解析
6. **`@Header`** → 关联到对应状态码的 `spec3.Response.Headers`
7. **`@Security`** → `spec3.SecurityRequirement`

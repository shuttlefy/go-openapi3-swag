# Swag3 架构设计

> 源代码 → AST 解析 → 注释提取 → spec3 构建 → JSON/YAML 输出

**核心依赖**: [`github.com/shuttlefy/go-openapi3-spec`](https://github.com/shuttlefy/go-openapi3-spec) — 提供完整的 OpenAPI 3.x Go 结构体，包含有序 Map、`$ref` 支持、vendor extension、easyjson 加速序列化、YAML 序列化。

**设计原则**: 务实分层 + 选择性依赖倒置。Builder 直接构建 `spec3.OpenAPI`，无自定义 IR。在真正需要可替换性和可测试性的边界使用接口，不为只有一个实现的组件强制抽象。

---

## 1. 目录结构

```
swag3/
├── cmd/
│   └── swag3/
│       └── main.go                # CLI 入口 + 组装
├── internal/
│   ├── parser/
│   │   ├── parser.go              # GoParser（AST 解析）
│   │   ├── model.go               # RawAST / RawFunc / RawStruct / RawTypeDef
│   │   └── module.go              # go.mod 解析 + 模块缓存路径解析
│   ├── extractor/
│   │   ├── extractor.go           # GoExtractor（注释提取）
│   │   ├── annotation.go          # ExtractResult / OperationAnnotation
│   │   └── tag_parser.go          # @Param, @Success 等标签解析
│   ├── builder/
│   │   ├── builder.go             # Builder：注解 → spec3.OpenAPI
│   │   ├── schema_builder.go      # Go type → spec3.Schema
│   │   ├── operation_builder.go   # 注解 → spec3.Operation
│   │   └── resolver.go            # $ref 引用管理 + 循环检测
│   └── output/
│       └── writer.go              # spec3.OpenAPI → JSON / YAML 文件
├── config/
│   └── config.go                  # 全局配置
├── testdata/
│   ├── go/
│   └── expected/
├── go.mod
└── README.md
```

每个 `internal/` 子包职责清晰、文件数适中（2-4 个），无过度嵌套。

---

## 2. 模块划分

### 2.1 `parser` — AST 解析

**职责**：将 Go 源文件递归解析成 AST 节点列表树。

| 组件 | 职责 |
|------|------|
| `GoParser` | 使用 `go/ast` + `go/types` 递归解析 Go 源码 |
| `model.go` | `RawFile`、`RawFunc`、`RawStruct`、`RawTypeAlias`、`RawConst`、`RawTypeDef` 等数据结构 |
| `module.go` | `ParseGoMod`、`ModuleCacheDir`、`ResolvePackageDir` —— 模块缓存路径解析 |

**解析范围**：

| 构造 | 说明 |
|------|------|
| 函数 / 方法 | 含注释、接收者、参数、返回值 |
| Struct | 含字段、tag、泛型类型参数 |
| Import 别名 | **文件级**：`import m "pkg/path"` → 仅在本文件有效 |
| 类型别名 | `type Foo = Bar` |
| 类型定义 | `type H map[string]any` / `type Params []Param` 等非 struct 新类型 → `RawTypeDef` |
| 泛型类型参数 | `type Resp[T any] struct` 中的 `TypeParams` |
| 常量 enum | `const` 块内同类型的常量组，作为候选 enum 值收集 |

```go
type GoParser struct {
    MaxDepth int // 递归扫描最大深度，来自 Config.ParseDepth
}

func (p *GoParser) Parse(dirs []string) ([]*RawFile, error)
func (p *GoParser) ParseDir(dir string) ([]*RawFile, error) // 仅解析单层目录（用于模块缓存懒加载）
```

**输入**：源文件目录列表（每个目录按 `MaxDepth` 递归展开子目录）
**输出**：`[]*RawFile`（每个文件对应一个节点，保留目录层级关系）

**递归规则**：
- `depth=0`：仅扫描 `dirs` 本身，不进入子目录
- `depth=N`：最多向下 N 层子目录
- `depth=-1`：无限递归（扫描全部子目录）

---

### 2.2 `extractor` — 注释提取

**职责**：从 `RawAST` 中提取结构化 API 注解。

| 组件 | 职责 |
|------|------|
| `GoExtractor` | 解析 `// @Tag value` 风格注释 |
| `TagParser` | 解析每个注解标签的结构化属性 |
| `annotation.go` | `ExtractResult`、`OperationAnnotation` 数据结构 |

```go
type GoExtractor struct{}

func (e *GoExtractor) Extract(files []*parser.RawFile) (*ExtractResult, error)
```

**输入**：`[]*parser.RawFile`
**输出**：`*ExtractResult`

---

### 2.3 `builder` — spec3 构建（核心逻辑）

**职责**：将注解 + 类型信息直接构建为 `spec3.OpenAPI`。

| 组件 | 职责 |
|------|------|
| `Builder` | 编排 SchemaBuilder + OperationBuilder，组装 `spec3.OpenAPI`；持有 `PackageLoader` |
| `SchemaBuilder` | Go struct / 类型别名 / 泛型实例化 → `spec3.Schema`，注册到 `Components.Schemas` |
| `OperationBuilder` | 注解 → `spec3.Operation` |
| `Resolver` | `SchemaKey` 管理 + `$ref` 生成 + 循环检测 + 包别名解析 + 类型别名穿透 + 第三方包懒加载 |

**输入**：`*extractor.ExtractResult` + `[]*parser.RawFile`
**输出**：`*spec3.OpenAPI`

**直接使用的 spec3 类型**：

| spec3 类型 | 用途 |
|-----------|------|
| `spec3.OpenAPI` | 根文档 |
| `spec3.Info` | API 元信息 |
| `spec3.Server` | 服务器地址 |
| `spec3.Paths` / `spec3.PathItem` | 路径 + 操作集 |
| `spec3.Operation` | 单个 API 操作 |
| `spec3.Parameter` | 请求参数 |
| `spec3.RequestBody` | 请求体 |
| `spec3.Response` / `spec3.OrderedResponses` | 响应 |
| `spec3.Schema` / `spec3.OrderedSchemas` | JSON Schema |
| `spec3.Components` | 可复用组件 |
| `spec3.SecurityScheme` / `spec3.SecurityRequirement` | 安全方案 |
| `spec3.Tag` | 标签分组 |
| `spec3.MediaType` / `spec3.OrderedMediaTypes` | 媒体类型 |

---

### 2.4 `output` — 序列化输出

**职责**：将 `*spec3.OpenAPI` 序列化为 JSON 或 YAML 文件。

```go
func Write(doc *spec3.OpenAPI, format string, path string) error
```

由 `spec3.MarshalJSON` / `spec3.MarshalYAML` 完成实际序列化，此层仅处理格式选择和文件写入。

---

## 3. 数据流

```
源文件 (*.go)
     │
     ▼
┌──────────────────────┐
│ parser.GoParser      │  go/ast 递归解析（深度由 ParseDepth 控制）
│ Parse(dirs)          │
└──────────┬───────────┘
           │ []*parser.RawFile
           ▼
┌──────────────────────┐
│ extractor.GoExtractor│  注释 → 结构化注解
│ Extract(ast)         │
└──────────┬───────────┘
           │ *extractor.ExtractResult
           ▼
┌──────────────────────────────────────────┐
│         builder.Builder                  │
│                                          │
│  ┌────────────────┐  ┌────────────────┐  │
│  │ SchemaBuilder  │  │ OperationBuilder│  │
│  │ Go struct →    │  │ Annotation →   │  │
│  │ spec3.Schema   │  │ spec3.Operation│  │
│  └───────┬────────┘  └───────┬────────┘  │
│          └──── Resolver ─────┘           │
│               ($ref 管理)                │
│                                          │
│  输出: *spec3.OpenAPI                     │
└──────────────────┬───────────────────────┘
                   │
                   ▼
┌──────────────────────┐
│ output.Write         │  spec3.MarshalJSON / spec3.MarshalYAML
└──────────┬───────────┘
           │
           ▼
  openapi.json / openapi.yaml
```

---

## 4. 核心数据结构

### 4.1 `parser/model.go` — 解析产物

解析产物为 `[]*RawFile` 列表，每个 `RawFile` 对应一个 `.go` 源文件，携带该文件内的所有节点。包别名（import alias）**仅存储在其所属的 `RawFile` 中**，不跨文件共享。

```go
package parser

// RawFile 对应一个 .go 源文件的解析结果
type RawFile struct {
    Package     string
    FilePath    string
    Imports     []RawImport    // import 别名，仅本文件有效
    Functions   []RawFunc
    Structs     []RawStruct
    TypeAliases []RawTypeAlias // type Foo = Bar（透明别名）
    TypeDefs    []RawTypeDef   // type Foo Bar（非 struct、非透明别名的新类型定义）
    Consts      []RawConst     // 常量（含 enum 候选）
}

// RawImport 记录一条 import 声明
// 包别名作用域为文件级，resolver 解析类型引用时必须结合当前文件的 Imports
type RawImport struct {
    Alias   string // 显式别名；"" 表示使用包名；"." 表示 dot-import
    Path    string // import path，如 "github.com/example/models"
    PkgName string // 包的实际名称（由 go/types 解析填充）
}

// RawTypeAlias 对应 type Foo = Bar（别名声明，非新类型）
type RawTypeAlias struct {
    Name     string
    TypeName string // 右侧类型，可能含包限定符，如 "time.Time"
    FilePath string
    Comments []string
}

// RawTypeDef 表示非 struct 的新类型定义，例如 type H map[string]any 或 type Params []Param。
type RawTypeDef struct {
    Name     string
    TypeName string   // 底层类型，如 "map[string]interface{}" / "[]Param"
    FilePath string
    Comments []string
}

// RawConst 对应一个具名常量，常用于 string/int enum
type RawConst struct {
    Name     string
    TypeName string // 显式类型名，如 "Status"；无类型常量为 ""
    Value    string // 字面量值，如 `"active"` / `1`
    FilePath string
    Comments []string
}

type RawFunc struct {
    Name         string
    FilePath     string
    Line         int
    Comments     []string
    Receiver     string
    Params       []RawParam
    Results      []RawParam
    LocalStructs []RawStruct // 函数体内定义的局部 struct，作用域仅限本函数
}

// RawStruct 支持泛型：TypeParams 非空表示该 struct 有类型参数
type RawStruct struct {
    Name       string
    FilePath   string
    Fields     []RawField
    Comments   []string
    TypeParams []RawTypeParam // 泛型参数，如 [T any, U comparable]
}

// RawTypeParam 描述一个泛型类型参数
type RawTypeParam struct {
    Name       string // 参数名，如 "T"
    Constraint string // 约束，如 "any" / "comparable" / "int|string"
}

type RawField struct {
    Name     string
    TypeName string // 可能是类型参数名（如 "T"）或具体类型（如 "m.User"）
    Tag      string
    Comments []string
}

type RawParam struct {
    Name     string
    TypeName string
}
```

> `extractor` 和 `builder` 接收 `[]*RawFile` 并扁平化遍历；解析类型引用时，`Resolver` 须携带当前文件的 `Imports` 上下文。

### 4.2 `extractor/annotation.go` — 注解模型

```go
package extractor

type ExtractResult struct {
    Global     GlobalAnnotation
    Operations []OperationAnnotation
}

type GlobalAnnotation struct {
    Title        string
    Description  string
    Version      string
    Host         string
    BasePath     string
    Schemes      []string
    Tags         []TagAnnotation
    SecurityDefs []SecurityDefAnnotation
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
    Params      []ParamAnnotation
    Responses   []ResponseAnnotation
    Security    []string
    Deprecated  bool
}

type RouteInfo struct {
    Method string
    Path   string
}

type ParamAnnotation struct {
    Name        string
    In          string
    TypeName    string
    Required    bool
    Description string
    Format      string
}

type ResponseAnnotation struct {
    Code        string
    TypeName    string
    Description string
    IsArray     bool
}
```

### 4.3 `builder/builder.go` — 核心构建逻辑

```go
package builder

import (
    spec3 "github.com/shuttlefy/go-openapi3-spec"
    "github.com/shuttlefy/go-openapi3-swag/internal/extractor"
    "github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

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
func (b *Builder) SetLoader(loader PackageLoader) {
    b.resolver.SetLoader(loader)
}

// NewModuleLoader 创建基于 go.mod 模块缓存的 PackageLoader。
// 需先调用 parser.ParseGoMod 和 parser.ModuleCacheDir 获取入参。
func NewModuleLoader(modInfo *parser.ModuleInfo, cacheDir string) PackageLoader

func (b *Builder) Build(result *extractor.ExtractResult, files []*parser.RawFile) (*spec3.OpenAPI, error) {
    doc := &spec3.OpenAPI{
        OpenAPI: "3.0.3",
        Info: &spec3.Info{
            Title:       result.Global.Title,
            Version:     result.Global.Version,
            Description: result.Global.Description,
        },
    }

    if result.Global.Host != "" {
        doc.Servers = []spec3.Server{
            {URL: buildServerURL(result.Global)},
        }
    }

    paths := spec3.NewPaths()
    for _, opAnno := range result.Operations {
        operation, err := b.op.Build(opAnno)
        if err != nil {
            return nil, err
        }
        setOperation(&paths, opAnno.Route.Path, opAnno.Route.Method, operation)
    }
    doc.Paths = &paths
    doc.Components = b.schema.Components()

    return doc, nil
}
```

### 4.4 `output/writer.go` — 序列化输出

```go
package output

import (
    "encoding/json"
    "os"

    spec3 "github.com/shuttlefy/go-openapi3-spec"
)

func Write(doc *spec3.OpenAPI, format, path string) error {
    var data []byte
    var err error
    switch format {
    case "json":
        data, err = json.MarshalIndent(doc, "", "  ")
    case "yaml":
        data, err = spec3.MarshalYAML(doc)
    }
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0644)
}
```

### 4.5 `cmd/swag3/main.go` — CLI 入口

```go
package main

import (
    "log"

    "github.com/shuttlefy/go-openapi3-swag/config"
    "github.com/shuttlefy/go-openapi3-swag/internal/builder"
    "github.com/shuttlefy/go-openapi3-swag/internal/extractor"
    "github.com/shuttlefy/go-openapi3-swag/internal/output"
    "github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

func main() {
    cfg := config.ParseFlags()

    // 1. Parse
    p := &parser.GoParser{MaxDepth: cfg.ParseDepth}
    files, err := p.Parse(cfg.InputDirs)
    if err != nil {
        log.Fatalf("parse: %v", err)
    }

    // 2. Extract
    e := &extractor.GoExtractor{}
    result, err := e.Extract(files)
    if err != nil {
        log.Fatalf("extract: %v", err)
    }

    // 3. Build（始终启用模块缓存懒加载）
    b := builder.NewBuilder()
    modInfo, err := parser.ParseGoMod("go.mod")
    if err != nil {
        log.Printf("warn: parse go.mod: %v (module loader disabled)", err)
    } else {
        b.SetLoader(builder.NewModuleLoader(modInfo, parser.ModuleCacheDir()))
    }
    doc, err := b.Build(result, files)
    if err != nil {
        log.Fatalf("build: %v", err)
    }

    // 4. Write
    if err := output.Write(doc, cfg.OutputFile); err != nil {
        log.Fatalf("write: %v", err)
    }
}
```

---

## 5. 接口的使用原则

**不为只有一个实现的组件创建接口**，但在以下场景保留接口：

### 5.1 Builder 内部接口 — 服务于可测试性

`SchemaBuilder` 和 `OperationBuilder` 在 Builder 内部通过接口引用，使 Builder 的单元测试可以用 stub 替换子组件：

```go
// builder 包内部定义，不导出
type schemaResolver interface {
    Resolve(typeName string, rawAST *parser.RawAST) *spec3.Schema
    RefSchema(typeName string) *spec3.Schema
    Components() *spec3.Components
}
```

### 5.2 何时引入包级接口

当以下条件**满足任一**时，将组件提升为接口：
- 存在第二个真实实现（如未来支持 TypeScript 解析）
- 测试需要替换真实 I/O 且 stub 不够用

**当前不需要接口的组件**：
- `GoParser` — 只有 Go 一种语言
- `GoExtractor` — 注解格式固定
- `output.Write` — 纯函数，直接测试输出字节即可

---

## 6. 关键设计决策

### 6.1 务实分层而非教条 DDD

| 考量 | 决策 |
|------|------|
| 项目复杂度 | 无状态数据转换流水线，无复杂业务规则 |
| 领域模型 | 纯数据载体，无行为（天然贫血，这是合理的） |
| 限界上下文 | 只有一个上下文，无需 DDD 上下文映射 |
| 接口数量 | 按需引入，不为一个实现强制抽象 |
| 目录深度 | 扁平 `internal/` 包，每包 2-4 文件 |

### 6.2 依赖倒置 — 在边界处选择性应用

**保留**依赖倒置的好实践：
- 构建逻辑（`builder/`）不依赖文件 I/O
- 数据结构定义在产出方的包中（`parser.RawAST` 在 parser 包，`extractor.ExtractResult` 在 extractor 包）
- `spec3` 作为共享类型贯穿 builder 和 output

**不做**的事：
- 不为 Parser/Extractor/Writer 创建 Port 接口包
- 不将领域模型抽到独立的 `domain/model` 包

### 6.3 直接构建 spec3，无自定义 IR

`go-openapi3-spec` 已提供完整的 OpenAPI 3 结构体，`builder` 直接产出 `*spec3.OpenAPI`。

### 6.4 只支持 OpenAPI 3

`go-openapi3-spec` 兼容 3.0 / 3.1 / 3.2，默认输出 3.0.3。

### 6.5 output 层极度精简

`spec3.MarshalJSON` + `spec3.MarshalYAML` 处理全部序列化逻辑。

### 6.6 注解风格与 swaggo/swag 兼容

```go
// @Summary     获取用户信息
// @Router      /users/{id} [get]
// @Param       id path int true "用户 ID"
// @Success     200 {object} UserResponse
// @Failure     404 {object} ErrorResponse
```

### 6.7 类型引用两阶段解析

1. **收集阶段**：扫描所有 struct / 类型别名 / 常量 enum，注册到 `spec3.Components.Schemas`
2. **解析阶段**：将类型引用替换为 `spec3.Reference`（`$ref`），检测循环引用

**Schema Key 命名规范（完整类型路径）**：

`Components.Schemas` 的 key 保留完整的包限定路径，确保跨包同名类型不冲突：

| 类型 | Schema Key | `$ref` |
|------|-----------|--------|
| `models.User` | `models.User` | `#/components/schemas/models.User` |
| `admin.User` | `admin.User` | `#/components/schemas/admin.User` |
| `common.PageData` | `common.PageData` | `#/components/schemas/common.PageData` |
| 函数局部类型 `handlers.CreateUser.Request` | `handlers.CreateUser.Request` | `#/components/schemas/handlers.CreateUser.Request` |
| 泛型实例 `common.PageData[models.User]` | `common.PageData[models.User]` | `#/components/schemas/common.PageData[models.User]` |
| 泛型实例 `common.PageData[[]models.User]` | `common.PageData[[]models.User]` | `#/components/schemas/common.PageData[[]models.User]` |

> OpenAPI 3 规范建议组件名符合 `^[a-zA-Z0-9\.\-_]+$`，`[` `]` 为扩展字符。本工具选择保留可读性，生成产物中直接使用完整路径作为 key，消费方（Swagger UI 等）均可正常展示。

**Resolver 内部使用 `SchemaKey` 类型统一标识**，避免裸字符串拼接散落各处：

```go
// resolver.go
type SchemaKey string

func NewSchemaKey(pkg, typeName string) SchemaKey {
    return SchemaKey(pkg + "." + typeName)
}

func GenericSchemaKey(base SchemaKey, args ...SchemaKey) SchemaKey {
    // e.g. "common.PageData[models.User]"
    parts := make([]string, len(args))
    for i, a := range args {
        parts[i] = string(a)
    }
    return SchemaKey(string(base) + "[" + strings.Join(parts, ",") + "]")
}

func (k SchemaKey) Ref() string {
    return "#/components/schemas/" + string(k)
}
```

### 6.9 包别名解析（文件级作用域）

包别名（`import m "github.com/example/models"`）的作用域是**单个文件**。

解析规则：
- `RawField.TypeName` 若含限定符（如 `m.User`），`Resolver` 查当前文件的 `RawFile.Imports` 找到 `m` → 包名 `models`
- 解析后统一转换为 `SchemaKey("models.User")`，用于在 `Components.Schemas` 中查找或注册
- dot-import（`alias="."`) 视为与当前包同级，TypeName 不含限定符时按此回退查找
- **别名仅影响解析阶段**，最终写入 JSON 的 key 始终是 `pkgname.TypeName` 形式，不含 alias

### 6.10 类型别名穿透

`type Foo = Bar` 是透明别名，不产生独立 schema：
- `Resolver` 遇到 `Foo` 时直接穿透为 `Bar` 的 schema
- 若 `Bar` 是外部类型（如 `time.Time`），映射为对应的 OpenAPI 原始类型（`string` + `format: date-time`）

### 6.11 泛型实例化

泛型 struct（`type Resp[T any] struct { Data T }`）的处理：
- **未实例化的泛型定义**不直接注册到 `Components.Schemas`
- 当注解中出现 `common.Resp[models.User]` 时，`SchemaBuilder` 按具体类型参数展开，以 `common.Resp[models.User]` 为 key 注册到 `Components.Schemas`
- 多次使用相同实例化时，`Resolver` 通过 `SchemaKey` 查重，复用已注册的 schema
- 类型参数如为 `[]models.User` / `*models.User` 等复合形式，递归展开，key 中保留完整路径（如 `common.Resp[[]models.User]`）

### 6.12 常量 enum

`const` 块内若多个常量共享同一具名类型（如 `Status`），视为该类型的枚举值集合：
- Parser 将同类型常量收集为 `[]RawConst`，存入 `RawFile.Consts`
- Builder 构建该类型的 schema 时，检查 `Consts` 中是否存在匹配 `TypeName`，若有则附加 `enum` 数组
- 无类型常量（`TypeName==""`）不参与 enum 推断
- enum 值保持常量声明顺序

### 6.13 注解类型查找算法

注解中凡出现类型引用（`@Param body`、`@Success {object}`、组合类型字段值等），均经过以下统一查找流程。查找入口为 `Resolver.Resolve(typeStr string, currentFile *parser.RawFile)`。

#### 第一步：解析类型字符串

类型字符串的完整文法（优先级由上到下）：

```
TypeExpr
  ├─ 前缀 "[]"        → isArray=true；递归解析 elemType
  ├─ 前缀 "*"         → nullable=true；递归解析 elemType
  ├─ 含 "{…}"         → 组合类型：解析 baseType + fieldOverrides{name=TypeExpr, …}
  ├─ 含 "[…]"（泛型）  → 解析 baseType + typeArgs[]（递归解析每个 arg）
  └─ 其余              → 简单类型：按 "." 分段
                         │  3 段  → qualifier=段1，funcName=段2，typeName=段3
                         │  2 段  → qualifier=段1，funcName=""，typeName=段2
                         └─ 1 段  → qualifier=""，typeName=typeStr（禁止，报错）
```

**组合类型语法**（`{…}` 优先于 `[…]` 匹配，避免歧义）：

```
CompositeExpr  = BaseTypeExpr "{" FieldOverrides "}"
FieldOverrides = FieldOverride { "," FieldOverride }
FieldOverride  = fieldName "=" TypeExpr          // TypeExpr 可再次为 CompositeExpr（嵌套）
BaseTypeExpr   = qualifier "." [funcName "."] TypeName  // 必须包限定；funcName 可选（函数局部类型）
```

示例解析结果：

| 注解字符串 | 解析结构 |
|-----------|---------|
| `common.PageData{data=[]models.User}` | base=`common.PageData`，overrides: `data`→`[]models.User` |
| `common.PageData{data=[]models.User,total=int64}` | base=`common.PageData`，overrides: `data`→`[]models.User`, `total`→`int64` |
| `common.Resp{data=common.Page{items=[]models.User}}` | 嵌套组合，`data` 字段值再次为组合类型 |

#### 第二步：解析 qualifier → pkgName

| 情况 | 处理 |
|------|------|
| qualifier == "" | **禁止**（注解强制要求包名限定，报错） |
| qualifier == `currentFile.Package` | 当前包；pkgName = qualifier |
| `currentFile.Imports` 中 `Alias == qualifier` | pkgName = 该条目的 `PkgName` |
| `currentFile.Imports` 中 `PkgName == qualifier` | pkgName = qualifier |
| dot-import（`Alias == "."` 的条目） | 回退：将 typeName 视为当前包或 dot-import 包内的类型 |
| 均未匹配 | 报错：`unknown qualifier "qualifier" in file X` |

`funcName` 若非空，携带至第四步，仅在匹配函数的 `LocalStructs` 中查找；`funcName` 为空时不查 `LocalStructs`。

#### 第三步：内置类型短路

在搜索 `RawFile` 之前，先检查是否命中已知外部类型映射表，命中则直接返回对应 schema，不进入后续查找：

| 类型 | OpenAPI schema |
|------|---------------|
| `time.Time` | `{type: string, format: date-time}` |
| `time.Duration` | `{type: integer, format: int64}` |
| `uuid.UUID` (google/gofrs) | `{type: string, format: uuid}` |
| `decimal.Decimal` (shopspring) | `{type: string, format: decimal}` |
| `json.RawMessage` | `{type: object}` |
| `net.IP` | `{type: string, format: ipv4}` |

> 此映射表可通过 `Config` 扩展（预留 `TypeMappings map[string]SchemaSnippet`）。

#### 第四步：在 RawFile 列表中查找定义

用 `(pkgName, typeName)` 在 `allFiles []*parser.RawFile` 中搜索：

```
for each file in allFiles:
    if file.Package != pkgName → skip

    // 优先级 1：函数局部类型（仅 funcName 非空时）
    if funcName != "" {
        if fn := file.Functions.Find(funcName); fn != nil {
            if local := fn.LocalStructs.Find(typeName); local != nil
                → 构建 schema，SchemaKey = pkgName.funcName.typeName
        }
        continue  // funcName 指定时不查包级定义，未找到则报错
    }

    // 优先级 2：包级 Struct
    if struct := file.Structs.Find(typeName); struct != nil
        → 构建/复用 schema，SchemaKey = pkgName.typeName

    // 优先级 3：类型别名穿透
    if alias := file.TypeAliases.Find(typeName); alias != nil
        → 递归 Resolve(alias.TypeName, file)   ← 用 alias 所在文件作为 currentFile

    // 优先级 4：常量 enum（无 struct 定义的裸类型）
    if consts := file.Consts.FindByType(typeName); len(consts) > 0
        → 推断基础类型（string/int），附加 enum 数组，构建 schema

    // 优先级 5：非 struct 类型定义（type H map[string]any 等）
    if td := file.TypeDefs.Find(typeName); td != nil
        → 递归 Resolve(td.TypeName, file)   ← 透传底层类型，不注册独立 schema key
```

若全部文件遍历完仍未找到（含懒加载后重试，见 § 9）：
- `funcName` 非空时报错：`type "pkgName.funcName.typeName" not found`
- 否则报错：`type "pkgName.typeName" not found in parsed files`

#### 第五步：泛型实例化展开

找到泛型 struct 定义后，将 typeArgs 逐一代入字段：

```
对每个 RawField：
    若 field.TypeName 是类型参数名（在 struct.TypeParams 中）
        → 替换为对应的具体 typeArg，递归 Resolve
    否则
        → 直接 Resolve(field.TypeName, structFile)
```

生成的 schema 以 `GenericSchemaKey(baseKey, argKeys...)` 为 key 注册，相同 key 直接复用。

#### 第六步：组合类型展开

当第一步识别出 `CompositeExpr` 时，走独立的展开路径（不走第二～五步的普通查找）：

```
1. 验证 base qualifier 可解析（未在 Imports 中声明则返回 nil）
2. 递归 Resolve(baseType)          → 得到 baseSchema（$ref）；nil 时返回 nil
3. 对每个 fieldOverride(name, TypeExpr):
       递归 Resolve(TypeExpr)       → 得到 overrideSchema
4. 生成内联 allOf schema（不注册到 Components.Schemas）：
       allOf:
         - $ref: baseKey.Ref()          ← 第一个元素：base 引用
         - type: object
           properties:
             name1: overrideSchema1     ← 第二个元素：覆盖字段
             name2: overrideSchema2
5. 直接返回内联 schema（调用方按需嵌入，无 $ref 指向复合 key）
```

**嵌套组合**递归处理：`overrideSchema` 本身若为组合类型，先走第六步展开，得到其内联 allOf 后作为 override 值嵌入。

> 组合类型 **不注册**到 `Components.Schemas`，不产生 `base{field=Type}` 形式的 key。每个使用点直接持有内联 schema，保持输出 JSON 语义清晰。

#### 完整查找流程图

```
typeStr + currentFile
        │
        ▼
   解析前缀（[] / *）
        │
        ▼
   是否组合类型？──Yes──► 第六步：展开 allOf + overrides → 返回内联 schema
   （含 "{…}"）
        │ No
   是否泛型？──Yes──► 解析 baseType + typeArgs[]
   （含 "[…]"）
        │ No
        ▼
  解析 qualifier → pkgName
  （查 currentFile.Imports）
        │
        ▼
   命中内置类型？──Yes──► 返回内置 schema
        │ No
        ▼
  SchemaKey 已注册？──Yes──► 返回 $ref（循环安全）
        │ No
        ▼
  搜索 allFiles（Package == pkgName）
        │
   ┌────┴──────────┐
  Struct         TypeAlias    Const-enum
   │ （含泛型展开）    │              │
   ▼               ▼              ▼
  构建 Schema   穿透递归        推断基础类型
  （第五步）     Resolve        + enum 数组
        │
        ▼
  注册 SchemaKey → Components.Schemas
        │
        ▼
  返回 $ref
        │
  （若全部文件未命中）
        ▼
  PackageLoader 懒加载（见 § 9）→ 追加文件 → 重试
```

### 6.8 配置

```go
type Config struct {
    InputDirs  StringSlice // 扫描目录（-dirs，逗号分隔或多次传入）
    OutputFile string      // 输出文件路径（-output，默认 ./docs/openapi.json；扩展名决定格式）
    OpenAPIVer string      // OpenAPI 版本（-openapi-ver，默认 "3.0.3"）
    ParseDepth int         // 目录递归深度（-depth，0=仅当前，N=最多N层，-1=无限，默认 -1）
    GoMod      string      // go.mod 路径（-gomod，默认 ./go.mod）
}
```

Title / Version 由注解（`@title` / `@version`）决定，不再作为 CLI 参数。模块缓存懒加载始终启用，无需额外 flag。

---

## 7. 可测试性

不依赖 DDD Port 体系，依然可以充分测试每层：

| 测试目标 | 策略 |
|---------|------|
| `parser.GoParser` | 真实 Go 文件（testdata/），断言 `[]*RawFile`；验证不同 `ParseDepth` 的目录递归行为；验证 import alias / 类型别名 / 泛型 TypeParams / 常量 enum / TypeDef 的正确提取 |
| `parser.module` | `TestParseGoMod`：给定 go.mod 文本，断言 `ModuleInfo.Require` 映射正确；`TestResolvePackageDir`：验证大写字母转义与路径拼接 |
| `extractor.GoExtractor` | 构造 `[]*RawFile`，断言 `ExtractResult` |
| `builder.Builder` | 构造 `ExtractResult` + `[]*RawFile`，断言 `spec3.OpenAPI` |
| `builder.SchemaBuilder` | 构造 `RawStruct`（含泛型参数），断言 `spec3.Schema`；验证泛型实例化去重 |
| `builder.Resolver` | 验证完整查找流程：qualifier 解析、内置类型短路、struct/alias/enum/typedef 五优先级、泛型展开；包别名文件作用域隔离；循环引用检测；断言 `$ref` == `SchemaKey.Ref()` |
| `builder.Builder`（懒加载） | `TestAnnotations_GinTypes_NoLoader`：无 loader 时 gin 类型解析为 nil，路由仍被提取；`TestAnnotations_GinTypes_WithModuleLoader`：注入 `NewModuleLoader` 后 gin.Error / gin.H / gin.ErrorType 均正确解析 |
| `output.Write` | 构造 `spec3.OpenAPI`，断言 JSON/YAML 字节 |
| **端到端** | testdata/ Go 源码 → 期望的 openapi.json golden file |

每层的输入都是可构造的值对象，无需 mock 框架。

---

## 8. 模块依赖关系

```
cmd/swag3/main.go
    │
    ├── config/
    ├── internal/parser          ← go/ast, go/types
    ├── internal/extractor       ← internal/parser (RawAST)
    ├── internal/builder         ← internal/extractor (ExtractResult)
    │                            ← internal/parser ([]*RawFile)
    │                            ← spec3 (github.com/shuttlefy/go-openapi3-spec)
    └── internal/output          ← spec3

依赖方向: cmd → 各 internal 包 → 标准库 / spec3
```

**外部依赖**：
- `github.com/shuttlefy/go-openapi3-spec` — OpenAPI 3 结构体 + JSON/YAML 序列化
- 标准库：`go/ast`, `go/types`, `go/parser`, `encoding/json`, `os`

---

## 9. 第三方包懒加载（模块缓存）

### 9.1 背景

当前 `Resolver` 只能在显式 `Parse` 的目录里查找类型。注解中引用第三方包类型（如 `gin.H`、`gin.Error`）时，因为 gin 源码未被解析，`lookupAndBuild` 找不到对应文件，直接返回 `nil`，导致响应 schema 缺失。

**解决方案**：「懒加载」—— 当 `Resolver` 在已知文件里找不到某个类型时，自动定位并解析模块缓存中的对应包，然后重试。

### 9.2 涉及文件

| 文件 | 操作 |
|------|------|
| `internal/parser/module.go` | 新建 —— go.mod 解析 + 模块缓存路径解析 |
| `internal/parser/model.go` | 修改 —— 新增 `RawTypeDef` + `RawFile.TypeDefs` |
| `internal/parser/parser.go` | 修改 —— 新增 `ParseDir`；`extractGenDecl` 捕获非 struct 类型定义 |
| `internal/builder/resolver.go` | 修改 —— 新增 `PackageLoader`、`loadedPkgs`、懒加载、TypeDef 查找；处理 `error` 内置类型 |
| `internal/builder/builder.go` | 修改 —— 新增 `SetLoader` / `NewModuleLoader` |
| `internal/builder/builder_test.go` | 修改 —— 拆分 `TestAnnotations_GinTypes`，新增 WithModuleLoader 变体 |

### 9.3 `parser/module.go` — 模块缓存定位

```go
type ModuleInfo struct {
    Module  string            // "github.com/foo/bar"
    Require map[string]string // importPath → version
}

func ParseGoMod(gomodPath string) (*ModuleInfo, error)
// 逐行扫描，支持 require (...) 块和单行 require，去掉 // indirect 注释

func ModuleCacheDir() string
// go env GOMODCACHE，fallback $GOPATH/pkg/mod

func ResolvePackageDir(importPath string, info *ModuleInfo, cacheDir string) (string, bool)
// 在 info.Require 里找最长前缀匹配，拼出 cacheDir/mod!path@version/subpkg

func escapeModulePath(p string) string
// 大写字母 → !小写，符合 Go module 缓存目录命名规范
```

### 9.4 `builder/resolver.go` — PackageLoader 与懒加载

```go
// PackageLoader 按需加载第三方包源文件。
// pkgName 是短包名（如 "gin"），srcFile 是引用该包的源文件（用于查 import path）。
type PackageLoader func(pkgName string, srcFile *parser.RawFile) []*parser.RawFile

// Resolver 新增字段
loader     PackageLoader
loadedPkgs map[string]bool  // key = import path，防止重复加载
```

**懒加载触发逻辑**（在 `lookupAndBuild` 末尾，遍历完所有已知文件后执行）：

```go
if r.loader != nil && currentFile != nil {
    importPath := findImportPath(pkg, currentFile)
    cacheKey := importPath; if cacheKey == "" { cacheKey = pkg }
    if !r.loadedPkgs[cacheKey] {
        r.loadedPkgs[cacheKey] = true
        if loaded := r.loader(pkg, currentFile); len(loaded) > 0 {
            r.files = append(r.files, loaded...)
            return r.lookupAndBuild(key, pkg, typeName, funcName, currentFile) // 重试
        }
    }
}
return nil
```

**`error` 内置类型处理**（在 `Resolve` switch 中新增）：

```go
case "interface{}", "any", "error":
    return &spec3.Schema{} // error 是内置接口，映射为空 schema
```

### 9.5 `builder/builder.go` — NewModuleLoader

```go
// NewModuleLoader 创建基于模块缓存的 PackageLoader。
func NewModuleLoader(modInfo *parser.ModuleInfo, cacheDir string) PackageLoader {
    p := &parser.GoParser{}
    loaded := make(map[string]bool)
    return func(pkgName string, srcFile *parser.RawFile) []*parser.RawFile {
        importPath := findImportPathFromFile(pkgName, srcFile)
        if importPath == "" || loaded[importPath] { return nil }
        loaded[importPath] = true
        dir, ok := parser.ResolvePackageDir(importPath, modInfo, cacheDir)
        if !ok { return nil }
        files, _ := p.ParseDir(dir)
        return files
    }
}
```

**CLI 中始终启用懒加载**（`main.go` 默认行为，go.mod 解析失败时降级运行）：

```go
modInfo, err := parser.ParseGoMod("go.mod")
if err != nil {
    log.Printf("warn: parse go.mod: %v (module loader disabled)", err)
} else {
    b.SetLoader(builder.NewModuleLoader(modInfo, parser.ModuleCacheDir()))
}
```

### 9.6 解析预期结果

| 类型 | 底层定义 | 解析结果 |
|------|---------|---------|
| `gin.H` | `type H map[string]any` | `{type: object}`（inline，TypeDef 透传） |
| `gin.Error` | struct | `$ref: #/components/schemas/gin.Error`（注册） |
| `gin.ErrorType` | `type ErrorType uint64` | `{type: integer, format: int64}`（inline） |
| `error` | 内置接口 | `{}`（空 schema） |

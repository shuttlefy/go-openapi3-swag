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
│   │   └── model.go               # RawAST / RawFunc / RawStruct
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

**职责**：将 Go 源文件解析成原始 AST 表示。

| 组件 | 职责 |
|------|------|
| `GoParser` | 使用 `go/ast` + `go/types` 解析 Go 源码 |
| `model.go` | `RawAST`、`RawFunc`、`RawStruct` 数据结构 |

```go
type GoParser struct{}

func (p *GoParser) Parse(dirs []string) (*RawAST, error)
```

**输入**：源文件目录列表
**输出**：`*RawAST`

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

func (e *GoExtractor) Extract(ast *parser.RawAST) (*ExtractResult, error)
```

**输入**：`*parser.RawAST`
**输出**：`*ExtractResult`

---

### 2.3 `builder` — spec3 构建（核心逻辑）

**职责**：将注解 + 类型信息直接构建为 `spec3.OpenAPI`。

| 组件 | 职责 |
|------|------|
| `Builder` | 编排 SchemaBuilder + OperationBuilder，组装 `spec3.OpenAPI` |
| `SchemaBuilder` | Go struct → `spec3.Schema`，注册到 `Components.Schemas` |
| `OperationBuilder` | 注解 → `spec3.Operation` |
| `Resolver` | `$ref` 引用生成 + 循环检测 |

**输入**：`*extractor.ExtractResult` + `*parser.RawAST`
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
│ parser.GoParser      │  go/ast 解析
│ Parse(dirs)          │
└──────────┬───────────┘
           │ *parser.RawAST
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

```go
package parser

type RawAST struct {
    Package   string
    Functions []RawFunc
    Structs   []RawStruct
}

type RawFunc struct {
    Name     string
    FilePath string
    Line     int
    Comments []string
    Receiver string
    Params   []RawParam
    Results  []RawParam
}

type RawStruct struct {
    Name     string
    FilePath string
    Fields   []RawField
    Comments []string
}

type RawField struct {
    Name     string
    TypeName string
    Tag      string
    Comments []string
}

type RawParam struct {
    Name     string
    TypeName string
}
```

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

func (b *Builder) Build(result *extractor.ExtractResult, rawAST *parser.RawAST) (*spec3.OpenAPI, error) {
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

    p := &parser.GoParser{}
    rawAST, err := p.Parse(cfg.InputDirs)
    if err != nil {
        log.Fatal(err)
    }

    e := &extractor.GoExtractor{}
    result, err := e.Extract(rawAST)
    if err != nil {
        log.Fatal(err)
    }

    b := builder.NewBuilder()
    doc, err := b.Build(result, rawAST)
    if err != nil {
        log.Fatal(err)
    }

    if err := output.Write(doc, cfg.OutputFormat, cfg.OutputFile); err != nil {
        log.Fatal(err)
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

1. **收集阶段**：扫描所有 struct，注册到 `spec3.Components.Schemas`
2. **解析阶段**：将类型引用替换为 `spec3.Reference`（`$ref`），检测循环引用

### 6.8 配置

```go
type Config struct {
    InputDirs    []string  // 扫描目录
    OutputFile   string    // 输出文件路径（默认 ./openapi.json）
    OutputFormat string    // "json" | "yaml"
    Title        string    // API 标题（可被注解覆盖）
    Version      string    // API 版本（可被注解覆盖）
    OpenAPIVer   string    // "3.0.3"（默认）| "3.1.0" | "3.2.0"
}
```

---

## 7. 可测试性

不依赖 DDD Port 体系，依然可以充分测试每层：

| 测试目标 | 策略 |
|---------|------|
| `parser.GoParser` | 真实 Go 文件（testdata/），断言 `RawAST` |
| `extractor.GoExtractor` | 构造 `RawAST`，断言 `ExtractResult` |
| `builder.Builder` | 构造 `ExtractResult` + `RawAST`，断言 `spec3.OpenAPI` |
| `builder.SchemaBuilder` | 构造 `RawStruct`，断言 `spec3.Schema` |
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
    │                            ← internal/parser (RawAST)
    │                            ← spec3 (github.com/shuttlefy/go-openapi3-spec)
    └── internal/output          ← spec3

依赖方向: cmd → 各 internal 包 → 标准库 / spec3
```

**外部依赖**：
- `github.com/shuttlefy/go-openapi3-spec` — OpenAPI 3 结构体 + JSON/YAML 序列化
- 标准库：`go/ast`, `go/types`, `go/parser`, `encoding/json`, `os`

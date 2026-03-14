# Swag3 实施计划

基于 [design.md](./design.md) 的架构设计，按依赖链自底向上实施。

---

## 阶段一：项目脚手架 + 基础数据结构

**目标**：建立项目骨架，定义各层的数据结构，确保编译通过。

- [ ] 1.1 初始化项目结构
  - 创建目录：`cmd/swag3/`、`internal/parser/`、`internal/extractor/`、`internal/builder/`、`internal/output/`、`config/`、`testdata/go/`、`testdata/expected/`
  - `go get github.com/shuttlefy/go-openapi3-spec`
  - 创建 `.gitignore`

- [ ] 1.2 `config/config.go` — 全局配置
  - `Config` 结构体（InputDirs、OutputFile、OutputFormat、Title、Version、OpenAPIVer）
  - `ParseFlags()` 函数，使用 `flag` 标准库解析 CLI 参数
  - 默认值：OutputFile=`./openapi.json`、OutputFormat=`json`、OpenAPIVer=`3.0.3`

- [ ] 1.3 `internal/parser/model.go` — 解析产物数据结构
  - `RawAST`、`RawFunc`、`RawStruct`、`RawField`、`RawParam`

- [ ] 1.4 `internal/extractor/annotation.go` — 注解数据结构
  - `ExtractResult`、`GlobalAnnotation`、`OperationAnnotation`
  - `RouteInfo`、`ParamAnnotation`、`ResponseAnnotation`
  - `TagAnnotation`、`SecurityDefAnnotation`

- [ ] 1.5 `internal/output/writer.go` — 输出函数
  - `Write(doc *spec3.OpenAPI, format, path string) error`
  - JSON: `json.MarshalIndent`；YAML: `spec3.MarshalYAML`

- [ ] 1.6 `cmd/swag3/main.go` — 最小入口骨架
  - 解析配置，按流水线顺序调用各模块（先占位 `log.Fatal("not implemented")`）

**验收**：`go build ./...` 编译通过。

---

## 阶段二：Parser — Go AST 解析

**目标**：扫描 Go 源文件，提取函数声明、结构体定义、注释块。

**依赖**：阶段一（model.go）

- [ ] 2.1 `internal/parser/parser.go` — GoParser 实现
  - `Parse(dirs []string) (*RawAST, error)`
  - 遍历目录，`go/parser.ParseDir` 解析每个包
  - 提取 `*ast.FuncDecl` → `RawFunc`（含注释、接收者、参数、返回值）
  - 提取 `*ast.TypeSpec` + `*ast.StructType` → `RawStruct`（含字段、tag、注释）
  - 使用 `go/types` 解析完整类型名称（处理包引用、指针、切片等）

- [ ] 2.2 `testdata/go/simple.go` — 测试用 Go 源文件
  - 包含一个带注解的 handler 函数 + 一个 struct

- [ ] 2.3 `internal/parser/parser_test.go` — 单元测试
  - 输入 `testdata/go/simple.go`，断言 `RawAST` 中的函数数量、结构体数量
  - 断言函数名、注释行、参数类型
  - 断言结构体字段名、类型名、tag 值

**验收**：`go test ./internal/parser/` 全部通过。

---

## 阶段三：Extractor — 注释提取

**目标**：将 `RawAST` 中的原始注释解析为结构化注解。

**依赖**：阶段一（annotation.go）+ 阶段二（RawAST 可构造）

- [ ] 3.1 `internal/extractor/tag_parser.go` — 标签解析器
  - 解析 `@Summary`、`@Description`、`@Router`（注意兼容 swaggo 的 @Router）
  - 解析 `@Param name in type required "description"`
  - 解析 `@Success code {type} TypeName "description"`、`@Failure` 同理
  - 解析 `@Tags`、`@Security`、`@Deprecated`
  - 解析 `@ID`（operationId）

- [ ] 3.2 `internal/extractor/extractor.go` — GoExtractor 实现
  - `Extract(ast *parser.RawAST) (*ExtractResult, error)`
  - 识别全局注解（`main` 包的包级注释或标记函数的注释）
  - 解析全局 `@title`、`@version`、`@host`、`@BasePath`、`@schemes`
  - 遍历 `RawFunc`，对每个函数的 Comments 调用 TagParser
  - 将每个函数的标签集合组装为 `OperationAnnotation`
  - 跳过无 `@Router` 注解的函数

- [ ] 3.3 `internal/extractor/extractor_test.go` — 单元测试
  - 构造 `RawAST`（手写注释行），断言 `ExtractResult`
  - 覆盖：单函数单路由、多参数、body 参数、多响应码、全局注解

**验收**：`go test ./internal/extractor/` 全部通过。

---

## 阶段四：Builder — spec3 构建（核心）

**目标**：将 `ExtractResult` + `RawAST` 转换为完整的 `*spec3.OpenAPI`。

**依赖**：阶段一 + 阶段二 + 阶段三（数据结构可构造）

- [ ] 4.1 `internal/builder/resolver.go` — 类型引用解析
  - 维护已注册的 schema name 集合
  - `Register(typeName string)` — 收集阶段
  - `RefSchema(typeName string) *spec3.Schema` — 返回 `$ref` 引用
  - `IsRegistered(typeName string) bool`
  - 循环引用检测（visited set）

- [ ] 4.2 `internal/builder/schema_builder.go` — Schema 构建
  - `NewSchemaBuilder(resolver *Resolver) *SchemaBuilder`
  - `Build(typeName string, rawStruct *parser.RawStruct) *spec3.Schema`
    - 遍历 `RawField` → `spec3.Schema` 属性
    - 基本类型映射：`string`→string、`int`/`int64`→integer、`float64`→number、`bool`→boolean
    - 指针类型（`*T`）→ 去掉指针，标记 nullable
    - 切片类型（`[]T`）→ type=array + items
    - 嵌套 struct 引用 → `$ref`（通过 Resolver）
    - 提取 `json:"name"` tag 作为属性名
    - 提取 `binding:"required"` 或 `validate:"required"` → required 列表
  - `Components() *spec3.Components` — 返回收集的所有 Schemas

- [ ] 4.3 `internal/builder/operation_builder.go` — Operation 构建
  - `NewOperationBuilder(schema *SchemaBuilder) *OperationBuilder`
  - `Build(anno extractor.OperationAnnotation) (*spec3.Operation, error)`
    - Tags、Summary、Description、OperationID、Deprecated → 直接映射
    - `ParamAnnotation`（in != "body"）→ `spec3.Parameter`
      - 类型映射（string/int/bool → schema.Type + Format）
      - path 参数自动 required=true
    - `ParamAnnotation`（in == "body"）→ `spec3.RequestBody`
      - Content: `application/json` → `spec3.MediaType{Schema: ...}`
    - `ResponseAnnotation` → `spec3.Response`
      - IsArray=true → schema.Type=array + items
      - Content: `application/json`
    - Security → `spec3.SecurityRequirement`

- [ ] 4.4 `internal/builder/builder.go` — 主编排
  - `NewBuilder() *Builder`
  - `Build(result *extractor.ExtractResult, rawAST *parser.RawAST) (*spec3.OpenAPI, error)`
    - 阶段 1：遍历 `rawAST.Structs`，SchemaBuilder 注册所有 struct
    - 阶段 2：构建 `spec3.Info`、`spec3.Server`
    - 阶段 3：遍历 `result.Operations`，OperationBuilder 构建每个 Operation
    - 阶段 4：组装 Paths（`spec3.Paths`）+ Components（`spec3.Components`）
    - 阶段 5：组装 Tags（`spec3.Tag`）
    - 返回 `*spec3.OpenAPI`

- [ ] 4.5 `internal/builder/builder_test.go` — 单元测试
  - 构造 `ExtractResult` + `RawAST`，断言 `spec3.OpenAPI` 各字段
  - 覆盖：基本 CRUD 操作、嵌套 struct $ref、数组响应、多 tag

- [ ] 4.6 `internal/builder/schema_builder_test.go` — Schema 单元测试
  - 覆盖：基本类型、指针、切片、嵌套 struct、required 提取

**验收**：`go test ./internal/builder/` 全部通过。

---

## 阶段五：端到端串联 + CLI

**目标**：将所有模块串联，CLI 可执行端到端生成。

**依赖**：阶段一～四全部完成

- [ ] 5.1 `cmd/swag3/main.go` — 完整实现
  - 解析 CLI flags → Config
  - Parser.Parse → Extractor.Extract → Builder.Build → output.Write
  - 错误处理 + 友好错误信息

- [ ] 5.2 `testdata/go/petstore.go` — 端到端测试用例
  - 模拟一个小型 Pet Store API（3-5 个 handler + 2-3 个 struct）
  - 覆盖：GET/POST/PUT/DELETE、path/query 参数、request body、多响应码

- [ ] 5.3 `testdata/expected/petstore.json` — 期望输出 golden file
  - 手写或首次生成后 review 固定

- [ ] 5.4 端到端测试
  - `TestEndToEnd_PetStore`：读取 `testdata/go/petstore.go` → 生成 → 与 golden file 比对
  - JSON 和 YAML 两种格式各测一次

**验收**：`go test ./...` 全部通过；`go run ./cmd/swag3/ -dir testdata/go -output /tmp/openapi.json` 生成合法 OpenAPI 3 文档。

---

## 阶段六：健壮性 + 边界情况

**目标**：处理真实项目中的复杂情况。

**依赖**：阶段五

- [ ] 6.1 类型解析增强
  - `map[string]T` → additionalProperties
  - `time.Time` → type=string, format=date-time
  - `interface{}` / `any` → type 为空（任意类型）
  - 匿名 struct 嵌入（embedded struct）→ allOf 展开

- [ ] 6.2 循环引用处理
  - struct A 引用 struct B 引用 struct A → 检测并用 `$ref` 打断
  - 自引用（如树结构 `Children []*Node`）→ 同上

- [ ] 6.3 错误处理与诊断
  - 注解格式错误 → 报告文件名 + 行号 + 具体错误
  - 缺少 `@Router` 但有其他注解 → 警告
  - 引用了未定义的 struct → 错误并列出可能的候选

- [ ] 6.4 补充测试
  - 循环引用 testcase
  - 错误注解 testcase（格式错、类型不存在）
  - 空项目（无注解函数）→ 生成最小合法文档

**验收**：`go test ./...` 全部通过，包括边界情况测试。

---

## 优先级与依赖关系

```
阶段一（脚手架）
  │
  ├──→ 阶段二（Parser）──→ 阶段三（Extractor）──→ 阶段四（Builder）
  │                                                     │
  │                                                     ▼
  └─────────────────────────────────────────────→ 阶段五（串联 + CLI）
                                                        │
                                                        ▼
                                                  阶段六（健壮性）
```

- 阶段一是所有后续阶段的前置
- 阶段二、三、四是线性依赖链（每阶段依赖前一阶段的数据结构）
- 阶段五依赖二、三、四全部完成
- 阶段六可在阶段五之后逐步补充

---

## 关键实现要点

### Parser 要点
- 使用 `go/parser.ParseDir` 而非 `ParseFile`，一次解析整个包获取完整类型信息
- `go/types.Config.Check` 获取完整类型信息（处理跨包引用、类型别名）
- 注释提取用 `ast.FuncDecl.Doc`（函数前方的文档注释），忽略行内注释

### Extractor 要点
- 注解标签名大小写不敏感（`@summary` = `@Summary`）
- `@Router` 格式：`/path [method]`，method 不含大括号
- `@Param` 格式：`name in type required "description"`，required 是 `true`/`false` 字符串
- `@Success`/`@Failure` 格式：`code {type} TypeName "description"`，type 可选 `object`/`array`/`string` 等

### Builder 要点
- Schema 收集必须在 Operation 构建之前完成（Resolver 需要知道所有已注册类型）
- `spec3.NewPaths()`、`spec3.NewOrderedResponses()` 等工厂函数确保 OrderedMap 正确初始化
- `spec3.Reference` 的 `$ref` 值格式：`#/components/schemas/TypeName`
- 使用 `spec3.OrderedSchemas.Set()` 保持 Schema 注册顺序

### Output 要点
- `json.MarshalIndent(doc, "", "  ")` 用于 JSON 美化输出
- `spec3.MarshalYAML(doc)` 内部走 JSON → YAML 转换，保留所有自定义序列化行为

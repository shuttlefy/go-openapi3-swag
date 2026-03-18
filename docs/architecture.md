# 整体架构

## 流水线概览

swag3 采用四阶段线性流水线，每阶段职责单一、输入/输出类型明确：

```
Go 源文件 (*.go)
      │
      ▼
┌─────────────────────────┐
│  Stage 1: parser        │  go/ast 递归解析
│  GoParser.Parse()       │
└──────────┬──────────────┘
           │ []*parser.RawFile
           ▼
┌─────────────────────────┐
│  Stage 2: extractor     │  // @Tag 注释解析
│  GoExtractor.Extract()  │
└──────────┬──────────────┘
           │ *extractor.ExtractResult
           ▼
┌─────────────────────────┐
│  Stage 3: builder       │  类型解析 + spec3 构建
│  Builder.Build()        │
└──────────┬──────────────┘
           │ *spec3.OpenAPI
           ▼
┌─────────────────────────┐
│  Stage 4: output        │  JSON / YAML 序列化
│  output.Write()         │
└──────────┬──────────────┘
           │
           ▼
  openapi.json / openapi.yaml
```

## 模块依赖关系

```
cmd/swag3/main.go
    ├── config/
    ├── internal/parser          ← go/ast, go/types（标准库）
    ├── internal/extractor       ← internal/parser
    ├── internal/builder         ← internal/extractor, internal/parser
    │                               spec3 (go-openapi3-spec)
    └── internal/output          ← spec3

pkg/swaggin/                     ← gin（独立，不依赖 internal/）
```

依赖方向严格单向：`cmd → internal/* → 标准库 / spec3`，`internal/` 各包间无循环依赖。

## 核心外部依赖

| 依赖 | 用途 |
|------|------|
| `github.com/shuttlefy/go-openapi3-spec` | OpenAPI 3.x Go 结构体 + JSON/YAML 序列化 |
| `go/ast`, `go/types`, `go/parser` | Go 源码 AST 解析（标准库） |
| `github.com/gin-gonic/gin` | swaggin 包的 HTTP 路由集成 |

## 设计原则

### 直接构建 spec3，无自定义 IR

`go-openapi3-spec` 已提供完整的 OpenAPI 3 结构体，`builder` 直接产出 `*spec3.OpenAPI`，不引入中间表示层。

### 务实分层，按需使用接口

| 做 | 不做 |
|----|------|
| 在真正需要可测试性的边界使用接口（`PackageLoader`） | 为只有一个实现的组件强制抽象 |
| 数据结构定义在产出方包中 | 创建独立的 `domain/model` 包 |
| 构建逻辑不依赖文件 I/O | 为 Parser/Extractor/Writer 创建 Port 接口 |

### 文件级 import 作用域

包别名（`import m "pkg/path"`）的作用域是单个文件，`Resolver` 解析类型引用时始终携带当前文件的 `Imports` 上下文。

### 两阶段类型解析

1. **收集阶段**：扫描所有 struct / 类型别名 / 常量 enum，注册到 `Components.Schemas`
2. **引用阶段**：将类型引用替换为 `$ref`，同时进行循环引用检测

### 懒加载第三方包

当 `Resolver` 在已扫描文件中找不到某类型时，通过 `PackageLoader` 自动定位并解析模块缓存中的对应包，然后重试。CLI 默认启用。

## Schema Key 命名规则

`Components.Schemas` 的 key 使用完整包限定路径，确保跨包同名类型不冲突：

| 类型 | Schema Key |
|------|-----------|
| 包级 struct/enum | `pkg.TypeName` |
| 函数局部 struct | `pkg.FuncName.TypeName` |
| 泛型实例化 | `pkg.TypeName[ArgKey,...]` |
| 组合类型 `Base{field=T}` | 不注册（内联 allOf） |
| 透明别名 `type A = B` | 穿透为 B |
| 非 struct 类型定义 | 穿透底层类型，不注册自身 |

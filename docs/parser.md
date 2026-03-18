# parser 包（Stage 1）

**代码位置**：`internal/parser/`

| 文件 | 职责 |
|------|------|
| `parser.go` | `GoParser`：递归扫描目录，用 `go/ast` 解析 `.go` 文件 |
| `model.go` | 解析产物数据结构（`RawFile`、`RawStruct` 等） |
| `module.go` | `go.mod` 解析 + 模块缓存路径解析（供懒加载使用） |

## 职责

将 Go 源文件递归解析为结构化的 AST 节点列表，供后续阶段使用。不做语义分析，不处理注解。

## 核心类型

### `GoParser`

```go
type GoParser struct {
    MaxDepth int // 目录递归深度，来自 Config.ParseDepth
}

func (p *GoParser) Parse(dirs []string) ([]*RawFile, error)
func (p *GoParser) ParseDir(dir string) ([]*RawFile, error) // 仅解析单层目录（懒加载使用）
```

**递归规则**：

| `MaxDepth` | 行为 |
|-----------|------|
| `-1` | 无限递归 |
| `0` | 仅扫描 `dirs` 本身 |
| `N` | 最多向下 N 层子目录 |

### `RawFile`

一个 `.go` 源文件对应一个 `RawFile`。**包别名（import alias）的作用域是文件级**，不跨文件共享。

```go
type RawFile struct {
    Package     string
    FilePath    string
    Imports     []RawImport    // import 声明，仅本文件有效
    Functions   []RawFunc
    Structs     []RawStruct
    TypeAliases []RawTypeAlias // type Foo = Bar（透明别名）
    TypeDefs    []RawTypeDef   // type Foo Bar（非 struct 新类型）
    Consts      []RawConst     // 具名常量（含 enum 候选）
}
```

### `RawImport`

```go
type RawImport struct {
    Alias   string // 显式别名；"" 表示使用包本名；"." 表示 dot-import
    Path    string // import 路径，如 "github.com/example/models"
    PkgName string // 包的实际名称（import path 最后一段的近似值）
}
```

### `RawFunc`

```go
type RawFunc struct {
    Name         string
    FilePath     string
    Line         int
    Comments     []string    // 原始注释行（含 @Tag 注解）
    Receiver     string      // 方法接收者；函数为 ""
    Params       []RawParam
    Results      []RawParam
    LocalStructs []RawStruct // 函数体内定义的局部 struct
}
```

### `RawStruct`

```go
type RawStruct struct {
    Name       string
    FilePath   string
    Fields     []RawField
    Comments   []string
    TypeParams []RawTypeParam // 泛型参数，如 [T any, U comparable]
}

type RawTypeParam struct {
    Name       string // 参数名，如 "T"
    Constraint string // 约束，如 "any" / "comparable"
}

type RawField struct {
    Name     string
    TypeName string   // 可能是类型参数名（"T"）或具体类型（"models.User"）
    Tag      string   // 原始 struct tag，不含反引号
    Comments []string
    Embedded bool     // 是否为嵌入字段
}
```

### `RawTypeAlias`

对应 `type Foo = Bar`（透明别名，非新类型）。

```go
type RawTypeAlias struct {
    Name     string
    TypeName string   // 右侧类型，可能含包限定符，如 "time.Time"
    FilePath string
    Comments []string
}
```

### `RawTypeDef`

对应非 struct 的新类型定义，如 `type H map[string]any`、`type Params []Param`。

```go
type RawTypeDef struct {
    Name     string
    TypeName string   // 底层类型字符串，如 "map[string]interface{}" / "[]Param"
    FilePath string
    Comments []string
}
```

### `RawConst`

对应具名常量，用于推断 enum 值。无类型常量（`TypeName == ""`）不收录。

```go
type RawConst struct {
    Name     string
    TypeName string   // 显式类型名，如 "Status"
    Value    string   // 字面量值，如 "active" 或 "0"
    FilePath string
    Comments []string // 行尾注释，用于 x-enumdescriptions
}
```

**iota 支持**：`constValueToString` 处理 `iota`、`1 << iota`、二元表达式等多种形式。

## module.go — 模块缓存定位

供 `builder.NewModuleLoader` 使用，实现第三方包懒加载。

```go
type ModuleInfo struct {
    Module  string            // 当前模块路径，如 "github.com/foo/bar"
    Require map[string]string // importPath → version
}

// 解析 go.mod，提取 module 名和所有 require 条目
func ParseGoMod(gomodPath string) (*ModuleInfo, error)

// 返回 Go 模块缓存目录（go env GOMODCACHE，fallback $GOPATH/pkg/mod）
func ModuleCacheDir() string

// 在模块缓存中定位包目录（最长前缀匹配 + 大写字母转义）
func ResolvePackageDir(importPath string, info *ModuleInfo, cacheDir string) (string, bool)
```

大写字母转义规则（Go module 缓存命名规范）：`A` → `!a`，例如 `github.com/BurntSushi/toml` → `github.com/!burnt!sushi/toml`。

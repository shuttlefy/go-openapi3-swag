package parser

// RawFile 对应一个 .go 源文件的解析结果。
// 包别名（Imports）作用域为文件级，跨文件不共享。
type RawFile struct {
	Package     string
	FilePath    string
	Imports     []RawImport    // import 声明，仅本文件有效
	Functions   []RawFunc
	Structs     []RawStruct
	TypeAliases []RawTypeAlias // type Foo = Bar（透明别名）
	TypeDefs    []RawTypeDef   // type Foo Bar（非 struct、非透明别名的新类型定义）
	Consts      []RawConst     // 具名常量（含 enum 候选）
}

// RawImport 记录一条 import 声明。
//
//   - Alias == ""  → 使用包的默认名称（PkgName）
//   - Alias == "." → dot-import
//   - 其他          → 显式别名
//
// 空白 import（`_`）不收录。
type RawImport struct {
	Alias   string // 显式别名 / "." / ""
	Path    string // import path，如 "github.com/example/models"
	PkgName string // 包的实际名称（import path 最后一段的近似值）
}

// RawTypeAlias 对应 type Foo = Bar（透明别名，非新类型定义）。
type RawTypeAlias struct {
	Name     string
	TypeName string   // 右侧类型，可能含包限定符，如 "time.Time"
	FilePath string
	Comments []string
}

// RawTypeDef 表示非 struct 的新类型定义，例如：
//   - type H map[string]any
//   - type Params []Param
//   - type ErrorType uint64
type RawTypeDef struct {
	Name     string
	TypeName string   // 底层类型字符串，如 "map[string]interface{}" / "[]Param" / "uint64"
	FilePath string
	Comments []string
}

// RawConst 对应一个具名常量，通常用于 string / int enum。
// 无类型常量（TypeName == ""）不收录。
type RawConst struct {
	Name     string
	TypeName string   // 显式类型名，如 "Status"
	Value    string   // 字面量值，如 `active` 或 `0`
	FilePath string
	Comments []string
}

// RawFunc 对应一个函数或方法声明。
type RawFunc struct {
	Name         string
	FilePath     string
	Line         int
	Comments     []string
	Receiver     string      // 方法接收者类型字符串，如 "*UserService"；函数为 ""
	Params       []RawParam
	Results      []RawParam
	LocalStructs  []RawStruct  // 函数体内定义的局部 struct，作用域仅限本函数
	LocalTypeDefs []RawTypeDef // 函数体内的非 struct 类型定义（type Foo Bar）
}

// RawStruct 对应一个 struct 类型声明，支持泛型。
type RawStruct struct {
	Name       string
	FilePath   string
	Fields     []RawField
	Comments   []string
	TypeParams []RawTypeParam // 泛型参数，非空表示该 struct 有类型参数
}

// RawTypeParam 描述一个泛型类型参数，如 [T any]。
type RawTypeParam struct {
	Name       string // 参数名，如 "T"
	Constraint string // 约束表达式，如 "any" / "comparable" / "int|string"
}

// RawField 对应 struct 中的一个字段。
type RawField struct {
	Name     string   // 字段名；嵌入字段使用类型的基名（如 "BaseModel"）
	TypeName string   // 类型字符串，如 "string" / "*models.User" / "[]int"
	Tag      string   // 原始 struct tag，不含反引号
	Comments []string
	Embedded bool     // true 表示嵌入字段（匿名字段）
}

// RawParam 对应函数参数或返回值。
type RawParam struct {
	Name     string // 可为 ""（匿名参数）
	TypeName string
}

// Package types 提供覆盖所有 Go 类型使用场景的测试数据。
// 供 GoParser 的解析测试使用，不含可执行逻辑。
package types

import (
	"cmp"
	"context"
	alias "encoding/json"
	"net/url"
	"time"
	"unsafe"
)

// ─────────────────────────────────────────────
// 1. 基础标量字段（所有内置原始类型）
// ─────────────────────────────────────────────

// Primitives 包含所有 Go 内置原始类型字段。
type Primitives struct {
	B    bool
	I    int
	I8   int8
	I16  int16
	I32  int32
	I64  int64
	U    uint
	U8   uint8
	U16  uint16
	U32  uint32
	U64  uint64
	Uptr uintptr
	F32  float32
	F64  float64
	C64  complex64
	C128 complex128
	By   byte   // byte == uint8
	R    rune   // rune == int32
	S    string
	Up   unsafe.Pointer
}

// ─────────────────────────────────────────────
// 2. 指针类型
// ─────────────────────────────────────────────

// Pointers 包含各种指针字段。
type Pointers struct {
	PS   *string
	PI   *int
	PF   *float64
	PB   *bool
	PP   **string       // 双重指针
	PT   *time.Time
	PU   *url.URL
	Self *Pointers      // 自身指针（递归类型）
}

// ─────────────────────────────────────────────
// 3. 切片与固定长度数组
// ─────────────────────────────────────────────

// Slices 包含切片和数组字段。
type Slices struct {
	Strings  []string
	Ints     []int
	Float64s []float64
	Bytes    []byte         // []byte 常见场景
	Any      []interface{}
	PtrSlice []*string
	Nested   [][]int        // 二维切片

	Arr3   [3]string        // 固定长度数组，parser 统一视为切片
	Arr0   [0]byte
	Matrix [4][4]float64
}

// ─────────────────────────────────────────────
// 4. Map 类型
// ─────────────────────────────────────────────

// Maps 包含各种 map 字段。
type Maps struct {
	StrToInt      map[string]int
	StrToStr      map[string]string
	StrToAny      map[string]interface{}
	StrToPtr      map[string]*string
	StrToSlice    map[string][]int
	IntToStr      map[int]string
	StrToStruct   map[string]Primitives
	NestedMap     map[string]map[string]int
}

// ─────────────────────────────────────────────
// 5. 嵌入字段（匿名字段）
// ─────────────────────────────────────────────

// BaseModel 是被嵌入的基类型。
type BaseModel struct {
	ID        int64  `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Auditable 另一个可嵌入类型。
type Auditable struct {
	CreatedBy string `json:"created_by"`
	UpdatedBy string `json:"updated_by"`
}

// Embedded 包含嵌入字段（匿名字段）。
type Embedded struct {
	BaseModel           // 值类型嵌入
	*Auditable          // 指针嵌入
	Name string `json:"name"`
}

// DeepEmbed 多级嵌入。
type DeepEmbed struct {
	Embedded
	Extra string
}

// ─────────────────────────────────────────────
// 6. 匿名 struct 字段
// ─────────────────────────────────────────────

// WithAnon 包含直接匿名 struct 字段（非切片/指针包装）。
type WithAnon struct {
	Meta struct {
		Version int    `json:"version"`
		Tag     string `json:"tag"`
	} `json:"meta"`
	Config struct {
		Debug   bool   `json:"debug"`
		Timeout int    `json:"timeout"`
	} `json:"config"`
	Plain string `json:"plain"`
}

// WithAnonSlice 包含 []struct{...} 匿名 struct 切片字段。
// 典型场景：API 返回的嵌套列表，如云厂商 DescribeInstanceTypes 的选项列表。
type WithAnonSlice struct {
	ComputingArchitecture []struct {
		Text  string `json:"text"`
		Value string `json:"value"`
	} `json:"computingArchitecture"`
	CustomizedFamily []struct {
		Text  string `json:"text"`
		Value string `json:"value"`
	} `json:"customizeFamily"`
}

// WithAnonPtr 包含 *struct{...} 匿名 struct 指针字段。
type WithAnonPtr struct {
	Header *struct {
		RequestID string `json:"request_id"`
		TraceID   string `json:"trace_id"`
	} `json:"header"`
	Body string `json:"body"`
}

// WithNestedAnon 包含嵌套匿名 struct（匿名 struct 内再含匿名 struct 切片）。
type WithNestedAnon struct {
	Result struct {
		Items []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
		Total int `json:"total"`
	} `json:"result"`
}

// ─────────────────────────────────────────────
// 7. interface{} / any 字段
// ─────────────────────────────────────────────

// WithInterface 包含接口类型字段。
type WithInterface struct {
	Data      interface{}            `json:"data"`
	AnyField  any                    `json:"any_field"`
	MapOfAny  map[string]interface{} `json:"map_of_any"`
	SliceAny  []any                  `json:"slice_any"`
}

// ─────────────────────────────────────────────
// 8. struct tag 覆盖
// ─────────────────────────────────────────────

// TagVariants 覆盖常见 struct tag 格式。
type TagVariants struct {
	// json tag：omitempty、-、字段重命名
	Exported    string `json:"exported"`
	OmitEmpty   string `json:"omit_empty,omitempty"`
	Ignored     string `json:"-"`
	Inline      string `json:",inline"`

	// yaml tag
	YAMLField string `yaml:"yaml_field"`
	YAMLOmit  string `yaml:"yaml_omit,omitempty"`

	// xml tag
	XMLAttr   string `xml:"xml_attr,attr"`
	XMLCDATA  string `xml:"xml_cdata,cdata"`

	// validate tag（常见校验场景）
	Required  string `json:"required" validate:"required"`
	MinMax    int    `json:"min_max" validate:"min=1,max=100"`
	Email     string `json:"email" validate:"email"`

	// db / gorm tag
	PrimaryKey int64  `json:"pk" db:"id" gorm:"primaryKey"`

	// 多 tag 组合
	Multi string `json:"multi" yaml:"multi" xml:"multi" validate:"required,min=1"`

	// 无 tag
	NoTag string
}

// ─────────────────────────────────────────────
// 9. 外部包类型引用
// ─────────────────────────────────────────────

// ExternalTypes 引用标准库及外部包类型。
type ExternalTypes struct {
	T       time.Time      `json:"t"`
	Dur     time.Duration  `json:"dur"`
	URL     url.URL        `json:"url"`
	PURL    *url.URL       `json:"p_url"`
	Ctx     context.Context
	Raw     alias.RawMessage `json:"raw"` // 显式别名 import
}

// ─────────────────────────────────────────────
// 10. 类型别名（type Foo = Bar）
// ─────────────────────────────────────────────

type StringAlias = string
type IntAlias = int
type TimeAlias = time.Time
type PrimitivesAlias = Primitives

// ─────────────────────────────────────────────
// 11. 新类型定义（type Foo Bar，非别名）
// ─────────────────────────────────────────────

type UserID int64
type Score float64
type Label string
type Flags uint32

// ─────────────────────────────────────────────
// 12. 枚举（string / int 常量 + iota）
// ─────────────────────────────────────────────

// Status 是一个字符串枚举。
type Status string

const (
	StatusActive   Status = "active"   // 活跃
	StatusInactive Status = "inactive" // 未激活
	StatusDeleted  Status = "deleted"  // 已删除
)

// Direction 是一个 int 枚举（iota）。
type Direction int

const (
	DirectionNorth Direction = iota // 北
	DirectionSouth                  // 南
	DirectionEast                   // 东
	DirectionWest                   // 西
)

// Priority 混合 iota 位移运算。
type Priority int

const (
	PriorityLow      Priority = 1 << iota // 低优先级
	PriorityMedium                        // 中优先级
	PriorityHigh                          // 高优先级
	PriorityCritical                      // 紧急
)

// ─────────────────────────────────────────────
// 13. 泛型 struct（Go 1.18+）
// ─────────────────────────────────────────────

// Pair 泛型二元组，单类型参数。
type Pair[T any] struct {
	First  T `json:"first"`
	Second T `json:"second"`
}

// KV 泛型键值对，两个类型参数。
type KV[K comparable, V any] struct {
	Key   K `json:"key"`
	Value V `json:"value"`
}

// Page 泛型分页结果，约束为接口。
type Page[T any] struct {
	Items    []T  `json:"items"`
	Total    int  `json:"total"`
	PageNo   int  `json:"page_no"`
	PageSize int  `json:"page_size"`
}

// Numeric 泛型，约束为联合类型。
type Numeric[T int | int32 | int64 | float32 | float64] struct {
	Value T `json:"value"`
	Min   T `json:"min"`
	Max   T `json:"max"`
}

// ─────────────────────────────────────────────
// 14. 泛型实例化字段
// ─────────────────────────────────────────────

// GrayList 泛型灰度名单，约束为 cmp.Ordered（外部包约束）。
type GrayList[T cmp.Ordered] struct {
	Blacklist []T `json:"blacklist"`
	Whitelist []T `json:"whitelist"`
}

// InitVersionGrayScaleConfig 版本灰度配置，使用 GrayList 泛型实例化。
type InitVersionGrayScaleConfig struct {
	UserAlias GrayList[string] `json:"user_alias"`
	DepartID  GrayList[int]    `json:"depart_id"`
}

// GrayscaleConf 顶层灰度配置，嵌套 InitVersionGrayScaleConfig。
type GrayscaleConf struct {
	InitVersionConf InitVersionGrayScaleConfig `json:"init_version_conf"`
}

// GenericUsage 在 struct 字段中使用泛型实例化类型。
type GenericUsage struct {
	StringPair  Pair[string]         `json:"string_pair"`
	IntPage     Page[int]            `json:"int_page"`
	StrKV       KV[string, int]      `json:"str_kv"`
	UserPage    Page[BaseModel]      `json:"user_page"`
}

// ─────────────────────────────────────────────
// 15. 递归 / 自引用类型
// ─────────────────────────────────────────────

// TreeNode 单叉递归。
type TreeNode struct {
	Value    int       `json:"value"`
	Children []*TreeNode `json:"children,omitempty"`
	Parent   *TreeNode  `json:"parent,omitempty"`
}

// LinkedList 链表节点。
type LinkedList struct {
	Data int
	Next *LinkedList
	Prev *LinkedList
}

// ─────────────────────────────────────────────
// 16. 多名字段声明（共享类型）
// ─────────────────────────────────────────────

// MultiName 一行声明多个同类型字段。
type MultiName struct {
	X, Y, Z  float64
	Min, Max int
	A, B     string `json:"a_b"`
}

// ─────────────────────────────────────────────
// 17. func / chan 字段（parser 应识别并跳过/标记）
// ─────────────────────────────────────────────

// WithFuncChan 含函数和 chan 字段。
type WithFuncChan struct {
	Handler    func(ctx context.Context) error
	Middleware func(string) string
	Done       chan struct{}
	Events     chan<- string // 只写 chan
	Results    <-chan int    // 只读 chan
}

// ─────────────────────────────────────────────
// 18. 组合复杂场景（综合嵌套）
// ─────────────────────────────────────────────

// Address 地址值对象。
type Address struct {
	Street  string `json:"street" validate:"required"`
	City    string `json:"city"   validate:"required"`
	Country string `json:"country"`
	Zip     string `json:"zip"    validate:"len=6"`
}

// ContactInfo 联系信息。
type ContactInfo struct {
	Email   string  `json:"email"   validate:"email"`
	Phone   string  `json:"phone"`
	Address Address `json:"address"`
}

// User 综合使用多种类型场景的 struct。
type User struct {
	BaseModel
	Username  string      `json:"username"  validate:"required,min=3,max=32"`
	Email     string      `json:"email"     validate:"required,email"`
	Status    Status      `json:"status"`
	Score     Score       `json:"score"`
	Tags      []string    `json:"tags,omitempty"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
	Contact   *ContactInfo           `json:"contact,omitempty"`
	Friends   []*User     `json:"friends,omitempty"` // 自引用切片
}

// APIResponse 泛型 API 响应封装。
type APIResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
	Data    T      `json:"data"`
	TraceID string `json:"trace_id"`
}

// ─────────────────────────────────────────────
// 19. 函数声明（各类参数 / 返回值场景）
// ─────────────────────────────────────────────

// 普通函数
func Plain() {}

// 多参数多返回值
func MultiReturn(a int, b string) (int, error) { return 0, nil }

// 具名返回值
func NamedReturn(n int) (result string, err error) { return "", nil }

// 变参函数
func Variadic(prefix string, values ...int) []int { return nil }

// 泛型函数（Go 1.18+）
func Map[T, U any](slice []T, fn func(T) U) []U { return nil }

// 方法（值接收者）
func (u User) DisplayName() string { return u.Username }

// 方法（指针接收者）
func (u *User) SetStatus(s Status) { u.Status = s }

// 接受外部类型的函数
func ParseTime(s string) (time.Time, error) { return time.Time{}, nil }

// 返回泛型类型的函数
func PageOf[T any](items []T, total int) Page[T] {
	return Page[T]{Items: items, Total: total}
}

// ─────────────────────────────────────────────
// 20. 局部 struct（函数内定义）
// ─────────────────────────────────────────────

// FuncWithLocalStruct 函数内含局部 struct 定义。
func FuncWithLocalStruct() {
	type LocalReq struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	type LocalResp struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	_ = LocalReq{}
	_ = LocalResp{}
}

// ─────────────────────────────────────────────
// 21. 局部非 struct 类型定义（函数内 type Foo Bar）
// ─────────────────────────────────────────────

// FuncWithLocalTypeDef 函数内含局部非 struct 类型定义。
// 模拟 type Response config.SomeStruct 场景（跨包类型别名）。
func FuncWithLocalTypeDef() {
	type View BaseModel    // 新类型定义（非别名、非 struct 字面量）
	type ViewList []string // 复合底层类型
	_ = View{}
	_ = ViewList(nil)
}

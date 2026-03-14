# Go 代码规范

> 适用于本仓库（`github.com/go-openapi/spec3`）及通用 Go 项目。

---

## 1. 错误处理

**规则**：错误必须被检查，不得静默忽略；错误信息用小写、不加标点。

```go
// ✅ 正确
data, err := json.Marshal(v)
if err != nil {
    return fmt.Errorf("marshal schema: %w", err)
}

// ❌ 错误 — 忽略 error
data, _ = json.Marshal(v)

// ❌ 错误 — 错误信息首字母大写
return fmt.Errorf("Marshal failed: %w", err)
```

---

## 2. 接收者类型（Receiver）

**规则**：含可变状态或较大结构体用指针接收者；纯只读且结构体小（≤4字段）可用值接收者；同一类型的所有方法必须统一。

```go
// ✅ 正确 — 结构体有状态，统一用指针接收者
func (s *OrderedMap) Set(key string, val interface{}) bool { ... }
func (s *OrderedMap) Get(key string) interface{} { ... }

// ❌ 错误 — 同一类型混用指针和值接收者
func (s *OrderedMap) Set(...) bool { ... }
func (s OrderedMap) Len() int { ... }  // 应为 *OrderedMap
```

---

## 3. 命名

**规则**：导出名用 `MixedCaps`，缩写词全大写（`URL`、`ID`、`HTTP`），包名小写单数。

```go
// ✅ 正确
type OrderedMap struct { ... }
func (s *Schema) MarshalJSON() ([]byte, error) { ... }
var userID int

// ❌ 错误
type ordered_map struct { ... }    // 下划线
func (s *Schema) Marshaljson() {} // 缩写词未大写
var userId int                    // Id 不是惯用写法
```

---

## 4. MarshalJSON 实现

**规则**：自定义 `MarshalJSON` 应优先处理 `$ref`，使用 `wire` struct 解耦内部字段，最后合并 vendor extensions。

```go
// ✅ 正确
func (p *Parameter) MarshalJSON() ([]byte, error) {
    if ref := p.Ref.String(); ref != "" {
        return json.Marshal(map[string]string{"$ref": ref})
    }
    type wire struct {
        Name string `json:"name,omitempty"`
        In   string `json:"in,omitempty"`
    }
    // ... 填充 wire，marshal，合并 extensions
}

// ❌ 错误 — 直接 marshal 含嵌入类型的原始 struct，
//            导致 VendorExtensible.MarshalJSON 覆盖所有字段
func (p *Parameter) MarshalJSON() ([]byte, error) {
    return json.Marshal(*p)
}
```

---

## 5. interface{} 类型断言

**规则**：使用 comma-ok 形式，避免 panic。

```go
// ✅ 正确
if m, ok := value.(json.Marshaler); ok {
    out.Raw(m.MarshalJSON())
}

// ❌ 错误 — 无 ok 检查，value 不实现接口时 panic
out.Raw(value.(json.Marshaler).MarshalJSON())
```

---

## 6. 切片操作

**规则**：删除元素后应 `break`（唯一键场景）；避免在迭代中修改切片后继续依赖索引。

```go
// ✅ 正确
for i, k := range s.keys {
    if k == target {
        s.keys = append(s.keys[:i], s.keys[i+1:]...)
        break
    }
}

// ❌ 错误 — 删除后未 break，继续访问已变更的切片
for i, k := range s.keys {
    if k == target {
        s.keys = append(s.keys[:i], s.keys[i+1:]...)
        // 继续循环，逻辑错误
    }
}
```

---

## 7. nil 安全

**规则**：懒初始化不应有副作用（不应在读操作中写回字段）；零值结构体应安全可用。

```go
// ✅ 正确 — 读操作无副作用
func (s *OrderedMap) normalizeKey(key string) string {
    if s.normalize != nil {
        return s.normalize(key)
    }
    return key
}

// ❌ 错误 — 读操作中写回字段，对值接收者拷贝无效，语义混乱
func (s *OrderedMap) normalizeKey(key string) string {
    if s.normalize == nil {
        s.normalize = NOPNormalizer  // 副作用
    }
    return s.normalize(key)
}
```

---

## 8. `omitempty` 使用

**规则**：可选字段必须加 `omitempty`；`bool`/`int` 等零值有语义区分时不加。

```go
// ✅ 正确
type Schema struct {
    Title    string  `json:"title,omitempty"`
    Required bool    `json:"required"`          // false 有语义
    MaxItems *int64  `json:"maxItems,omitempty"` // 指针，omitempty 生效
}

// ❌ 错误
type Schema struct {
    Title string `json:"title"` // 空字符串会序列化为 ""
}
```

---

## 扩展此文件

在对应规则节末尾追加新条目，保持编号连续。示例：

```
## 9. 新规则名称
**规则**：...
```

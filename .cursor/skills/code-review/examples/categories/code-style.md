# 🟠 [代码规范]

**触发条件**：不符合语言惯用写法、项目约定或命名规范，但不影响正确性。

---

```go
// ❌ 触发 [代码规范] — 错误信息首字母大写，违反 Go 惯例
return fmt.Errorf("Marshal failed: %w", err)

// ❌ 触发 [代码规范] — 使用下划线命名
type ordered_map struct { ... }

// ❌ 触发 [代码规范] — Id 应为 ID
var userId int
```

**review 评论**：
> 🟠 **[代码规范]** Go 惯例要求错误信息全小写且不加标点，应改为 `"marshal failed: %w"`。参见 [standards/go.md § 错误处理](../../standards/go.md)。· `schema.go:42`

```go
// ✅ 修复后
return fmt.Errorf("marshal failed: %w", err)
var userID int
```

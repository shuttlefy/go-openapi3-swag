# 🔷 [接口/控件使用]

**触发条件**：标准库、第三方库或项目内 API 的使用方式不当或不符合其设计意图。

---

```go
// ❌ 触发 [接口/控件使用] — 直接 marshal 含嵌入类型的 struct，
//    导致 VendorExtensible.MarshalJSON 覆盖其他字段
func (r *RequestBody) MarshalJSON() ([]byte, error) {
    return json.Marshal(*r)  // VendorExtensible 的 MarshalJSON 被调用，丢失 Description/Content
}

// ❌ 触发 [接口/控件使用] — 值接收者实现 json.Marshaler，
//    encoding/json 对指针类型不会自动调用值接收者方法
func (s Schema) MarshalJSON() ([]byte, error) { ... }
```

**review 评论**：
> 🔷 **[接口/控件使用]** `encoding/json` 遇到嵌入了 `VendorExtensible`（自带 `MarshalJSON`）的 struct 时，会优先调用嵌入类型的方法，导致外层字段丢失。应使用 `wire` struct 模式显式控制序列化。参见 [standards/go.md § MarshalJSON 实现](../../standards/go.md)。· `request_body.go:20`

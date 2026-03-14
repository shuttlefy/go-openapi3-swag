# 🟡 [代码设计]

**触发条件**：代码能运行，但结构、职责或抽象层次有改进空间。

---

```go
// ❌ 触发 [代码设计] — MarshalJSON 同时做 $ref 判断、字段填充、extension 合并，职责过多
func (p *Parameter) MarshalJSON() ([]byte, error) {
    if ref := p.Ref.String(); ref != "" {
        return json.Marshal(map[string]string{"$ref": ref})
    }
    // 50 行内联逻辑...
    var m map[string]interface{}
    json.Unmarshal(b, &m)
    // 合并 extension...
    return json.Marshal(m)
}
```

**review 评论**：
> 🟡 **[代码设计]** extension 合并逻辑在多个 `MarshalJSON` 中重复出现，建议提取为 `mergeExtensions(base []byte, ext Extensions) ([]byte, error)` 工具函数。· `parameter.go:80`

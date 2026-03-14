# 💬 [疑问]

**触发条件**：代码意图不明确，需要作者解释，不确定是否存在问题。

---

```go
// 触发 [疑问] — NilMapAsEmpty flag 的判断意图不清晰
func encodeSortedMap(out *jwriter.Writer, in OrderedMap) {
    if in.data == nil && (out.Flags&jwriter.NilMapAsEmpty) == 0 {
        out.RawString(`null`)
        return
    }
    // ...
}
```

**review 评论**：
> 💬 **[疑问]** 当 `NilMapAsEmpty` 未设置时输出 `null`，这是期望行为吗？`OrderedMap` 在多数场景作为对象字段，输出 `null` 可能导致 JSON 解析方出现空指针。是否应该统一输出 `{}`？· `orderedmap.go:219`

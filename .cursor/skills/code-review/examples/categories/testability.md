# 🩶 [可测性]

**触发条件**：代码逻辑正确，但结构导致难以编写单元测试，或关键路径缺少测试覆盖。

---

```go
// ❌ 触发 [可测性] — normalizeKey 是私有方法且有副作用，无法从外部直接测试
func (s *OrderedMap) normalizeKey(key string) string {
    if s.normalize == nil {
        s.normalize = NOPNormalizer  // 副作用：只能通过 Set/Get 间接观测
    }
    return s.normalize(key)
}
```

**review 评论**：
> 🩶 **[可测性]** `normalizeKey` 的副作用（写回 `s.normalize`）只能通过 `Set`/`Get` 间接测试，建议去除副作用，使行为通过公开方法完整可观测。· `orderedmap.go:75`

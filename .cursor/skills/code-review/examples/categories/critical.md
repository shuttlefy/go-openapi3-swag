# 🔴 [严重]

**触发条件**：影响正确性、数据安全、程序崩溃，必须在合并前修复。

---

```go
// ❌ 触发 [严重] — Delete 未检查 key 是否存在，始终返回 true
func (s *OrderedMap) Delete(k string) bool {
    key := s.normalizeKey(k)
    delete(s.data, key)   // key 不存在时也执行，无副作用但返回值错误
    return true           // 调用方无法判断是否真正删除
}
```

**review 评论**：
> 🔴 **[严重]** `Delete` 未检查 key 是否存在即返回 `true`，调用方无法感知删除失败。应先检查 `s.data[key]` 是否存在，不存在时返回 `false`。· `orderedmap.go:109`

```go
// ✅ 修复后
func (s *OrderedMap) Delete(k string) bool {
    key := s.normalizeKey(k)
    if _, ok := s.data[key]; !ok {
        return false
    }
    delete(s.data, key)
    // ...
    return true
}
```

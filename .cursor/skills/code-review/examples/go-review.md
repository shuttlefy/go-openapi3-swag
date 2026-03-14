# 示例：Go 代码 Review

> 展示完整的 review 输出格式，供参考。

---

**被审查代码**（`orderedmap.go` 旧版本片段）：

```go
func (s *OrderedMap) normalizeKey(key string) string {
    if s.normalize == nil {
        s.normalize = NOPNormalizer  // 副作用写回
    }
    return s.normalize(key)
}

func (s *OrderedMap) Delete(k string) bool {
    key := s.normalizeKey(k)
    delete(s.data, key)
    for i, k := range s.keys {
        if k == key {
            s.keys = append(s.keys[:i], s.keys[i+1:]...)
            // 没有 break
        }
    }
    return true  // 未检查 key 是否存在
}

func encodeSortedMap(out *jwriter.Writer, in OrderedMap) {
    out.RawByte('{')
    first := true
    for _, k := range in.keys {
        _ = first     // 死代码
        if !first { out.RawByte(',') }
        first = false
        ...
    }
}
```

---

## 总结

整体逻辑清晰，但存在三处影响正确性的问题：`Delete` 缺少存在性检查和 `break`，`normalizeKey` 在读操作中有副作用写回。

## 问题列表

- 🔴 **[严重]** `Delete` 未检查 key 是否存在就调用 `delete()`，始终返回 `true`，调用方无法判断删除是否成功。 · `orderedmap.go:109`

- 🔴 **[严重]** `Delete` 循环删除 key 后未 `break`；由于 `keys` 已缩短，继续迭代会跳过相邻元素（逻辑错误）。 · `orderedmap.go:121`

- 🟡 **[警告]** `normalizeKey` 在读操作中写回 `s.normalize`，对值接收者拷贝无效，对指针接收者是副作用，破坏"读操作不改变状态"原则。 · `orderedmap.go:75`

- 🔵 **[建议]** `encodeSortedMap` 中 `_ = first` 是死代码，可用下标 `if i > 0` 替代 `first` 标志，逻辑更清晰。 · `orderedmap.go:227`

## 亮点

- `Set` 的懒初始化 `data` map 写法简洁，零值可用设计良好。
- `encodeSortedMap` 正确处理了 `NilMapAsEmpty` flag，兼容 easyjson 的标志位语义。

## 结论

**需要修改后通过** — 严重问题修复后可合并。

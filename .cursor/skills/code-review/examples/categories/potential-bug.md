# 🟣 [潜在缺陷]

**触发条件**：当前测试可能通过，但在特定边界条件、并发或异常输入下会出错。

---

```go
// ❌ 触发 [潜在缺陷] — 删除后未 break，修改切片后继续迭代
for i, k := range s.keys {
    if k == key {
        s.keys = append(s.keys[:i], s.keys[i+1:]...)
        // 未 break：keys 已缩短，下次迭代可能跳过紧邻元素
    }
}
```

**review 评论**：
> 🟣 **[潜在缺陷]** 删除元素后继续 range，由于切片已缩短，后续迭代会跳过紧邻元素。`keys` 中每个 key 唯一，找到后应立即 `break`。· `orderedmap.go:121`

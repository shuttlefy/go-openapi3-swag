# 🟢 [不错的代码]

**触发条件**：写法简洁、符合惯例、设计良好，值得保留和推广。

---

```go
// ✅ 触发 [不错的代码] — 零值可用，懒初始化 map，无需构造函数
func (s *OrderedMap) Set(key string, value interface{}) bool {
    if s.data == nil {
        s.data = make(map[string]interface{})
    }
    // ...
}
```

**review 评论**：
> 🟢 **[不错的代码]** `Set` 的懒初始化设计使 `OrderedMap` 零值即可用，避免调用方必须显式构造，符合 Go 惯例。· `orderedmap.go:91`

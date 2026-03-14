# ⚫ [其他]

**触发条件**：不属于上述任何分类的观察，如 TODO 遗留、文档缺失、文件组织问题。

---

```go
// 触发 [其他] — TODO 遗留超过 6 个月未处理
// TODO: (s *OrderedStrings) Implement Marshal & Unmarshal -> JSON, YAML

// 触发 [其他] — 魔法字符串无说明
if strings.HasPrefix(key, "x-") { ... }  // "x-" 含义未通过常量或注释说明
```

**review 评论**：
> ⚫ **[其他]** `orderedmap.go` 末尾的 TODO 已存在较长时间，建议转为 GitHub Issue 追踪，或在本次 PR 中实现。· `orderedmap.go:326`

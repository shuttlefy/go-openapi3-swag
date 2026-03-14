# 🔵 [建议]

**触发条件**：代码正确，但有可选的可读性或性能改进。

---

```go
// 可改进 — 用 first bool 标志控制逗号
first := true
for _, k := range in.keys {
    _ = first
    if !first { out.RawByte(',') }
    first = false
    // ...
}
```

**review 评论**：
> 🔵 **[建议]** 用下标 `if i > 0` 替代 `first` 标志，逻辑更直接，并可删除无意义的 `_ = first`。· `orderedmap.go:227`

```go
// ✅ 改进后
for i, k := range in.keys {
    if i > 0 { out.RawByte(',') }
    // ...
}
```

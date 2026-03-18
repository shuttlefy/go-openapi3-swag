# output 包（Stage 4）

**代码位置**：`internal/output/writer.go`

## 职责

将 `*spec3.OpenAPI` 序列化为 JSON 或 YAML 文件。职责极度精简：格式选择 + 目录创建 + 文件写入。实际序列化由 `spec3.MarshalJSON` / `spec3.MarshalYAML` 完成。

## 函数

```go
func Write(doc *spec3.OpenAPI, path string) error
```

| 行为 | 说明 |
|------|------|
| 格式选择 | 根据 `path` 扩展名决定：`.yaml` → YAML，其余（含 `.json`）→ JSON |
| 自动建目录 | 若输出目录不存在，自动 `MkdirAll` |
| 缩进 | JSON 输出使用 2 空格缩进（`json.MarshalIndent`） |
| 序列化 | JSON：`encoding/json.MarshalIndent`；YAML：`spec3.MarshalYAML` |

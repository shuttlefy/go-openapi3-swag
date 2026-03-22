# CLI 入口

**代码位置**：`cmd/swag3/main.go`、`cmd/swag3/e2e_test.go`

## 用法

```bash
swag3 -dirs <目录> [选项]
```

### 参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-dirs` | 字符串（逗号分隔 / 可重复） | 必填 | 要扫描的源码目录 |
| `-output` | 字符串 | `./docs/openapi.json` | 输出文件路径；扩展名决定格式（`.json` / `.yaml`） |
| `-openapi-ver` | 字符串 | `3.2.0` | 输出的 OpenAPI 版本号（支持 3.0.3 / 3.1.0 / 3.2.0） |
| `-depth` | 整数 | `-1` | 目录递归深度：`-1`=无限，`0`=仅当前，`N`=最多 N 层 |
| `-gomod` | 字符串 | `go.mod` | go.mod 路径，用于模块缓存懒加载 |
| `-query-struct-explode` | 布尔 | `false` | `query` 注解类型为 struct 时自动展开字段（打散） |

### 示例

```bash
# 基础用法：扫描当前目录，输出 JSON
swag3 -dirs .

# 多目录
swag3 -dirs ./cmd,./internal -output ./docs/openapi.json

# 仅扫描指定目录（不递归子目录）
swag3 -dirs ./api -depth 0

# 输出 YAML
swag3 -dirs . -output ./docs/openapi.yaml

# 指定 OpenAPI 版本
swag3 -dirs . -openapi-ver 3.1.0
```

## 流水线组装逻辑

`main.go` 是纯粹的组装层，不含业务逻辑，只负责按顺序调用各阶段并处理错误：

```go
// 1. 解析命令行参数
cfg := config.ParseFlags()

// 2. Stage 1: 解析 Go 源文件
p := &parser.GoParser{MaxDepth: cfg.ParseDepth}
files, err := p.Parse(cfg.InputDirs)

// 3. Stage 2: 提取注解
e := &extractor.GoExtractor{}
result, err := e.Extract(files)

// 4. Stage 3: 构建 OpenAPI 文档（始终尝试启用懒加载）
b := builder.NewBuilder()
if modInfo, err := parser.ParseGoMod(cfg.GoMod); err == nil {
    b.SetLoader(builder.NewModuleLoader(modInfo, parser.ModuleCacheDir()))
}
doc, err := b.Build(result, files)

// 5. Stage 4: 写入文件
output.Write(doc, cfg.OutputFile)
```

go.mod 解析失败时，懒加载自动降级（禁用），不影响主流程。

## 端到端测试

`e2e_test.go` 以 `testdata/annotations/` 作为输入，运行完整流水线，将生成结果与预期 golden file 对比。

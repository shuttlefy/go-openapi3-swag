# config 包

**代码位置**：`config/config.go`

## 职责

解析 CLI flags，提供统一的配置结构体供各阶段使用。

## 类型

### `Config`

```go
type Config struct {
    InputDirs  StringSlice // -dirs：扫描目录列表
    OutputFile string      // -output：输出文件路径
    OpenAPIVer string      // -openapi-ver：OpenAPI 版本号
    ParseDepth int         // -depth：目录递归深度
    GoMod      string      // -gomod：go.mod 路径
}
```

### `StringSlice`

实现 `flag.Value` 接口，支持两种传入方式：

```bash
# 逗号分隔
swag3 -dirs ./cmd,./internal

# 多次传入
swag3 -dirs ./cmd -dirs ./internal
```

## 函数

### `ParseFlags() *Config`

解析命令行参数并填充默认值：

| 字段 | 默认值 |
|------|--------|
| `OutputFile` | `./docs/openapi.json` |
| `OpenAPIVer` | `3.2.0` |
| `ParseDepth` | `-1`（无限递归） |
| `GoMod` | `go.mod` |

`Title` 和 `Version` 不作为 CLI 参数，由源码注解（`@title` / `@version`）决定。

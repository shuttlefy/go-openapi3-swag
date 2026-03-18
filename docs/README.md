# swag3 文档索引

> Go 源码 → OpenAPI 3.x 规范生成器

## 文档目录

| 文档 | 对应代码 | 说明 |
|------|---------|------|
| [architecture.md](architecture.md) | 全局 | 整体架构、数据流、设计原则 |
| [cli.md](cli.md) | `cmd/swag3/` | CLI 入口与流水线组装 |
| [config.md](config.md) | `config/` | 命令行参数与配置结构 |
| [parser.md](parser.md) | `internal/parser/` | Stage 1：Go AST 解析 |
| [extractor.md](extractor.md) | `internal/extractor/` | Stage 2：注释注解提取 |
| [builder.md](builder.md) | `internal/builder/` | Stage 3：OpenAPI 文档构建 |
| [output.md](output.md) | `internal/output/` | Stage 4：JSON/YAML 序列化输出 |
| [swaggin.md](swaggin.md) | `pkg/swaggin/` | Gin 框架集成（公开 API） |

## 相关文件

| 文件 | 说明 |
|------|------|
| [`../annotations.md`](../annotations.md) | 注解标签完整参考手册（面向用户） |
| [`../design.md`](../design.md) | 架构设计决策记录（详细版） |

## 快速导航

- **使用 swag3**：查看 [cli.md](cli.md) 了解命令行用法
- **编写注解**：查看 [`../annotations.md`](../annotations.md)
- **理解架构**：查看 [architecture.md](architecture.md)
- **扩展构建逻辑**：查看 [builder.md](builder.md)
- **集成到 Gin**：查看 [swaggin.md](swaggin.md)

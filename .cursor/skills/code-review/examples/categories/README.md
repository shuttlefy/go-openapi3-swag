# 分类示例索引

每个分类独立一个文件，含触发条件、❌ 反例、review 评论写法、✅ 修复示例（适用时）。

| 文件 | 分类 | 触发条件摘要 |
|------|------|------------|
| [critical.md](critical.md) | 🔴 严重 | 影响正确性或安全性，必须修复 |
| [code-style.md](code-style.md) | 🟠 代码规范 | 不符合语言惯用写法或项目约定 |
| [design.md](design.md) | 🟡 代码设计 | 结构、职责或抽象层次有改进空间 |
| [good-code.md](good-code.md) | 🟢 不错的代码 | 值得肯定的写法，无需修改 |
| [suggestion.md](suggestion.md) | 🔵 建议 | 可选的可读性或性能改进 |
| [potential-bug.md](potential-bug.md) | 🟣 潜在缺陷 | 边界条件或异常路径存在隐患 |
| [other.md](other.md) | ⚫ 其他 | TODO 遗留、文档缺失、文件组织问题 |
| [testability.md](testability.md) | 🩶 可测性 | 结构导致难以测试或缺少覆盖 |
| [api-usage.md](api-usage.md) | 🔷 接口/控件使用 | API 或组件使用方式不当 |
| [question.md](question.md) | 💬 疑问 | 意图不明，需要作者澄清 |

## 新增分类

1. 在此目录下创建 `<name>.md`，参照现有文件格式
2. 在上表末尾追加一行
3. 在 `../../SKILL.md` 的问题列表中追加对应的 emoji + 标签行

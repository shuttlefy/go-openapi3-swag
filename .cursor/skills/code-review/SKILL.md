---
name: code-review
description: Reviews code for quality, bugs, security, and performance. Use when
  the user asks for code review, PR review, 审查代码, review 代码, 帮我看看代码,
  检查代码, 代码质量分析, or wants feedback on any code changes or files.
argument-hint: "[文件路径 | PR 编号 | 留空则审查 git diff]"
---

# Code Review

对 $ARGUMENTS 进行审查。若未提供参数，审查当前工作区的 `git diff`。

## 审查流程

1. **识别语言** → 读取对应规范文件（见下方语言支持）
2. **按规范逐项检查** → 标注严重程度
3. **按输出格式输出结果**

## 语言支持

| 语言 | 规范文件 | 状态 |
|------|---------|------|
| Go   | [standards/go.md](standards/go.md) | ✅ |
| 通用 | [standards/general.md](standards/general.md) | ✅ |

> 新增语言：在 `standards/` 下创建 `<lang>.md`，参照 `go.md` 格式，并更新上表。

## 输出格式

### 总结
2-3 句话概括整体质量和主要发现。

### 问题列表

按以下类型标注，每条注明文件名和行号：

- 🔴 **[严重]** 影响正确性或安全性，必须修复 · `文件:行号`
- 🟠 **[代码规范]** 不符合语言惯用写法或项目约定 · `文件:行号`
- 🟡 **[代码设计]** 结构、抽象或职责划分有改进空间 · `文件:行号`
- 🟢 **[不错的代码]** 值得肯定的写法，无需修改 · `文件:行号`
- 🔵 **[建议]** 可选优化，提升可读性或性能 · `文件:行号`
- 🟣 **[潜在缺陷]** 边界条件或异常路径存在隐患 · `文件:行号`
- ⚫ **[其他]** 不属于以上类别的观察 · `文件:行号`
- 🩶 **[可测性]** 代码难以测试或缺少测试覆盖 · `文件:行号`
- 🔷 **[接口/控件使用]** API 或组件使用方式不当 · `文件:行号`
- 💬 **[疑问]** 需要作者澄清意图，不确定是否问题 · `文件:行号`

### 亮点
代码中值得肯定的地方。

### 结论
**通过** / **需要修改后通过** / **需要重大修改**

---

完整示例参见：
- [examples/go-review.md](examples/go-review.md) — 完整 review 输出示例
- [examples/categories/](examples/categories/README.md) — 每个分类独立一个文件，含触发条件、代码示例与评论写法

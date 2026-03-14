---
name: create-skill
description: 创建新的 skill（技能插件）。当用户要求增加/创建新技能、新插件、new skill 时触发。
argument-hint: "<skill-name> [简短描述]"
---

# 创建新 Skill

## 目标

根据用户需求，在 `.cursor/skills/<name>/SKILL.md` 创建一个新技能，然后运行同步脚本将其传播到 `.claude/skills/`。

## 步骤

### 1. 确认技能名称与用途

若 `$ARGUMENTS` 为空，询问用户：
- 技能名称（kebab-case，如 `go-test`）
- 触发场景：什么情况下应该使用这个技能？
- 核心功能：这个技能要做什么？

若 `$ARGUMENTS` 已提供，从中解析名称和描述。

### 2. 创建 SKILL.md

在 `.cursor/skills/<name>/SKILL.md` 按以下模板创建：

```markdown
---
name: <skill-name>
description: <一句话描述，说明何时触发>
argument-hint: "<参数提示，如有>"
---

# <技能标题>

<技能的主体 prompt，直接描述 AI 应执行的操作>

## 使用说明（可选）

...
```

**要求：**
- `description` 字段必须清楚说明**何时触发**，方便 AI 自动选择正确技能
- `argument-hint` 仅在技能接受参数时填写
- SKILL.md 正文是直接执行的 prompt，不要写"你是一个助手"之类的角色声明
- 内容简洁，只包含完成任务所需的信息

### 3. 同步到 Claude

```bash
bash .cursor/skills/sync-skills/scripts/sync.sh
```

### 4. 确认

列出新创建的技能文件路径，告知用户技能已就绪，可通过 `/create-skill` 或对应触发词调用。

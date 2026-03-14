---
name: sync-skills
description: Keeps .cursor/skills/ and .claude/skills/ in sync so Cursor Agent and Claude CLI always share the same skill set. Use when creating a new skill, updating an existing skill, or when the user asks to sync skills between Cursor and Claude.
---

# Sync Skills

`.cursor/skills/` and `.claude/skills/` must always stay in sync.
Run the sync script after any skill create/update operation.

## When to apply

- After creating a new skill in either directory
- After modifying any `SKILL.md`
- When the user asks to sync, mirror, or align skills

## Sync workflow

```bash
bash .cursor/skills/sync-skills/scripts/sync.sh
```

The script copies skills bidirectionally:
- Skills only in `.cursor/skills/` → copied to `.claude/skills/`
- Skills only in `.claude/skills/` → copied to `.cursor/skills/`
- Newer file wins when both exist (compares modification time)

## Creating a new skill

1. Create the skill in `.cursor/skills/<name>/SKILL.md`
2. Run the sync script — it propagates to `.claude/skills/` automatically
3. No need to manually duplicate files

## SKILL.md compatibility

Both systems read the same SKILL.md format. The `argument-hint` field used
by Claude CLI is ignored by Cursor; Cursor's `name`/`description` frontmatter
is the authoritative metadata for both.

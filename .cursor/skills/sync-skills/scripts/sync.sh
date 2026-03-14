#!/usr/bin/env bash
# sync.sh — bidirectional sync between .cursor/skills/ and .claude/skills/
# Newer file wins. Run from the repository root.
set -euo pipefail

CURSOR_DIR=".cursor/skills"
CLAUDE_DIR=".claude/skills"

mkdir -p "$CURSOR_DIR" "$CLAUDE_DIR"

sync_skill() {
  local name="$1"
  local src="$2/$name/SKILL.md"
  local dst="$3/$name/SKILL.md"

  if [[ ! -f "$src" ]]; then
    return
  fi

  if [[ ! -f "$dst" ]]; then
    echo "  + $3/$name  (new)"
    mkdir -p "$3/$name"
    cp "$src" "$dst"
    return
  fi

  # Newer file wins
  if [[ "$src" -nt "$dst" ]]; then
    echo "  ~ $3/$name  (updated from $2)"
    cp "$src" "$dst"
  fi
}

echo "=== Syncing skills ==="

# Collect all skill names from both sides
all_names=()
for dir in "$CURSOR_DIR"/*/; do
  [[ -d "$dir" ]] && all_names+=("$(basename "$dir")")
done
for dir in "$CLAUDE_DIR"/*/; do
  [[ -d "$dir" ]] && all_names+=("$(basename "$dir")")
done

# Deduplicate
mapfile -t unique_names < <(printf '%s\n' "${all_names[@]}" | sort -u)

for name in "${unique_names[@]}"; do
  sync_skill "$name" "$CURSOR_DIR" "$CLAUDE_DIR"
  sync_skill "$name" "$CLAUDE_DIR" "$CURSOR_DIR"
done

echo "=== Done (${#unique_names[@]} skill(s) checked) ==="

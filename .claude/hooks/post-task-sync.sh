#!/usr/bin/env bash
# PostToolUse Hook: Sync TaskCreate/TaskUpdate/TaskDelete to Markdown files
# USMP Task Persistence System

set +e  # Hook failure must NOT block main flow

# Directory setup
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TASKS_DIR="$REPO_ROOT/openspec/tasks"

# Log function for debugging
log() {
    echo "[post-task-sync] $*" >> "$REPO_ROOT/.claude/hooks/post-task-sync.log" 2>/dev/null || true
}

# Check if tool is one we care about
if [[ "$TOOL_NAME" != "TaskCreate" && "$TOOL_NAME" != "TaskUpdate" && "$TOOL_NAME" != "TaskDelete" ]]; then
    log "Skipping: $TOOL_NAME"
    exit 0
fi

log "Processing: $TOOL_NAME"
log "TOOL_INPUT: $TOOL_INPUT"
log "TOOL_RESULT: $TOOL_RESULT"

# Get current git branch and worktree info
get_branch() {
    git -C "$REPO_ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown"
}

get_worktree() {
    git -C "$REPO_ROOT" rev-parse --git-dir 2>/dev/null | grep -q "worktrees" && echo "yes" || echo "no"
}

# Generate slug from subject (supports Chinese — transliterates common chars, falls back to timestamp)
generate_slug() {
    local subject="$1"
    local slug
    # Try transliteration via python3; if slug is empty (all non-ASCII), use timestamp
    slug=$(python3 -c "
import sys, re, unicodedata
text = sys.argv[1].strip().lower()
# Normalize unicode and remove combining marks for basic transliteration
text = unicodedata.normalize('NFKD', text)
text = re.sub(r'[̀-ͯ]', '', text)  # remove combining marks
text = re.sub(r'[^\w\s-]', '-', text)
text = re.sub(r'[\s_]+', '-', text)
text = re.sub(r'-+', '-', text)
text = text.strip('-')
if not text or set(text) == {'-'}:
    # Fallback: use timestamp-based slug
    import time
    text = f'task-{int(time.time())}'
print(text[:60])
" "$subject" 2>/dev/null)
    echo "${slug:-task-$(date +%s)}"
}

# Get current timestamp in ISO format
get_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

# Get current date for filename
get_date() {
    date -u +"%Y-%m-%d"
}

# Find task file by task ID
find_task_file() {
    local task_id="$1"
    grep -rl "id: $task_id" "$TASKS_DIR" --include="*.md" 2>/dev/null | head -1
}

# Extract JSON field using Python (no jq dependency)
extract_json_field() {
    local json="$1"
    local field="$2"
    python3 -c "import sys, json; d=json.loads(sys.argv[1]); print(d.get(sys.argv[2], ''))" "$json" "$field" 2>/dev/null
}

# Extract nested JSON field
extract_nested_field() {
    local json="$1"
    python3 -c "import sys, json; d=json.loads(sys.argv[1]); print(d.get(sys.argv[2], {}).get(sys.argv[3], ''))" "$json" "$2" "$3" 2>/dev/null
}

# Handle TaskCreate
handle_create() {
    log "Handling TaskCreate"

    # Extract taskId - try TOOL_RESULT first, then TOOL_INPUT
    local task_id=""
    if [[ -n "$TOOL_RESULT" ]]; then
        task_id=$(extract_json_field "$TOOL_RESULT" "taskId")
    fi
    if [[ -z "$task_id" ]]; then
        task_id=$(extract_json_field "$TOOL_INPUT" "taskId")
    fi

    # Extract fields from TOOL_INPUT
    local subject=$(extract_json_field "$TOOL_INPUT" "subject")
    local description=$(extract_json_field "$TOOL_INPUT" "description")
    local status=$(extract_json_field "$TOOL_INPUT" "status")
    local priority=$(extract_json_field "$TOOL_INPUT" "priority")

    # Defaults
    [[ -z "$status" ]] && status="pending"
    [[ -z "$priority" ]] && priority="medium"
    [[ -z "$task_id" ]] && { log "No taskId found"; return 1; }

    log "TaskCreate: id=$task_id, subject=$subject, status=$status, priority=$priority"

    # Generate filename
    local slug=$(generate_slug "$subject")
    local date=$(get_date)
    local filename="$date-$slug.md"
    local filepath="$TASKS_DIR/$filename"

    # Handle conflict
    local counter=1
    while [[ -f "$filepath" ]]; do
        filename="$date-$slug-$counter.md"
        filepath="$TASKS_DIR/$filename"
        counter=$((counter + 1))
    done

    # Get metadata
    local created=$(get_timestamp)
    local updated="$created"
    local branch=$(get_branch)
    local worktree=$(get_worktree)

    # Create the Markdown file
    cat > "$filepath" <<EOF
---
id: $task_id
title: $subject
status: $status
priority: $priority
assignee:
created: $created
updated: $updated
branch: $branch
worktree: $worktree
plan:
---

## 目标

$description

## 当前进度

- [ ] 待开始

## 上下文恢复提示

在此处记录关键上下文信息、决策点、临时状态等，帮助中断后恢复。

## 恢复指令

在此处列出恢复工作时需要执行的关键步骤或命令。

EOF

    log "Created: $filepath"
    echo "Created task file: $filepath"
}

# Handle TaskUpdate
handle_update() {
    log "Handling TaskUpdate"

    # Extract taskId from TOOL_INPUT
    local task_id=$(extract_json_field "$TOOL_INPUT" "taskId")
    [[ -z "$task_id" ]] && { log "No taskId found for update"; return 1; }

    # Find the file
    local filepath=$(find_task_file "$task_id")
    if [[ -z "$filepath" || ! -f "$filepath" ]]; then
        log "No file found for task $task_id, falling back to create"
        handle_create
        return $?
    fi

    log "Updating: $filepath"

    # Extract fields from TOOL_INPUT
    local status=$(extract_json_field "$TOOL_INPUT" "status")
    local title=$(extract_json_field "$TOOL_INPUT" "subject")
    local priority=$(extract_json_field "$TOOL_INPUT" "priority")
    local updated=$(get_timestamp)

    # Use Python to parse and update frontmatter properly
    python3 <<EOF
import yaml
import os
import re

filepath = "$filepath"
task_id = "$task_id"
status = "$status"
title = "$title"
priority = "$priority"
updated = "$updated"

# Read the file
with open(filepath, 'r', encoding='utf-8') as f:
    content = f.read()

# Split frontmatter and body
match = re.match(r'^---\n(.*?)\n---\n(.*)$', content, re.DOTALL)
if not match:
    print("Invalid format")
    exit(1)

fm_str, body = match.groups()
fm = yaml.safe_load(fm_str) or {}

# Update only non-empty fields
if status:
    fm['status'] = status
if title:
    fm['title'] = title
if priority:
    fm['priority'] = priority
fm['updated'] = updated

# Write back
with open(filepath, 'w', encoding='utf-8') as f:
    f.write('---\n')
    yaml.dump(fm, f, default_flow_style=False, sort_keys=False, allow_unicode=True)
    f.write('---\n')
    f.write(body)

print("Updated frontmatter")
EOF

    log "Updated: $filepath"
    echo "Updated task file: $filepath"
}

# Handle TaskDelete
handle_delete() {
    log "Handling TaskDelete"

    # Extract taskId from TOOL_INPUT
    local task_id=$(extract_json_field "$TOOL_INPUT" "taskId")
    [[ -z "$task_id" ]] && { log "No taskId found for delete"; return 1; }

    # Find the file
    local filepath=$(find_task_file "$task_id")
    if [[ -z "$filepath" || ! -f "$filepath" ]]; then
        log "No file found for task $task_id"
        return 0
    fi

    log "Marking as deleted: $filepath"
    local updated=$(get_timestamp)

    # Use Python to update status
    python3 <<EOF
import yaml
import os
import re

filepath = "$filepath"
updated = "$updated"

# Read the file
with open(filepath, 'r', encoding='utf-8') as f:
    content = f.read()

# Split frontmatter and body
match = re.match(r'^---\n(.*?)\n---\n(.*)$', content, re.DOTALL)
if not match:
    print("Invalid format")
    exit(1)

fm_str, body = match.groups()
fm = yaml.safe_load(fm_str) or {}

# Mark as deleted
fm['status'] = 'deleted'
fm['updated'] = updated

# Write back
with open(filepath, 'w', encoding='utf-8') as f:
    f.write('---\n')
    yaml.dump(fm, f, default_flow_style=False, sort_keys=False, allow_unicode=True)
    f.write('---\n')
    f.write(body)

print("Marked as deleted")
EOF

    log "Marked as deleted: $filepath"
    echo "Marked task as deleted: $filepath"
}

# Main logic
case "$TOOL_NAME" in
    TaskCreate)
        handle_create
        ;;
    TaskUpdate)
        handle_update
        ;;
    TaskDelete)
        handle_delete
        ;;
esac

exit 0

#!/bin/bash
# Command Generation Script for Claude Code Spec Workflow (Unix/Linux/macOS)
#
# This script generates individual task commands for each task in a spec's tasks.md file.
# It creates a folder structure under .claude/commands/{spec-name}/ with individual
# command files for each task that call /spec-execute with the appropriate parameters.
#
# Usage: ./generate-commands.sh <spec-name>

set -e

if [ -z "$1" ]; then
    echo "Error: Spec name is required"
    echo "Usage: ./generate-commands.sh <spec-name>"
    exit 1
fi

SPEC_NAME="$1"
PROJECT_ROOT="$(pwd)"
SPEC_DIR="$PROJECT_ROOT/.claude/specs/$SPEC_NAME"
TASKS_FILE="$SPEC_DIR/tasks.md"
COMMANDS_SPEC_DIR="$PROJECT_ROOT/.claude/commands/$SPEC_NAME"

# Check if tasks.md exists
if [ ! -f "$TASKS_FILE" ]; then
    echo "Error: tasks.md not found at $TASKS_FILE"
    exit 1
fi

# Create spec commands directory
mkdir -p "$COMMANDS_SPEC_DIR"

# Parse tasks and generate commands
TASK_COUNT=0
echo "Parsing tasks from $TASKS_FILE..."

generate_task_command() {
    local task_id="$1"
    local task_desc="$2"
    local leverage_info="$3"
    local requirements_info="$4"
    local command_file="$COMMANDS_SPEC_DIR/task-$task_id.md"

    cat > "$command_file" << EOF
# $SPEC_NAME - Task $task_id

Execute task $task_id for the $SPEC_NAME specification.

## Task Description
$task_desc

EOF

    # Add Leverage section if present
    if [ -n "$leverage_info" ]; then
        cat >> "$command_file" << EOF
## Code Reuse
**Leverage existing code**: $leverage_info

EOF
    fi

    # Add Requirements section if present
    if [ -n "$requirements_info" ]; then
        cat >> "$command_file" << EOF
## Requirements Reference
**Requirements**: $requirements_info

EOF
    fi

    cat >> "$command_file" << EOF
## Usage
```
/$SPEC_NAME-task-$task_id
```

## Instructions
This command executes a specific task from the $SPEC_NAME specification.

**Automatic Execution**: This command will automatically execute:
```
/spec-execute $task_id $SPEC_NAME
```

**Process**:
1. Load the $SPEC_NAME specification context (requirements.md, design.md, tasks.md)
2. Execute task $task_id: "$task_desc"
3. **Prioritize code reuse**: Use existing components and utilities identified above
4. Follow all implementation guidelines from the main /spec-execute command
5. Mark the task as complete in tasks.md
6. Stop and wait for user review

**Important**: This command follows the same rules as /spec-execute:
- Execute ONLY this specific task
- **Leverage existing code** whenever possible to avoid rebuilding functionality
- Mark task as complete by changing [ ] to [x] in tasks.md
- Stop after completion and wait for user approval
- Do not automatically proceed to the next task

## Next Steps
After task completion, you can:
- Review the implementation
- Run tests if applicable
- Execute the next task using /$SPEC_NAME-task-[next-id]
- Check overall progress with /spec-status $SPEC_NAME
EOF
}

# Parse tasks from markdown
current_task_id=""
current_task_desc=""
current_leverage=""
current_requirements=""
in_task=false

while IFS= read -r line; do
    # Match task lines like "- [ ] 1. Task description" or "- [ ] 2.1 Task description"
    if [[ $line =~ ^[[:space:]]*-[[:space:]]*\[[[:space:]]*\][[:space:]]*([0-9]+(.[0-9]+)*)[[:space:]]*\.?[[:space:]]*(.+)$ ]]; then
        # If we were processing a previous task, generate its command
        if [ "$in_task" = true ] && [ -n "$current_task_id" ]; then
            generate_task_command "$current_task_id" "$current_task_desc" "$current_leverage" "$current_requirements"
            ((TASK_COUNT++))
        fi
        
        # Start new task
        current_task_id="${BASH_REMATCH[1]}"
        current_task_desc="${BASH_REMATCH[3]}"
        current_leverage=""
        current_requirements=""
        in_task=true
    elif [ "$in_task" = true ]; then
        # Look for _Leverage: lines
        if [[ $line =~ ^[[:space:]]*-[[:space:]]*_Leverage:[[:space:]]*(.+)$ ]]; then
            current_leverage="${BASH_REMATCH[1]}"
        # Look for _Requirements: lines
        elif [[ $line =~ ^[[:space:]]*-[[:space:]]*_Requirements:[[:space:]]*(.+)$ ]]; then
            current_requirements="${BASH_REMATCH[1]}"
        # If we hit another checkbox or end of task context, stop collecting for this task
        elif [[ $line =~ ^[[:space:]]*-[[:space:]]*\[ ]]; then
            # This might be the start of a new task or sub-task, let the main parser handle it
            continue
        fi
    fi
done < "$TASKS_FILE"

# Don't forget the last task
if [ "$in_task" = true ] && [ -n "$current_task_id" ]; then
    generate_task_command "$current_task_id" "$current_task_desc" "$current_leverage" "$current_requirements"
    ((TASK_COUNT++))
fi

echo
echo "Generated $TASK_COUNT task commands for spec: $SPEC_NAME"
echo "Commands created in: .claude/commands/$SPEC_NAME/"
echo
echo "Generated commands:"

# Show generated commands  
while IFS= read -r line; do
    if [[ $line =~ ^[[:space:]]*-[[:space:]]*\[[[:space:]]*\][[:space:]]*([0-9]+(.[0-9]+)*)[[:space:]]*\.?[[:space:]]*(.+)$ ]]; then
        task_id="${BASH_REMATCH[1]}"
        task_desc="${BASH_REMATCH[3]}"
        echo "  /$SPEC_NAME-task-$task_id - $task_desc"
    fi
done < "$TASKS_FILE"

echo
echo "============================================================"
echo "IMPORTANT: Please restart Claude Code for the new commands to be visible"
echo "============================================================"
echo
echo "The task commands have been generated successfully."
echo "After restarting Claude Code, you can use commands like:"
echo "  /$SPEC_NAME-task-1"
echo "  /$SPEC_NAME-task-2"
echo "  etc."
echo

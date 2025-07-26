# Command Generation Instructions

## Platform-Specific Script Execution

The command generation system now uses platform-specific scripts instead of JavaScript to avoid conflicts and ensure cross-platform compatibility.

### Available Scripts:
- **Windows**: `generate-commands.bat`
- **macOS/Linux**: `generate-commands.sh`
- **Launcher**: `generate-commands-launcher.sh` (auto-detects OS)

### Agent Instructions:

**CRITICAL**: Use the OS detection launcher script instead of the old JavaScript version.

1. **After tasks.md approval**, execute the appropriate command:

   **Option 1 - Use the launcher (recommended):**
   ```bash
   ./.claude/scripts/generate-commands-launcher.sh {spec-name}
   ```

   **Option 2 - Platform-specific execution:**

   **Windows:**
   ```cmd
   .claude\scripts\generate-commands.bat {spec-name}
   ```

   **macOS/Linux:**
   ```bash
   ./.claude/scripts/generate-commands.sh {spec-name}
   ```

2. **IMPORTANT**: After script completion, inform the user to restart Claude Code for the new commands to be visible.

2. **OS Detection**: The launcher script automatically detects the operating system and runs the appropriate platform-specific script.

3. **Functionality**: All scripts provide the same functionality:
   - Parse tasks.md files
   - Support hierarchical task numbering (1, 2, 2.1, 2.2, etc.)
   - Generate command files in .claude/commands/{spec-name}/ directory
   - Create individual task commands like /{spec-name}-task-{id}

4. **Integration**: These scripts replace the old `node .claude/scripts/generate-commands.js` command.

### Migration Notes:
- **DO NOT** use `node .claude/scripts/generate-commands.js` anymore
- **DO NOT** reference the JavaScript version in instructions
- **ALWAYS** use the platform-specific scripts or the launcher
- The scripts are generated during setup and stored in `.claude/scripts/`

### Error Handling:
If the launcher fails to detect the OS or find the appropriate script:
1. Check that all three scripts exist in `.claude/scripts/`
2. Ensure the launcher script has execute permissions
3. Manually run the platform-specific script if needed

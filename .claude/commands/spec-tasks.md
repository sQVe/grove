# Spec Tasks Command

Generate implementation task list based on approved design.

## Usage
```
/spec-tasks [feature-name]
```

## Phase Overview
**Your Role**: Break design into executable implementation tasks

This is Phase 3 of the spec workflow. Your goal is to create a detailed task breakdown that will guide the implementation of the feature.

## Instructions
You are working on the tasks phase of the spec workflow.

**WORKFLOW**: This is the FINAL step before command generation.
**SEQUENCE**: Create Tasks → Get Approval → Ask User → Generate Commands
**DO NOT** run task command generation until tasks are approved.

1. **Prerequisites**
   - Ensure design.md exists and is approved in `.claude/specs/{feature-name}/`
   - Load both documents from the spec directory:
     - Load `.claude/specs/{feature-name}/requirements.md` for feature context
     - Load `.claude/specs/{feature-name}/design.md` for technical design
   - **Load steering documents** (if available):
     - Check for .claude/steering/structure.md for project conventions
     - Check for .claude/steering/tech.md for technical patterns
   - Understand the complete feature scope

2. **Task Creation Process**
   1. **Read the task template**: Load `.claude/templates/tasks-template.md` and follow its exact structure
   2. Convert design into atomic, executable coding tasks
   3. Ensure each task:
      - Has a clear, actionable objective
      - References specific requirements using _Requirements: X.Y_ format
      - References existing code to leverage using _Leverage: path/to/file.ts_ format
      - Builds incrementally on previous tasks
      - Focuses on coding activities only
   4. Use checkbox format with hierarchical numbering as shown in template
   5. Present complete task list following the template structure
   6. Ask: "Do the tasks look good?"
   7. **CRITICAL**: Wait for explicit approval before proceeding
   8. **AFTER APPROVAL**: Ask "Would you like me to generate individual task commands for easier execution? (yes/no)"
   9. **IF YES**: Execute `npx @pimzino/claude-code-spec-workflow@latest generate-task-commands {feature-name}`
   10. **IF NO**: Continue with traditional task execution approach

3. **Generate Task List** (prioritize code reuse and follow conventions)
   - Break design into atomic, executable coding tasks
   - **Follow structure.md**: Ensure tasks respect project file organization
   - **Prioritize extending/adapting existing code** over building from scratch
   - Use checkbox format with numbered hierarchy
   - Each task should reference specific requirements AND existing code to leverage
   - Focus ONLY on coding tasks (no deployment, user testing, etc.)

4. **Task Guidelines**
   - Tasks should be concrete and actionable
   - **Reference existing code to reuse**: Include specific files/components to extend or adapt
   - Include specific file names and components
   - Build incrementally (each task builds on previous)
   - Reference requirements using _Requirements: X.Y_ format
   - Use test-driven development approach leveraging existing test patterns

## Template Usage
- **Follow exact structure**: Use `.claude/templates/tasks-template.md` precisely
- **Include all sections**: Don't omit any required template sections
- **Use checkbox format**: Follow the exact task format with requirement and leverage references
- **See template for format rules**: The template includes detailed formatting guidelines

### Excluded Task Types
- User acceptance testing
- Production deployment
- Performance metrics gathering
- User training or documentation
- Business process changes

5. **Approval Process**
   - Present the complete task list
   - Ask: "Do the tasks look good?"
   - Make revisions based on feedback
   - Continue until explicit approval
   - **CRITICAL**: Do not proceed without explicit approval

6. **Critical Task Command Generation Rules**

**Use NPX Command for Task Generation**: Task commands are now generated using the package's CLI command.
- **COMMAND**: `npx @pimzino/claude-code-spec-workflow@latest generate-task-commands {spec-name}`
- **TIMING**: Only run after tasks.md is approved AND user confirms they want task commands
- **USER CHOICE**: Always ask the user if they want task commands generated (yes/no)
- **CROSS-PLATFORM**: Works automatically on Windows, macOS, and Linux

### Generate Task Commands (ONLY after tasks approval)
- **WAIT**: Do not run command generation until user explicitly approves tasks
- **ASK USER**: "Would you like me to generate individual task commands for easier execution? (yes/no)"
- **IF YES**: Execute `npx @pimzino/claude-code-spec-workflow@latest generate-task-commands {feature-name}`
- **IF NO**: Continue with traditional task execution approach
- **PURPOSE**: Creates individual task commands in `.claude/commands/{feature-name}/`
- **RESULT**: Each task gets its own command: `/{feature-name}-task-{task-id}`
- **EXAMPLE**: Creates `/{feature-name}-task-1`, `/{feature-name}-task-2.1`, etc.
- **RESTART REQUIRED**: Inform user to restart Claude Code for new commands to be visible


## Critical Rules
- **NEVER** proceed to the next phase without explicit user approval
- Accept only clear affirmative responses: "yes", "approved", "looks good", etc.
- If user provides feedback, make revisions and ask for approval again
- Continue revision cycle until explicit approval is received

## Next Phase
After approval and command generation:
1. **RESTART Claude Code** for new commands to be visible
2. Then you can:
   - Use individual task commands (if generated): `/{feature-name}-task-1`, `/{feature-name}-task-2`, etc.
   - Or execute tasks individually as needed
   - Check progress with `/spec-status {feature-name}`

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
   - Ensure design.md exists and is approved
   - Load both requirements.md and design.md for context
   - **Load steering documents**:
     - Check for .claude/steering/structure.md for project conventions
     - Check for .claude/steering/tech.md for technical patterns
   - Understand the complete feature scope

2. **Process**
   1. Convert design into atomic, executable coding tasks
   2. Ensure each task:
      - Has a clear, actionable objective
      - References specific requirements using _Requirements: X.Y_ format
      - Builds incrementally on previous tasks
      - Focuses on coding activities only
   3. Use checkbox format with hierarchical numbering
   4. Present complete task list
   5. Ask: "Do the tasks look good?"
   6. **CRITICAL**: Wait for explicit approval before proceeding
   7. **AFTER APPROVAL**: Ask "Would you like me to generate individual task commands for easier execution? (yes/no)"
   8. **IF YES**: Execute `npx @pimzino/claude-code-spec-workflow@latest generate-task-commands {feature-name}`
   9. **IF NO**: Continue with traditional `/spec-execute` approach

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

### Task Format
```markdown
- [ ] 1. Task description
  - Specific implementation details
  - Files to create/modify
  - _Leverage: existing-component.ts, utils/helpers.js_
  - _Requirements: 1.1, 2.3_
```

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
- **IF NO**: Continue with traditional `/spec-execute` approach
- **PURPOSE**: Creates individual task commands in `.claude/commands/{feature-name}/`
- **RESULT**: Each task gets its own command: `/{feature-name}-task-{task-id}`
- **EXAMPLE**: Creates `/{feature-name}-task-1`, `/{feature-name}-task-2.1`, etc.
- **RESTART REQUIRED**: Inform user to restart Claude Code for new commands to be visible

## Task Structure Example
```markdown
# Implementation Plan

## Task Overview
[Brief description of the implementation approach]

## Steering Document Compliance
[How tasks follow structure.md conventions and tech.md patterns]

## Tasks

- [ ] 1. Set up project structure and core interfaces
  - Create directory structure following existing patterns
  - Define core interfaces extending existing base classes
  - Set up basic configuration
  - _Leverage: src/types/base.ts, src/models/BaseModel.ts_
  - _Requirements: 1.1_

- [ ] 2. Implement data models and validation
- [ ] 2.1 Create base model classes
  - Define data structures/schemas
  - Implement validation methods
  - Write unit tests for models
  - _Leverage: src/utils/validation.ts, tests/helpers/testUtils.ts_
  - _Requirements: 2.1, 2.2_

- [ ] 2.2 Implement specific model classes
  - Create concrete model implementations
  - Add relationship handling
  - Test model interactions
  - _Requirements: 2.3_
```

## Critical Rules
- **NEVER** proceed to the next phase without explicit user approval
- Accept only clear affirmative responses: "yes", "approved", "looks good", etc.
- If user provides feedback, make revisions and ask for approval again
- Continue revision cycle until explicit approval is received

## Next Phase
After approval and command generation:
1. **RESTART Claude Code** for new commands to be visible
2. Then you can:
   - Use `/spec-execute` to implement tasks
   - Use individual task commands: `/{feature-name}-task-1`, `/{feature-name}-task-2`, etc.
   - Check progress with `/spec-status {feature-name}`

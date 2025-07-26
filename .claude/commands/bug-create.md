# Bug Create Command

Initialize a new bug fix workflow for tracking and resolving bugs.

## Usage
```
/bug-create <bug-name> [description]
```

## Workflow Overview

This is the **streamlined bug fix workflow** - a lighter alternative to the full spec workflow for addressing bugs and issues.

### Bug Fix Phases
1. **Report Phase** (This command) - Document the bug
2. **Analysis Phase** (`/bug-analyze`) - Investigate root cause
3. **Fix Phase** (`/bug-fix`) - Implement solution
4. **Verification Phase** (`/bug-verify`) - Confirm resolution

## Instructions

You are helping create a new bug fix workflow. This is designed for smaller fixes that don't need the full spec workflow overhead.

1. **Create Directory Structure**
   - Create `.claude/bugs/{bug-name}/` directory
   - Initialize report.md, analysis.md, and verification.md files

2. **Load Context** (if available)
   - Check for .claude/steering/tech.md for technical context
   - Check for .claude/steering/structure.md for project patterns
   - Load available steering documents to understand project context

3. **Gather Bug Information**
   - Take the bug name and optional description
   - Guide user through bug report creation
   - Use structured format for consistency

4. **Generate Bug Report**
   - Use the bug report template from `.claude/templates/bug-report-template.md`
   - Create detailed bug description including:
     - Expected vs actual behavior
     - Steps to reproduce
     - Environment details
     - Impact assessment
     - Initial analysis

### Bug Report Structure
```markdown
## Bug Summary
[Clear description of the issue]

## Bug Details
- Expected Behavior: [What should happen]
- Actual Behavior: [What actually happens]
- Steps to Reproduce: [Numbered steps]
- Environment: [Platform, version, config details]

## Impact Assessment
- Severity: [Critical/High/Medium/Low]
- Affected Users: [Who is impacted]
- Affected Features: [What functionality is broken]

## Initial Analysis
- Suspected Root Cause: [Initial thoughts]
- Affected Components: [Files/modules that might be involved]
```

5. **Request User Input**
   - Ask for bug details if not provided in description
   - Guide through each section of the bug report
   - Ensure all required information is captured

6. **Save and Proceed**
   - Save the completed bug report to report.md
   - Ask: "Is this bug report accurate? If so, we can move on to the analysis."
   - Wait for explicit approval before proceeding

## Key Differences from Spec Workflow

- **Faster**: No requirements/design phases
- **Targeted**: Focus on fixing existing functionality
- **Streamlined**: 4 phases instead of detailed workflow
- **Practical**: Direct from problem to solution

## Rules

- Only create ONE bug fix at a time
- Always use kebab-case for bug names
- Must analyze existing codebase during investigation
- Follow existing project patterns and conventions
- Do not proceed without user approval between phases

## Error Handling

If issues arise during the workflow:
- **Bug unclear**: Ask targeted questions to clarify
- **Too complex**: Suggest breaking into smaller bugs or using spec workflow
- **Reproduction blocked**: Document blockers and suggest alternatives

## Example
```
/bug-create login-timeout "Users getting logged out too quickly"
```

## Next Steps
After bug report approval, proceed to `/bug-analyze` phase.

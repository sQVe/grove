# Spec Requirements Command

Generate or update requirements document for an existing spec.

## Usage
```
/spec-requirements [feature-name]
```

## Phase Overview
**Your Role**: Generate comprehensive requirements based on user input

This is Phase 1 of the spec workflow. Your goal is to create a complete requirements document that will guide the rest of the feature development.

## Instructions
You are working on the requirements phase of the spec workflow.

1. **Identify Current Spec**
   - If no feature-name provided, look for specs in `.claude/specs/` directory
   - If multiple specs exist, ask user to specify which one
   - If feature-name provided, load from `.claude/specs/{feature-name}/requirements.md`
   - Check if requirements.md already exists in the spec directory

2. **Load Context**
   - **Load steering documents**: 
     - Check for .claude/steering/product.md for product vision alignment
     - Check for .claude/steering/tech.md for technical constraints
     - Check for .claude/steering/structure.md for organizational patterns
   - **Analyze codebase**: Search for similar features and patterns

3. **Generate Requirements Document**
   - **Template to Follow**: Use the exact structure from `.claude/templates/requirements-template.md`
   - **Use EARS format**: Easy Approach to Requirements Syntax for acceptance criteria
   - Each requirement should have:
     - User story: "As a [role], I want [feature], so that [benefit]"
     - Numbered acceptance criteria: "WHEN [event] THEN [system] SHALL [response]"
   - **Ensure alignment**: Verify requirements support goals outlined in product.md

### Requirements Creation Process
1. **Read the requirements template**: Load `.claude/templates/requirements-template.md` and follow its exact structure
2. Parse the feature description provided by the user
3. Create user stories in format: "As a [role], I want [feature], so that [benefit]"
4. Generate acceptance criteria using EARS format:
   - WHEN [event] THEN [system] SHALL [response]
   - IF [condition] THEN [system] SHALL [response]
5. Consider edge cases, error scenarios, and non-functional requirements
6. Present complete requirements document following the template structure
7. Ask: "Do the requirements look good? If so, we can move on to the design phase."
8. **CRITICAL**: Wait for explicit approval before proceeding to the design phase

4. **Content Guidelines**
   - Consider edge cases and error handling
   - Include non-functional requirements (performance, security, etc.)
   - Reference existing codebase patterns where relevant
   - **Align with product vision**: Ensure all requirements support product.md goals
   - Ensure requirements are testable and verifiable

5. **Approval Process**
   - Present the complete requirements document
   - Ask: "Do the requirements look good? If so, we can move on to the design phase."
   - Make revisions based on feedback
   - Continue until explicit approval is received
   - **CRITICAL**: Do not proceed without explicit approval

## Template Usage
- **Follow exact structure**: Use `.claude/templates/requirements-template.md` precisely
- **Include all sections**: Don't omit any required template sections
- **EARS format**: Use proper WHEN/IF/THEN statements for acceptance criteria

## Critical Rules
- **NEVER** proceed to the next phase without explicit user approval
- Accept only clear affirmative responses: "yes", "approved", "looks good", etc.
- If user provides feedback, make revisions and ask for approval again
- Continue revision cycle until explicit approval is received

## Next Phase
After approval, the next phase is design creation. The user can proceed to work on the design document.

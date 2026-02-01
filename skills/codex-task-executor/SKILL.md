---
name: codex-task-executor
description: Execute coding tasks using GPT-5.2-Codex (WRITES CODE, modifies files). Implements features based on plan files and task specifications. Creates, modifies, and edits files to build functionality. Requires explicit task description and plan context from Claude Code's planning phase. Use when Claude Code creates tasks from a plan and needs to delegate actual implementation work. Triggers when user says "execute this task with codex", "implement task", "delegate implementation to codex", "build this feature", "codex create this", or when explicitly offloading coding work. NOT for code review or analysis - use codex-review for that.
---

# Codex Task Executor

Execute coding tasks using GPT-5.2-Codex with full context from plan files and task descriptions.

## Purpose

This skill enables **delegation of implementation work** from Claude Code (orchestrator) to Codex (specialist coder):

- **Claude Code**: Creates plans, breaks down tasks, orchestrates workflow
- **Codex**: Implements specific tasks autonomously with minimal context overhead

## Requirements

**Pre-built binary included:**
- macOS Apple Silicon (M1/M2/M3) - `bin/execute-task-darwin-arm64`

**For other platforms:**
- Go 1.22+ required to build from source
- See [appendix/BUILD.md](appendix/BUILD.md) for build instructions
- AI agents can build it following the guide

**Runtime:**
- `OPENAI_API_KEY` environment variable

## Invocation

```bash
~/.claude/skills/codex-task-executor/bin/execute-task-darwin-arm64 "<task-name>" "<task-description>" "<plan-file-path>"
```

**Task Name**: Generate using plan file pattern (adjective-verb-noun).
- Examples: "auth-implementing-lovelace", "ui-building-hopper"
- Same name for follow-up if [QUESTION] appears

**Task Description**: Structured description for Codex implementation (see Context Preparation below).

**Plan File**: Path to plan markdown file with overall architecture.

**Example**:
```python
task_description = """
Implement UserAuth component with JWT validation.

Requirements:
- Email/password inputs, POST /api/auth/login
- Store JWT, redirect to /dashboard
- Error handling

Patterns: Read LoginForm.tsx, use apiCall from lib/api.ts
Edge Cases: Empty fields, network errors, 401 responses
"""

Bash(
    command=f'~/.claude/skills/codex-task-executor/bin/execute-task-darwin-arm64 "auth-implementing-lovelace" "{task_description}" "plan.md"',
    description="Implementing UserAuth"
)
```

### Parameters

1. **task-id**: Unique identifier (e.g., `task-3`, `implement-auth-2`)
   - Must match `^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$` (start with alphanumeric)
   - Used for session persistence

2. **task-description**: What needs to be implemented
   - Should be specific and actionable
   - Example: "Implement UserAuth component with JWT validation"

3. **plan-file-path**: Path to the plan markdown file
   - Absolute or relative path
   - Contains overall architecture and context

### Task Naming

**Generate using plan file pattern** (adjective-verb-noun):
- Examples: "auth-implementing-lovelace", "ui-building-hopper"
- Unique and readable
- Reuse for follow-up if [QUESTION] appears

## Context Preparation (Claude Code's Role)

Before invoking this skill, Claude Code should provide rich context. The more context, the better Codex performs.

### Task Description Structure (CRITICAL)

Include in task description passed to Codex:

**Requirements**:
- Specific requirements with acceptance criteria
- What to build, what behavior to implement

**Patterns to Match**:
- Existing code files to reference (e.g., "Read LoginForm.tsx for patterns")
- Helpers and utilities to use (e.g., "Use apiCall from lib/api.ts")
- Styling and structure conventions to follow

**Integration Points**:
- Where to add/modify code
- Which files to update
- How to connect with existing systems

**Edge Cases**:
- Input validation scenarios
- Error handling requirements
- Network/API failure cases

**Example task description format**:
```
Implement UserAuth component with JWT validation.

Requirements:
- Email/password form inputs
- POST /api/auth/login, store JWT, redirect to /dashboard
- Display inline error messages

Patterns: Read LoginForm.tsx for structure, use apiCall from lib/api.ts
Edge Cases: Empty fields, network errors, 401 responses
```

### Automatically Included

Codex binary automatically loads:
- **Plan file contents**: Full plan markdown
- **CLAUDE.md**: Project guidelines (if exists)
- **Repository root**: Auto-detected from git

---

## Output Monitoring

Codex communicates via **output markers** in stdout:

### [PROGRESS] - Progress Updates
```
[PROGRESS] Created src/components/UserAuth.tsx
[PROGRESS] Implementing JWT validation logic
```

**Claude Code action**: Update UI, show progress to user

### [QUESTION] - Needs Clarification
```
[QUESTION] Should I use localStorage or sessionStorage?
Options:
1. localStorage - Persists across sessions
2. sessionStorage - Cleared on close
```

**Claude Code action**:
1. Subagent pauses execution when [QUESTION] appears
2. Ask user via AskUserQuestion tool
3. Re-invoke with answer appended to task description

### [BLOCKED] - Cannot Proceed
```
[BLOCKED] API_BASE_URL not defined in config
```

**Claude Code action**: Resolve blocker, update context, re-run

### [FILES_MODIFIED] - Summary
```
[FILES_MODIFIED]
- src/components/UserAuth.tsx (created)
- src/App.tsx (modified)
```

**Claude Code action**: Parse and update task metadata

### [CODEX_COMPLETE] - Done
```
[CODEX_COMPLETE] Task completed in 12 iterations
```

**Claude Code action**: Mark task as completed

---

## Subagent Execution Workflow

**Pattern with structured contexts:**

### 1. Build Both Contexts

Structure Codex input AND subagent task:

```python
# Codex input - detailed and organized (CRITICAL)
task_description = """
Implement UserAuth component with JWT validation.

Requirements:
- Email/password inputs with validation
- POST /API/auth/login, store JWT, redirect to /dashboard
- Display inline error messages

Patterns to Match:
- Read LoginForm.tsx for form structure
- Use apiCall helper from lib/api.ts

Edge Cases:
- Empty fields, network errors, 401 responses
"""

Bash(
    command=f'~/.claude/skills/codex-task-executor/bin/execute-task-darwin-arm64 "auth-implementing-lovelace" "{task_description}" "plan.md"',
    description="Implementing UserAuth"
)
```

### 2. Inform User

"Delegating implementation to subagent. Will process results when complete."

### 3. Subagent Automatically Returns

**No manual checking needed** - subagent returns output when complete.

**Parse the output**:
- Check for `[CODEX_COMPLETE]` - success
- Check for `[BLOCKED]` or `[QUESTION]` - needs intervention
- Check for `[FILES_MODIFIED]` - parse modified files
- Check exit code - non-zero indicates error

**Action based on result**:

**Success**:
```
"Task #1 completed successfully. Codex modified:
- src/components/UserAuth.tsx (created)
- src/App.tsx (modified)

Would you like me to review the changes or move to the next task?"
```

**Needs intervention**:
```
"Task #1 needs your input:
[QUESTION] Should I use localStorage or sessionStorage?

Please choose and I'll restart the task with your answer."
```

**Error**:
```
"Task #1 encountered an error:
[BLOCKED] API_BASE_URL not defined in config

Let me resolve this and restart the task."
```

### 4. Parallel Task Execution

Subagent delegation enables parallel implementation:

```python
# Delegate multiple tasks concurrently
for i, task_desc in enumerate(tasks):
    # Generate unique task name (plan file pattern)
    task_name = f"task{i}-implementing-{['lovelace', 'hopper', 'turing', 'knuth'][i % 4]}"

    Bash(
        command=f'execute-task-darwin-arm64 "{task_name}" "{task_desc}" "plan.md"',
        description=f"Task #{i}"
    )

# All subagents run in parallel, each auto-returns when complete
```

---

## Session Management

Sessions stored per-task in project directory:
```
{repo}/.codex-sessions/tasks/
├── task-1-1738224567-a3f9.json  # Unique session
├── task-2-1738224590-b2d1.json
└── task-3-1738224612-c5e8.json
```

**Multi-turn support**: Same task-id = same conversation continues

**CRITICAL: Concurrent Session Safety**

When multiple Claude Code sessions run simultaneously, use **unique task IDs** (timestamp + random) to avoid session file collisions.

**Safe**: Different sessions use different IDs
- Session A: `task-1-1738224567-a3f9`
- Session B: `task-1-1738224590-b2d1`
- ✅ No collision: Different session files

**Unsafe**: Same simple ID across sessions
- Session A: `task-1`
- Session B: `task-1`
- ❌ COLLISION: Both write to same task-1.json

**Follow-up conversations**: Use same task-id in new Task invocation to continue conversation.

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENAI_API_KEY` | (required) | OpenAI API key |
| `OPENAI_MODEL` | `gpt-5.2-codex` | Model name |
| `REASONING_EFFORT` | `medium` | low/medium/high/xhigh |
| `REPO_ROOT` | git root | Repository root |
| `STATE_DIR` | `{repo}/.codex-sessions/tasks` | Session storage |
| `MAX_ITERS` | `50` | Max tool iterations |

**Reasoning effort guide:**
- `low`: Simple CRUD, file copying
- `medium`: Components, API integration (default)
- `high`: Complex algorithms, refactoring
- `xhigh`: Critical security/performance

---

## Platform Security

### Unix (Linux/macOS) - Production Ready ✅

**Security: 9.5/10** - Perfect symlink and TOCTOU protection

Uses `openat` syscalls for bulletproof file operations:
- No symlink attacks possible
- No TOCTOU race conditions
- Repository escape impossible

### Windows - Best Effort ⚠️

**Security: 7/10** - Good for trusted repositories

Limitations:
- Junction/reparse points not fully blocked
- Small TOCTOU window exists
- **Recommended**: Use WSL2 for production workloads

See [appendix/SECURITY.md](appendix/SECURITY.md) for detailed analysis and recommendations.

---

## Complete Workflow Example (Subagent Delegation)

### 1. Claude Code Creates Plan
```markdown
# Plan: Authentication System

## Task #3: Implement UserAuth Component
Create login component with JWT validation
```

### 2. Claude Code Prepares Context
```python
# Read patterns to include
pattern = Read("src/components/LoginForm.tsx")

task_desc = """
Implement UserAuth component.

Requirements:
- Email/password inputs
- POST /api/auth/login on submit
- Store JWT in localStorage
- Redirect to /dashboard
- Show inline errors

Match LoginForm.tsx pattern (read file).
Use apiCall from lib/api.ts.
"""
```

### 3. Execute with Bash
```python
Bash(
    command=f'~/.claude/skills/codex-task-executor/bin/execute-task-darwin-arm64 "auth-implementing-lovelace" "{task_desc}" ".claude/plans/auth.md"',
    description="Implementing UserAuth"
)

# Tell user immediately
"Delegated Task #3 to subagent. Will process results when complete."
```

### 4. Codex Output (In Subagent)
```
[PROGRESS] Exploring existing patterns
[Tool: Glob] src/components/**/*.tsx
[Tool: Read] src/components/LoginForm.tsx

[PROGRESS] Creating UserAuth component
[Tool: Write] src/components/UserAuth.tsx

[PROGRESS] Integrating into App
[Tool: Edit] src/App.tsx

[FILES_MODIFIED]
- src/components/UserAuth.tsx (created)
- src/App.tsx (modified)

[CODEX_COMPLETE] Task completed in 8 iterations
```

### 5. Subagent Auto-Returns Results

**Claude Code parses and responds to user**:
```
"Task #3 completed successfully! Codex implemented:
- src/components/UserAuth.tsx (created)
- src/App.tsx (modified - added UserAuth to routes)

The component matches LoginForm.tsx patterns and includes error handling for empty fields and network errors. Would you like me to review the implementation or proceed to the next task?"
```

### 6. Parallel Task Delegation

```python
# Execute all tasks in parallel
for task in plan.tasks:
    # Generate task name (plan file pattern)
    task_name = f"task{task.number}-implementing-{get_random_scientist()}"

    Bash(
        command=f'execute-task-darwin-arm64 "{task_name}" "{task.description}" "plan.md"',
        description=f"Task #{task.number}"
    )

"Delegated 5 tasks to subagents in parallel. Will process each as they complete."
```

---

## When to Use

### ✅ Good Use Cases
- Task has clear spec from plan
- Straightforward implementation work
- Want to save Claude's context
- Specialized coding model beneficial

### ❌ Bad Use Cases
- Architecture decisions needed
- Requires mid-execution user input
- Exploratory research
- Trivial (< 10 lines)

---

## Troubleshooting

**"OPENAI_API_KEY required"**
→ `export OPENAI_API_KEY="sk-..."`

**"Plan file not found"**
→ Use absolute path or check cwd

**"MAX_ITERS reached"**
→ Increase `MAX_ITERS=100` or break task smaller

**Edit fails "old_string not found"**
→ Codex should Read first for exact match

---

## Tips for Best Results

1. **Provide examples**: "Add error handling like LoginForm.tsx"
2. **Reference existing code**: "Match pattern in UserList.tsx"
3. **Be specific**: "Import in App.tsx and add to routes array"
4. **Specify edge cases**: "Handle empty email, network errors"
5. **Set expectations**: "Simple CRUD, no fancy optimizations"

---

## Reference Materials

**Load when needed:**
- [references/tool-protocols.md](references/tool-protocols.md) - Output marker format specification

## Appendix

*Human reference only:*
- [appendix/BUILD.md](appendix/BUILD.md) - Build instructions for all platforms
- [appendix/SECURITY.md](appendix/SECURITY.md) - Security implementation details

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
# macOS Apple Silicon (pre-built, ready to use)
~/.claude/skills/codex-task-executor/bin/execute-task-darwin-arm64 \
  "<task-id>" \
  "<task-description>" \
  "<plan-file-path>"

# Other platforms (after building - see BUILD.md)
~/.claude/skills/codex-task-executor/bin/execute-task \
  "<task-id>" \
  "<task-description>" \
  "<plan-file-path>"
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

### IMPORTANT: Task ID Uniqueness

**When multiple Claude Code sessions run simultaneously**, task IDs must be globally unique to prevent session file collisions.

**Generate unique task IDs with timestamp or random suffix:**

```python
import time, random

# Method 1: Timestamp-based
task_id = f"task-1-{int(time.time())}"
# → "task-1-1738224567"

# Method 2: Timestamp + Random (recommended)
task_id = f"task-1-{int(time.time())}-{random.randint(0x1000, 0xffff):04x}"
# → "task-1-1738224567-a3f9"

# Method 3: Using TaskCreate ID
task_id = f"task-{task.id}"  # If TaskCreate returns unique ID
```

**Examples:**
```bash
# ✅ Good (unique)
./execute-task.py "task-1-1738224567-a3f9" "..." "plan.md"
./execute-task.py "implement-auth-fe82a1" "..." "plan.md"

# ❌ Bad (collision risk)
./execute-task.py "task-1" "..." "plan.md"
./execute-task.py "implement-auth" "..." "plan.md"
```

**Why:** Session files are stored as `{task-id}.json`. If two sessions use the same task-id, they will overwrite each other's conversation state.

## Context Preparation (Claude Code's Role)

Before invoking this skill, Claude Code should provide rich context. The more context, the better Codex performs.

### Required Context

**Minimal invocation:**
```bash
./execute-task.py "task-1" \
  "Add login button to navbar" \
  ".claude/plans/auth-feature.md"
```

### Rich Context (Recommended)

Include in task description:
- **Specific requirements** with acceptance criteria
- **Existing patterns** to match (mention file names)
- **Integration points** (where to add/modify)
- **Edge cases** to handle

**Example:**
```bash
./execute-task.py "task-3" \
  "Implement UserAuth component with JWT validation.

  Requirements:
  - Email/password form
  - Call POST /api/auth/login
  - Store JWT in localStorage
  - Redirect to /dashboard on success
  - Display errors inline

  Match LoginForm.tsx style (read for patterns).
  Use apiCall helper from lib/api.ts.
  Handle: empty fields, network errors, 401 responses." \
  ".claude/plans/auth-system.md"
```

### Automatically Included

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
1. Stop execution (Ctrl+C)
2. Ask user via AskUserQuestion tool
3. Re-run with answer appended to task description

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

When multiple Claude Code sessions run simultaneously on the same project, they must use **unique task IDs** to avoid session file collisions.

**Safe pattern** (generated in "Task ID Uniqueness" section above):
```bash
# Session A
./execute-task.py "task-1-1738224567-a3f9" "..." "plan.md"

# Session B (different timestamp/random)
./execute-task.py "task-1-1738224590-b2d1" "..." "plan.md"

# ✅ No collision: Different session files
```

**Unsafe pattern:**
```bash
# Session A
./execute-task.py "task-1" "..." "plan.md"

# Session B
./execute-task.py "task-1" "..." "plan.md"

# ❌ COLLISION: Both write to same task-1.json
```

**Follow-up example:**
```bash
# Initial
./execute-task.py "task-3-1738224567-a3f9" "Implement auth" "plan.md"
# → Asks question

# Follow-up (same task-id for continuation)
./execute-task.py "task-3-1738224567-a3f9" "Use localStorage" "plan.md"
# → Continues same conversation
```

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

## Complete Workflow Example

### 1. Claude Code Creates Plan
```markdown
# Plan: Authentication System

## Task #3: Implement UserAuth Component
Create login component with JWT validation
```

### 2. Claude Code Prepares Context
```typescript
// Read patterns to include
const pattern = await Read("src/components/LoginForm.tsx");

const taskDesc = `
Implement UserAuth component.

Requirements:
- Email/password inputs
- POST /api/auth/login on submit
- Store JWT in localStorage
- Redirect to /dashboard
- Show inline errors

Match LoginForm.tsx pattern (read file).
Use apiCall from lib/api.ts.
`;
```

### 3. Invoke Codex
```bash
./execute-task.py "task-3" "${taskDesc}" ".claude/plans/auth.md"
```

### 4. Codex Output
```
[PROGRESS] Exploring existing patterns
[Tool: Glob] src/components/**/*.tsx
[Tool: Read] src/components/LoginForm.tsx

[PROGRESS] Creating UserAuth component
[Tool: Write] src/components/UserAuth.tsx

[QUESTION] Validation on blur or on submit?
Options:
1. On blur - Immediate feedback
2. On submit - Less intrusive

Continuing with on submit (LoginForm pattern)...

[PROGRESS] Integrating into App
[Tool: Edit] src/App.tsx

[FILES_MODIFIED]
- src/components/UserAuth.tsx (created)
- src/App.tsx (modified)

[CODEX_COMPLETE] Task completed in 8 iterations
```

### 5. Claude Code Processes
- Parses [FILES_MODIFIED]
- Updates task status → completed
- Shows user summary
- Moves to next task

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

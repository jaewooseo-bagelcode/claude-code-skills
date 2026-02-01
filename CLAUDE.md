# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is a multi-skill repository containing professional AI agent skills for Claude Code. Skills are modular packages that extend Claude's capabilities with specialized knowledge, workflows, and tools.

**Architecture**: Two-tier skill system
- **Installable skills** (`skills/`): Production skills for distribution (codex-review, codex-task-executor)
- **Development tools** (`.claude/skills/`): Meta-tools for creating/maintaining skills (skill-creator)

## Common Commands

### Skill Development

```bash
# Initialize new skill
.claude/skills/skill-creator/scripts/init_skill.py <skill-name> --path ./skills/<skill-name>

# Validate skill structure
.claude/skills/skill-creator/scripts/quick_validate.py skills/<skill-name>

# Package skill for distribution
.claude/skills/skill-creator/scripts/package_skill.py skills/<skill-name>
```

### Building Go-based Skills

Both codex-review and codex-task-executor are Go-based tools:

```bash
# Build from scripts/ directory
cd skills/<skill-name>/scripts
go mod download
go build -ldflags="-s -w" -o ../bin/<binary-name>-$(go env GOOS)-$(go env GOARCH)

# Platform-specific builds
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o ../bin/<binary-name>-darwin-arm64
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ../bin/<binary-name>-linux-amd64
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ../bin/<binary-name>-windows-amd64.exe
```

Binary names:
- codex-review: `codex-review-<platform>-<arch>`
- codex-task-executor: `execute-task-<platform>-<arch>`

### Installation Testing

```bash
# Test OpenSkills installation (recommended method)
npx openskills install jaewooseo-bagelcode/claude-code-skills
npx openskills sync

# Test individual skill installation
npx openskills install jaewooseo-bagelcode/claude-code-skills/skills/codex-review
```

## Architecture

### Skill Structure

Every skill follows this anatomy:

```
skill-name/
├── SKILL.md              # Required: YAML frontmatter + markdown instructions
├── bin/                  # Pre-built binaries (for Go-based skills)
├── scripts/              # Executable code (Python/Go/Bash)
├── references/           # Documentation loaded as needed
├── assets/               # Templates/resources used in output
└── appendix/             # Human-only reference (BUILD.md, SECURITY.md)
```

**Progressive disclosure loading**:
1. Metadata (name + description) - always in context
2. SKILL.md body - when skill triggers
3. Bundled resources - as needed by Claude

### Go Implementation Pattern

Both codex skills share common Go implementation:

**Core files** (in `scripts/`):
- `main.go`: Entry point, session management, argument parsing
- `api.go`: OpenAI API integration, conversation management
- `tools.go`: Tool implementations (Glob, Grep, Read)
- `secure.go`: Unix security (openat syscalls, symlink protection)
- `secure_windows.go`: Windows security (best-effort)
- `session.go`: Session persistence and state management
- `go.mod`: Dependencies

**Security model**:
- Unix (macOS/Linux): 9.5/10 - openat-based protection against symlink/TOCTOU attacks
- Windows: 7/10 - best-effort protection, WSL2 recommended for production

### Skill Types in This Repo

**codex-review**: READ-ONLY code analysis
- Analyzes security, bugs, performance, code quality
- Does NOT modify code
- Uses GPT-5.2-Codex model
- **Execution**: Background (can take 2-5 minutes)
- **Session format**: `review-{timestamp}-{random-hex}`

**codex-task-executor**: WRITE implementation
- Implements features from plans
- Modifies/creates files with full tool access
- Uses GPT-5.2-Codex model
- **Execution**: Background (can take 3-10 minutes)
- **Session format**: `task-{number}-{timestamp}-{random-hex}`

**skill-creator**: Meta-tool for skill development
- Provides templates and validation
- Scripts for init/package/validate
- **Execution**: Immediate (Python scripts)

## Skill Creation Principles

When creating or modifying skills:

1. **Concise is key**: Context window is a public good. Only add what Claude doesn't already know.

2. **Set appropriate degrees of freedom**:
   - High freedom (text): Multiple valid approaches
   - Medium freedom (pseudocode): Preferred patterns exist
   - Low freedom (specific scripts): Fragile operations requiring consistency

3. **SKILL.md body limit**: Keep under 500 lines. Split into references/ when approaching limit.

4. **Frontmatter description**: Primary triggering mechanism. Must include:
   - What the skill does
   - When to use it (triggers, scenarios, file types)
   - Key capabilities

5. **Progressive disclosure**: Reference detailed docs from SKILL.md, load only when needed.

## Background Task Lifecycle

**How background execution improves process management:**

### Without Background (Current Pain Point)
```
User: "Review these 3 files"
Claude Code: [Blocks for 15 minutes while codex reviews all files]
User: [Waits, cannot do other work]
Claude Code: [Finally returns results]
```

### With Background (Recommended)
```
User: "Review these 3 files"
Claude Code: [Starts 3 reviews in background, returns immediately]
             "Started 3 reviews in background (file1, file2, file3)..."
User: [Can continue working, ask other questions]
Claude Code: [Receives notification] "Review #1 complete: file1 has security issues..."
Claude Code: [Receives notification] "Review #2 complete: file2 looks good..."
Claude Code: [Receives notification] "Review #3 complete: file3 has performance issues..."
```

**Process management benefits**:
- Claude Code remains responsive during long operations
- Can handle multiple codex tasks concurrently
- User gets incremental updates instead of waiting for everything
- Errors in one task don't block others
- Can queue up next work while previous tasks run

### Monitoring Background Tasks

**Built-in**: Claude Code automatically receives completion notifications

**Manual check** (if needed):
```bash
# List running background tasks
/tasks

# Check output of specific task
TaskOutput(task_id="task-abc123")
```

## Session Management

Both codex skills use session persistence:

**Location**: `{repo}/.codex-sessions/` (project-isolated, git-ignored)

**Session ID format**: Use timestamp + random hex to avoid collisions:
```python
import time, random
session_id = f"review-{int(time.time())}-{random.randint(0x1000, 0xffff):04x}"
# → "review-1738224567-a3f9"
```

**Why**: Prevents session file collisions when multiple Claude Code instances run simultaneously.

## Environment Variables

Required for all codex skills:
- `OPENAI_API_KEY`: OpenAI API key

Optional configuration:
- `OPENAI_MODEL`: Default `gpt-5.2-codex`
- `REASONING_EFFORT`: `low`/`medium`/`high`/`xhigh` (default varies by skill)
- `MAX_ITERS`: Max tool iterations (default: 50)
- `REPO_ROOT`: Repository root (auto-detected from git)
- `STATE_DIR`: Session storage location (default: `{repo}/.codex-sessions`)

## Key Patterns

### Background Execution for Codex Skills (Recommended)

**CRITICAL**: Both codex skills can take several minutes to complete. Always use background execution unless there's a specific reason not to.

**Pattern for codex-review**:
```python
import time, random
session_id = f"review-{int(time.time())}-{random.randint(0x1000, 0xffff):04x}"

# Run in background
Bash(
    command=f'~/.claude/skills/codex-review/bin/codex-review-darwin-arm64 "{session_id}" "[context]"',
    run_in_background=True,
    description="Running code review"
)

# Inform user immediately
"Started code review in background. Will notify when complete."
```

**Pattern for codex-task-executor**:
```python
import time, random
task_id = f"task-1-{int(time.time())}-{random.randint(0x1000, 0xffff):04x}"

# Run in background
Bash(
    command=f'~/.claude/skills/codex-task-executor/bin/execute-task-darwin-arm64 "{task_id}" "[desc]" "plan.md"',
    run_in_background=True,
    description="Implementing feature"
)

# Inform user immediately
"Started Task #1 implementation in background. Will notify when complete."
```

**When completion notification arrives**:
1. Parse output for markers: `[CODEX_COMPLETE]`, `[FILES_MODIFIED]`, `[BLOCKED]`, `[QUESTION]`
2. Handle appropriately:
   - **Success**: Summarize results to user
   - **Needs input**: Ask user via AskUserQuestion, then re-run
   - **Error**: Diagnose and resolve, then re-run
3. Update task status if using TaskCreate/TaskUpdate

**Output marker handling guide**:

```python
# After completion notification
output = result.output

if "[CODEX_COMPLETE]" in output:
    # Success - parse and summarize
    files = extract_between(output, "[FILES_MODIFIED]", "[CODEX_COMPLETE]")
    "Task completed. Modified files: {files}"

elif "[QUESTION]" in output:
    # Needs user input
    question = extract_after(output, "[QUESTION]")
    # Ask user via AskUserQuestion
    # Re-run with same session-id and appended answer

elif "[BLOCKED]" in output:
    # Cannot proceed
    blocker = extract_after(output, "[BLOCKED]")
    # Resolve blocker (create file, set env var, etc.)
    # Re-run with same session-id

else:
    # Check exit code for errors
    if exit_code != 0:
        "Task failed with error. Check stderr for details."
```

**Benefits**:
- Non-blocking: Handle multiple tasks concurrently
- Better UX: User sees progress notifications
- Error resilience: Automatic failure detection and handling
- Parallel execution: Run multiple codex tasks simultaneously
- Continue working: Claude Code can plan next tasks while Codex implements

**Complete example with parallel execution**:
```python
# User: "Review auth.ts for security and implement the fixes"

# Step 1: Start review in background
review_id = f"review-{int(time.time())}-{random.randint(0x1000, 0xffff):04x}"
Bash(
    command=f'codex-review-darwin-arm64 "{review_id}" "Review auth.ts for security"',
    run_in_background=True
)
"Started security review in background..."

# Step 2: When review completes (notification arrives)
# Parse findings: SQL injection at line 45, weak password hashing at line 78

# Step 3: Start implementation in background
task_id = f"task-1-{int(time.time())}-{random.randint(0x1000, 0xffff):04x}"
Bash(
    command=f'execute-task-darwin-arm64 "{task_id}" "Fix SQL injection at line 45..." "plan.md"',
    run_in_background=True
)
"Started implementing security fixes in background..."

# Step 4: When implementation completes
"Security fixes implemented. Modified auth.ts to use parameterized queries and bcrypt for passwords."
```

### Context Preparation for Codex Skills

Codex operates in headless mode and requires complete context upfront:

**Check conversation history first**: Extract files, issues, focus areas from prior messages before asking questions.

**Rich context template**:
```
Code Review Request:

FILES:
- path/to/file.ts (primary)
- related/file.ts (dependency)

FOCUS: Security + Performance
- Specific concern 1
- Specific concern 2

CONTEXT:
- Relevant background
- Known issues or bugs
- Production incidents

PRIORITY: Critical issues first, then improvements
```

**Use Context7 for external dependencies**: When code uses external libraries (React, Express, FastAPI, etc.), query Context7 for latest best practices BEFORE invoking codex skills.

### Skill Validation

Before packaging, validation checks:
- YAML frontmatter format and required fields
- Skill naming conventions
- Description completeness
- File organization and references

Run explicitly: `.claude/skills/skill-creator/scripts/quick_validate.py skills/<skill-name>`

Or automatically via package script (includes validation).

## File Organization

```
claude-code-skills/
├── .claude/
│   └── skills/
│       └── skill-creator/    # Meta-tool for skill development
├── skills/                   # Installable production skills
│   ├── codex-review/
│   └── codex-task-executor/
├── .gitignore               # Excludes .codex-sessions/, binaries
└── README.md                # User-facing documentation
```

**What's git-ignored**:
- Session files: `.codex-sessions/`, `skills/*/.codex-sessions/`
- Python cache: `__pycache__/`, `*.pyc`
- Build artifacts: `skills/*/scripts/codex-review`, `skills/*/scripts/execute-task`
- Binaries in `bin/` are COMMITTED (pre-built for distribution)

## Development Guidelines

### GitHub Operations Policy

**IMPORTANT**: Always request user approval before performing GitHub operations:

- `git push` to remote repository
- Creating pull requests (`gh pr create`)
- Force push operations
- Modifying remote branches
- Creating or modifying GitHub issues
- Any other operations that affect the remote repository

**Workflow**:
1. Prepare the changes locally (commits, branches, etc.)
2. Explain what will be pushed/created and why
3. Ask for explicit user approval
4. Only proceed after receiving confirmation

## Important Notes

- **Background execution required**: Both codex skills take several minutes. Always use `run_in_background=True` for better UX
- **Binary distribution**: Pre-built macOS ARM64 binaries included in `bin/` for immediate use
- **Multi-platform support**: Build from source for Linux/Windows using BUILD.md instructions
- **Session isolation**: Each skill maintains separate session storage to prevent cross-contamination
- **Security**: Go implementation uses openat syscalls on Unix for symlink/TOCTOU protection
- **No package.json**: Pure Go and Python, no npm dependencies required
- **Parallel execution**: Background mode enables running multiple codex tasks concurrently

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
- **Duration**: 2-5 minutes per review
- **Session format**: `review-{timestamp}-{random-hex}`

**codex-task-executor**: WRITE implementation
- Implements features from plans
- Modifies/creates files with full tool access
- Uses GPT-5.2-Codex model
- **Duration**: 3-10 minutes per task
- **Session format**: `task-{number}-{timestamp}-{random-hex}`

**skill-creator**: Meta-tool for skill development
- Provides templates and validation
- Scripts for init/package/validate

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

## Session Management

Both codex skills use session persistence:

**Location**: `{repo}/.codex-sessions/` (project-isolated, git-ignored)

**Session naming**: Use plan file pattern (adjective-verb-noun) for readable, unique names.

**Examples**:
- `security-reviewing-turing`
- `auth-implementing-lovelace`
- `performance-auditing-knuth`

**Follow-up**: Reuse same session name to continue conversation.

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

### Implementation Approval Policy

**CRITICAL**: Before implementing any code changes, new features, or architectural modifications, ALWAYS:

1. **Explain the proposed approach**
   - What will be modified
   - How it will work
   - Why this approach was chosen

2. **Present alternatives** when multiple valid approaches exist
   - Compare pros/cons
   - Recommend preferred option

3. **Ask for explicit user approval**
   - Wait for confirmation
   - Do NOT assume approval

4. **Only implement after receiving approval**

**This applies to**:
- Modifying Go source code in skills (`scripts/*.go`)
- Adding new features or functionality
- Changing skill behavior or architecture
- Updating binary implementations
- Modifying SKILL.md instructions
- Adding new dependencies

**Does NOT apply to**:
- Documentation updates (README.md, CLAUDE.md)
- Bug fixes for obvious errors
- Formatting/style changes

**Example workflow**:
```
User: "백그라운드 완료되면 알림 보내줘"

Claude Code:
"macOS 알림을 추가할 수 있습니다. 제안하는 방법:

접근 1: Go 코드에 osascript 통합
- main.go에서 완료 시 osascript 실행
- NOTIFY_ON_COMPLETE 환경 변수로 제어
- 장점: 네이티브 macOS 알림, 추가 설치 불필요
- 단점: 바이너리 재빌드 필요

접근 2: Bash 서브에이전트 사용
- Task tool로 Bash 서브에이전트에게 위임
- 완료되면 자동으로 결과 반환
- 장점: 코드 수정 불필요, 자동화된 흐름
- 단점: 서브에이전트 오버헤드

어느 방법으로 구현할까요?"

[Wait for user approval]

User: "2번으로 해"

Claude Code: [Proceeds with approach 2]
```

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

- **Binary distribution**: Pre-built macOS ARM64 binaries included in `bin/` for immediate use
- **Multi-platform support**: Build from source for Linux/Windows using BUILD.md instructions
- **Session isolation**: Each skill maintains separate session storage to prevent cross-contamination
- **Security**: Go implementation uses openat syscalls on Unix for symlink/TOCTOU protection
- **No package.json**: Pure Go and Python, no npm dependencies required

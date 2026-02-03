# Claude Code Skills

Professional AI agent skills for Claude Code.

## Available Skills

| Skill | Description | Type |
|-------|-------------|------|
| [codex-review](skills/codex-review/) | Code review & security analysis using GPT-5.2-Codex | READ-ONLY |
| [codex-task-executor](skills/codex-task-executor/) | Coding task execution using GPT-5.2-Codex | WRITE |
| [unity-coding](skills/unity-coding/) | Unity hypercasual game development patterns | Guide |
| [pencil-to-code](skills/pencil-to-code/) | Pencil design to React/TypeScript conversion | Guide |

## Quick Start

### Using acodex Agent

```
# Code review
"Use acodex to review src/auth.ts for security issues"

# Implementation
"Use acodex to implement the login feature from the plan"

# Background execution
"Use acodex in the background to review the authentication module"
```

### Direct Skill Invocation

```
/codex-review
/codex-task-executor
/unity-coding
/pencil-to-code
```

## Installation

### OpenSkills (Recommended)

```bash
# Install all skills
npx openskills install jaewooseo-bagelcode/claude-code-skills
npx openskills sync

# Install specific skill
npx openskills install jaewooseo-bagelcode/claude-code-skills/skills/codex-review
```

### Manual

```bash
git clone https://github.com/jaewooseo-bagelcode/claude-code-skills.git
ln -s $(pwd)/claude-code-skills/skills/* ~/.claude/skills/
```

## Skills Overview

### codex-review

Professional code review using GPT-5.2-Codex (READ-ONLY).

- Security analysis (SQL injection, XSS, auth bypass)
- Bug detection (logic errors, null references)
- Performance review (N+1 queries, algorithm efficiency)
- Code quality (SOLID principles, anti-patterns)

**Tech**: Go binary, 9.5/10 security score

### codex-task-executor

Execute coding tasks using GPT-5.2-Codex (WRITES CODE).

- Implements features from plans
- Creates and modifies files
- Multi-turn conversations with progress markers
- Full Write/Edit/Read/Grep/Glob tools

**Tech**: Go binary, 9.5/10 security score

### unity-coding

Unity hypercasual game development patterns.

- System + Manager 패턴 (DI 프레임워크 없이)
- ObjectSystem 기반 풀링
- GC-free Update 패턴
- 모바일 빌드 최적화 (< 80MB CPI 테스트)
- Advanced: MonoBase, Component Pool, Async FSM

### pencil-to-code

Pencil design to React/TypeScript conversion.

- CSS variable → hex 변환
- Pixel-perfect spacing (Tailwind 근사 없음)
- Fixed height 정렬 패턴
- Complete property mapping table
- 17가지 트러블슈팅 가이드

## Requirements

**Codex skills require**:
- `OPENAI_API_KEY` environment variable
- macOS Apple Silicon (pre-built) or build from source

**Optional**:
| Variable | Default | Description |
|----------|---------|-------------|
| `OPENAI_MODEL` | `gpt-5.2-codex` | Model name |
| `REASONING_EFFORT` | `high` / `medium` | low/medium/high/xhigh |
| `MAX_ITERS` | `50` | Max tool iterations |

## Project Structure

```
claude-code-skills/
├── skills/
│   ├── codex-review/          # Go-based code review
│   ├── codex-task-executor/   # Go-based task execution
│   ├── unity-coding/          # Unity dev patterns
│   └── pencil-to-code/        # Design-to-code conversion
├── agents/
│   └── acodex.md              # Codex orchestrator agent
└── .claude/
    └── skills/
        └── skill-creator/     # Skill development tool
```

## Development

```bash
# Create new skill
.claude/skills/skill-creator/scripts/init_skill.py my-skill --path ./skills/my-skill

# Build Go binary
cd skills/<skill>/scripts
go build -ldflags="-s -w" -o ../bin/<name>-$(go env GOOS)-$(go env GOARCH)
```

## License

Each skill has its own license. See individual LICENSE files.

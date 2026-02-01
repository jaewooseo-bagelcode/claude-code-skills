# Claude Code Skills

Collection of professional AI agent skills for Claude Code, powered by GPT-5.2-Codex.

## Quick Start: Using acodex Agent

The `acodex` agent orchestrates Codex skills for code review and implementation. It prepares context, manages sessions, and summarizes results.

### Foreground Execution (Blocking)

```
# Code review
"Use acodex to review src/auth.ts for security issues"

# Implementation
"Use acodex to implement the login feature from the plan"
```

Claude delegates to acodex, which runs in the foreground. You'll see progress and can interact with questions.

### Background Execution (Concurrent)

```
# Run in background while you continue working
"Use acodex in the background to review the authentication module"

# Or press Ctrl+B while a task is running to background it
```

Background agents run concurrently. Claude notifies you when complete.

### Direct Skill Invocation

Skip the acodex orchestrator and invoke skills directly:

```
# Code review
/codex-review

# Implementation
/codex-task-executor
```

**Comparison:**

| Method | Overhead | Best For |
|--------|----------|----------|
| acodex agent | Haiku model cost | Complex workflows, multiple files, context preparation |
| Direct skill | None | Simple reviews, when you provide complete context |

Both execute the same Go binaries internally.

## Available Skills

### codex-review

Professional code review and analysis using GPT-5.2-Codex (READ-ONLY).

**Features**:
- Security analysis (SQL injection, XSS, auth bypass)
- Bug detection (logic errors, null references, edge cases)
- Performance review (N+1 queries, algorithm efficiency)
- Code quality (SOLID principles, anti-patterns)
- Refactoring suggestions

**Tech**: Go implementation, 9.5/10 security, 5.7MB binary

[View Details](skills/codex-review/SKILL.md)

### codex-task-executor

Execute coding tasks using GPT-5.2-Codex (WRITES CODE).

**Features**:
- Implements features from plans
- Creates and modifies files
- Multi-turn conversations with progress markers
- Works from Claude Code's task specifications
- Full Write/Edit/Read/Grep/Glob tools

**Tech**: Go implementation, 9.5/10 security, 8.3MB binary

[View Details](skills/codex-task-executor/SKILL.md)

## acodex Agent Configuration

The `acodex` agent is defined in `agents/acodex.md`:

```yaml
---
name: acodex
description: Codex orchestrator for deep code review and implementation using GPT-5.2-Codex.
tools: Bash, Read, Glob, Grep
skills:
  - codex-review
  - codex-task-executor
model: haiku
---
```

**Key features:**
- Uses Haiku model for cost-efficient orchestration
- Preloads codex-review and codex-task-executor skills
- Generates session names using plan file pattern (adjective-verb-noun)
- Manages session continuity for follow-up questions

**Session naming examples:**
- `security-reviewing-turing`
- `auth-implementing-lovelace`
- `performance-auditing-knuth`

## Installation

### Method 1: OpenSkills (Recommended)

```bash
# Install all skills
npx openskills install jaewooseo-bagelcode/claude-code-skills
npx openskills sync

# Or install to global (~/.claude/skills/)
npx openskills install jaewooseo-bagelcode/claude-code-skills --global
npx openskills sync
```

**Individual skills**:
```bash
# Install specific skill only
npx openskills install jaewooseo-bagelcode/claude-code-skills/skills/codex-review
npx openskills sync
```

**Update skills**:
```bash
# Update all
npx openskills update

# Update specific skills
npx openskills update codex-review,codex-task-executor
```

### Method 2: Manual Git Clone

```bash
# Global installation
git clone https://github.com/jaewooseo-bagelcode/claude-code-skills.git ~/.claude-skills-repo
ln -s ~/.claude-skills-repo/skills/* ~/.claude/skills/
ln -s ~/.claude-skills-repo/agents/* ~/.claude/agents/

# Project-local installation
git clone https://github.com/jaewooseo-bagelcode/claude-code-skills.git
ln -s $(pwd)/claude-code-skills/skills/* .claude/skills/
ln -s $(pwd)/claude-code-skills/agents/* .claude/agents/
```

## Requirements

**All skills require**:
- `OPENAI_API_KEY` environment variable

**Platform support**:
- macOS Apple Silicon (pre-built binaries included)
- Linux, Windows (build from source - see appendix/BUILD.md in each skill)

**Optional environment variables**:
| Variable | Default | Description |
|----------|---------|-------------|
| `OPENAI_MODEL` | `gpt-5.2-codex` | Model name |
| `REASONING_EFFORT` | `high` (review) / `medium` (executor) | low/medium/high/xhigh |
| `MAX_ITERS` | `50` | Max tool iterations |

## Project Structure

```
claude-code-skills/
├── agents/               # Custom agent definitions
│   └── acodex.md         # Codex orchestrator agent
├── skills/               # Installable skills
│   ├── codex-review/
│   │   ├── SKILL.md
│   │   ├── bin/          # Pre-built binaries
│   │   ├── scripts/      # Go source code
│   │   └── references/   # Additional documentation
│   └── codex-task-executor/
│       ├── SKILL.md
│       ├── bin/
│       ├── scripts/
│       └── references/
├── .claude/              # Development tools
│   └── skills/
│       └── skill-creator/
└── README.md
```

## Skills Included

| Skill | Description | Type | Status |
|-------|-------------|------|--------|
| [codex-review](skills/codex-review/) | Code review & analysis | READ-ONLY | Ready |
| [codex-task-executor](skills/codex-task-executor/) | Coding task execution | WRITE | Ready |

| Agent | Description | Model | Status |
|-------|-------------|-------|--------|
| [acodex](agents/acodex.md) | Codex orchestrator | Haiku | Ready |

## Development

### Adding New Skills

```bash
# Use skill-creator helper
.claude/skills/skill-creator/scripts/init_skill.py my-new-skill --path ./skills/my-new-skill

# Follow skill-creator guidelines
# See .claude/skills/skill-creator/SKILL.md
```

### Building Go Binaries

```bash
cd skills/<skill-name>/scripts
go mod download
go build -ldflags="-s -w" -o ../bin/<binary-name>-$(go env GOOS)-$(go env GOARCH)
```

See `appendix/BUILD.md` in each skill for platform-specific instructions.

## Contributing

1. Fork this repository
2. Create your skill in `skills/your-skill-name/`
3. Follow skill-creator guidelines
4. Submit pull request

## License

Each skill has its own license. See individual LICENSE files.

## Credits

Built for Claude Code agent framework.

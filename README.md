# Claude Code Skills

Collection of professional AI agent skills for Claude Code, powered by advanced AI models.

## Available Skills

### codex-review

Professional code review and analysis using GPT-5.2-Codex (READ-ONLY).

**Features**:
- 🔒 Security analysis (SQL injection, XSS, auth bypass)
- 🐛 Bug detection (logic errors, null references, edge cases)
- ⚡ Performance review (N+1 queries, algorithm efficiency)
- 📝 Code quality (SOLID principles, anti-patterns)
- 🔧 Refactoring suggestions

**Tech**: Go implementation, 9.5/10 security, 100x faster, 5.7MB binary

[View Details](skills/codex-review/SKILL.md)

### codex-task-executor

Execute coding tasks using GPT-5.2-Codex (WRITES CODE).

**Features**:
- ✍️ Implements features from plans
- 🔨 Creates and modifies files
- 🔄 Multi-turn conversations with progress markers
- 📝 Works from Claude Code's task specifications
- 🛠️ Full Write/Edit/Read/Grep/Glob tools

**Tech**: Go implementation, 9.5/10 security, 8.3MB binary

[View Details](skills/codex-task-executor/SKILL.md)

## Installation

### Method 1: Install from GitHub (Recommended)

```bash
# Install all skills
git clone https://github.com/jaewooseo-bagelcode/claude-code-skills.git ~/.claude-skills-repo
ln -s ~/.claude-skills-repo/skills/* ~/.claude/skills/

# Or install specific skill only
ln -s ~/.claude-skills-repo/skills/codex-review ~/.claude/skills/codex-review
```

### Method 2: OpenSkills Installation

Add to your OpenSkills configuration:

```json
{
  "marketplaces": [
    {
      "name": "personal-skills",
      "url": "https://github.com/jaewooseo-bagelcode/claude-code-skills"
    }
  ]
}
```

Then install:
```bash
/plugin install codex-review@personal-skills
```

### Method 3: Project-Local Installation

```bash
# Clone to project
git clone https://github.com/jaewooseo-bagelcode/claude-code-skills.git .claude-skills-repo
ln -s .claude-skills-repo/skills/codex-review .claude/skills/codex-review
```

## Requirements

**All skills require**:
- `OPENAI_API_KEY` environment variable

**Platform support**:
- ✅ macOS Apple Silicon (pre-built binaries included)
- ✅ Linux, Windows (build from source - see appendix/BUILD.md in each skill)

## Development

### Adding New Skills

```bash
# Use skill-creator helper
.claude/skills/skill-creator/scripts/init_skill.py my-new-skill --path ./skills/my-new-skill

# Follow skill-creator guidelines
# See .claude/skills/skill-creator/SKILL.md
```

### Structure

```
claude-code-skills/
├── .claude/              # Development tools
│   └── skills/
│       └── skill-creator/
├── skills/               # Installable skills
│   ├── codex-review/
│   └── [future skills]/
└── README.md            # This file
```

## Skills Included

| Skill | Description | Type | Status |
|-------|-------------|------|--------|
| [codex-review](skills/codex-review/) | Code review & analysis | READ-ONLY | ✅ Ready |
| [codex-task-executor](skills/codex-task-executor/) | Coding task execution | WRITE | ✅ Ready |

## Contributing

1. Fork this repository
2. Create your skill in `skills/your-skill-name/`
3. Follow skill-creator guidelines
4. Submit pull request

## License

Each skill has its own license. See individual LICENSE files.

## Credits

Built for Claude Code agent framework.

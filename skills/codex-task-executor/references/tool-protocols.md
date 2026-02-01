# Tool Communication Protocols

This document specifies the output marker protocol used by Codex to communicate with Claude Code during task execution.

## Overview

Since Codex cannot use interactive tools like `ask_user` or `report_progress` (Responses API limitation), it uses **structured output markers** in its text output to communicate status, questions, and results.

Claude Code monitors stdout in real-time and parses these markers to:
- Update task progress UI
- Handle user questions
- Track file modifications
- Detect completion or blocking issues

---

## Marker Format

All markers follow this format:
```
[MARKER_NAME] optional content
```

**Rules:**
- Must appear at start of line (no leading whitespace for reliable parsing)
- Marker name in UPPERCASE, wrapped in square brackets
- Optional single space after closing bracket (marker may appear alone on line)
- Content continues to end of line or until next marker/blank line
- For multi-line markers (QUESTION, BLOCKED, FILES_MODIFIED), content continues until blank line or next marker

---

## Marker Specifications

### 1. [PROGRESS]

**Purpose**: Report progress on current work

**Format:**
```
[PROGRESS] Brief description of what was just done or is being done
```

**Examples:**
```
[PROGRESS] Analyzed existing LoginForm.tsx pattern
[PROGRESS] Created src/components/UserAuth.tsx with basic structure
[PROGRESS] Implementing JWT validation in src/lib/jwt.ts
[PROGRESS] Integration with App.tsx complete
```

**Claude Code Handling:**
- Display in UI as task progress
- Optionally update task activeForm
- Log to task history

**Frequency**: After each major step (file creation, significant logic implementation)

---

### 2. [QUESTION]

**Purpose**: Ask for clarification when specification is ambiguous

**Format:**
```
[QUESTION] The question with context?
Options:
1. First option - Description
2. Second option - Description
[Additional context about why asking]
```

**Examples:**
```
[QUESTION] Should JWT tokens be stored in localStorage or sessionStorage?
Options:
1. localStorage - Persists across browser sessions
2. sessionStorage - Cleared when browser closes

This affects logout behavior and security posture.
```

```
[QUESTION] Where should I place the error message component?
Options:
1. Inline in form - Below each input field
2. Top of form - Single error banner
3. Toast notification - Floating alert

LoginForm uses option 1, but this form is more complex.
```

**Claude Code Handling:**
1. **Parse question and options**
2. **Stop script execution** (Ctrl+C or wait for completion)
3. **Ask user** via AskUserQuestion tool
4. **Re-invoke script** with answer:
   ```bash
   ./execute-task.py "same-task-id" \
     "Previous description + Answer: Use option 1 (localStorage)" \
     "same-plan.md"
   ```

**Important Notes:**
- Codex should **continue working** on non-blocked parts after asking
- Claude Code can batch multiple questions if Codex asks several
- Same task-id ensures conversation continuity

---

### 3. [BLOCKED]

**Purpose**: Report complete blockage - cannot proceed at all

**Format:**
```
[BLOCKED] Clear description of what blocks progress
[Specific information needed or issue to resolve]
```

**Examples:**
```
[BLOCKED] Cannot find API base URL
The plan references API_BASE_URL but it's not defined in:
- .env file
- src/config.ts
- Environment variables
Please specify the API endpoint.
```

```
[BLOCKED] Missing dependency: axios
The plan requires making HTTP requests but axios is not installed.
Should I:
- Use native fetch instead?
- Add axios to package.json?
```

**Claude Code Handling:**
1. **Notify user** of blocker
2. **Resolve issue**:
   - Add missing info to plan
   - Create required config file
   - Install dependency
   - Update task description
3. **Re-invoke** with resolution

**Difference from [QUESTION]:**
- QUESTION: Can work around or continue partially
- BLOCKED: Completely stuck, zero progress possible

---

### 4. [FILES_MODIFIED]

**Purpose**: Summary of all files changed during execution

**Format:**
```
[FILES_MODIFIED]
- path/to/file1.ext (created|modified|deleted)
- path/to/file2.ext (created|modified|deleted)
```

**Examples:**
```
[FILES_MODIFIED]
- src/components/UserAuth.tsx (created)
- src/lib/jwt.ts (created)
- src/App.tsx (modified)
- src/types/user.ts (modified)
- tests/auth.test.tsx (created)
```

**Claude Code Handling:**
- Parse list of files
- Store in task metadata
- Display to user as summary
- Optionally verify files exist
- Use for git commit message generation

**Placement**: Typically at the end, before [CODEX_COMPLETE]

---

### 5. [CODEX_COMPLETE]

**Purpose**: Signal task completion

**Format:**
```
[CODEX_COMPLETE] Task completed in N iterations
```

**Examples:**
```
[CODEX_COMPLETE] Task completed in 8 iterations
[CODEX_COMPLETE] Task completed in 23 iterations
```

**Claude Code Handling:**
- Mark task as completed
- Update task status
- Move to next task in queue
- Archive session (optional)

**Automatic**: Script adds this when no more tool calls needed

---

## Parsing Implementation (Claude Code Side)

### Python Example
```python
import re

def parse_markers(output: str) -> dict:
    markers = {
        "progress": [],
        "questions": [],
        "blocked": [],
        "files_modified": [],
        "completed": False
    }

    lines = output.split('\n')
    i = 0
    while i < len(lines):
        line = lines[i].strip()

        if line.startswith('[PROGRESS]'):
            markers["progress"].append(line[10:].strip())

        elif line.startswith('[QUESTION]'):
            question = line[10:].strip()
            # Collect following lines for options
            options = []
            i += 1
            while i < len(lines) and (lines[i].strip().startswith(('1.', '2.', '3.', 'Options:'))):
                options.append(lines[i].strip())
                i += 1
            markers["questions"].append({"question": question, "options": options})
            continue

        elif line.startswith('[BLOCKED]'):
            blocked = line[9:].strip()
            # Collect following context lines
            context = []
            i += 1
            while i < len(lines) and lines[i].strip() and not lines[i].strip().startswith('['):
                context.append(lines[i].strip())
                i += 1
            markers["blocked"].append({"reason": blocked, "context": context})
            continue

        elif line.startswith('[FILES_MODIFIED]'):
            i += 1
            while i < len(lines) and lines[i].strip().startswith('-'):
                file_line = lines[i].strip()[1:].strip()
                markers["files_modified"].append(file_line)
                i += 1
            continue

        elif line.startswith('[CODEX_COMPLETE]'):
            markers["completed"] = True

        i += 1

    return markers
```

### TypeScript Example
```typescript
interface ParsedMarkers {
  progress: string[];
  questions: Array<{ question: string; options: string[] }>;
  blocked: Array<{ reason: string; context: string[] }>;
  filesModified: string[];
  completed: boolean;
}

function parseMarkers(output: string): ParsedMarkers {
  const markers: ParsedMarkers = {
    progress: [],
    questions: [],
    blocked: [],
    filesModified: [],
    completed: false
  };

  const lines = output.split('\n');

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();

    if (line.startsWith('[PROGRESS]')) {
      markers.progress.push(line.slice(10).trim());
    }
    else if (line.startsWith('[QUESTION]')) {
      const question = line.slice(10).trim();
      const options: string[] = [];

      // Collect option lines
      while (i + 1 < lines.length) {
        const next = lines[i + 1].trim();
        if (next.match(/^\d+\./) || next === 'Options:') {
          options.push(next);
          i++;
        } else break;
      }

      markers.questions.push({ question, options });
    }
    else if (line.startsWith('[BLOCKED]')) {
      const reason = line.slice(9).trim();
      const context: string[] = [];

      while (i + 1 < lines.length) {
        const next = lines[i + 1].trim();
        if (next && !next.startsWith('[')) {
          context.push(next);
          i++;
        } else break;
      }

      markers.blocked.push({ reason, context });
    }
    else if (line.startsWith('[FILES_MODIFIED]')) {
      while (i + 1 < lines.length) {
        const next = lines[i + 1].trim();
        if (next.startsWith('-')) {
          markers.filesModified.push(next.slice(1).trim());
          i++;
        } else break;
      }
    }
    else if (line.startsWith('[CODEX_COMPLETE]')) {
      markers.completed = true;
    }
  }

  return markers;
}
```

---

## Streaming Monitoring (Real-time)

For real-time progress updates, Claude Code can monitor stdout as script runs:

```typescript
import { spawn } from 'child_process';

function executeTaskWithMonitoring(
  taskId: string,
  description: string,
  planFile: string
) {
  const proc = spawn('python3', [
    scriptPath,
    taskId,
    description,
    planFile
  ]);

  let buffer = '';

  proc.stdout.on('data', (data) => {
    buffer += data.toString();

    // Check for complete markers in buffer
    const lines = buffer.split('\n');
    buffer = lines.pop() || ''; // Keep incomplete line

    for (const line of lines) {
      const trimmed = line.trim();

      if (trimmed.startsWith('[PROGRESS]')) {
        updateTaskProgress(trimmed);
      }
      if (trimmed.startsWith('[QUESTION]')) {
        handleQuestion(trimmed);
      }

      // Show output to user
      console.log(line);
    }
  });

  proc.on('close', (code) => {
    if (code === 0) {
      markTaskCompleted();
    } else {
      handleError();
    }
  });
}
```

---

## Best Practices

### For Codex (in system-prompt.md)

1. **Report progress regularly**: After each file operation or major step
2. **Ask specific questions**: Provide options when possible
3. **Continue working**: Don't wait for answers if other work possible
4. **Be honest about blocks**: Use [BLOCKED] when truly stuck
5. **Summarize changes**: Always include [FILES_MODIFIED] at end

### For Claude Code (when invoking)

1. **Provide rich context**: Detailed task description with examples
2. **Include existing patterns**: Mention files to read for style
3. **Handle questions promptly**: Parse and ask user via AskUserQuestion
4. **Monitor in real-time**: Stream stdout for immediate feedback
5. **Verify completion**: Check [FILES_MODIFIED] files actually exist

---

## Why This Protocol?

### Limitations Addressed

**Responses API constraints:**
- No auto-tool-execution (unlike Chat Completions `runTools`)
- No built-in progress callbacks
- No interactive user input mid-execution

**Our solution:**
- Text markers = Universal, parseable, human-readable
- Simple regex parsing = Reliable extraction
- Stdout streaming = Real-time monitoring
- Session continuity = Multi-turn support

### Benefits

✅ **Simple**: Plain text, easy to parse
✅ **Robust**: Doesn't break if marker malformed
✅ **Real-time**: Streaming stdout for immediate feedback
✅ **Debuggable**: Human-readable output
✅ **Flexible**: Easy to add new markers

---

## Future Extensions

Potential additional markers:

- `[TEST_RESULTS]` - Codex runs tests and reports results
- `[LINT_WARNINGS]` - Codex detects linting issues
- `[DEPENDENCY_NEEDED]` - Codex needs npm/pip package
- `[GIT_COMMIT_SUGGESTION]` - Codex suggests commit message

Add these to system-prompt.md and update parser as needed.

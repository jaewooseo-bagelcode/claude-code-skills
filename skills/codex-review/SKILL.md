---
name: codex-review
description: Professional code review and analysis using GPT-5.2-Codex (READ-ONLY, never modifies code). Analyzes bugs, security vulnerabilities, performance issues, and code quality. Provides detailed reports with actionable suggestions but does NOT implement fixes. Use when the user wants to understand code issues, find bugs, or get improvement suggestions. Triggers on phrases like "review this code", "analyze this", "find bugs in", "what's wrong with", "check security of", "audit this code", "is this code safe", "identify issues". NOT for implementing fixes - use codex-task-executor for that.
---

# Instructions

Execute Codex-powered code review with complete context preparation.

**IMPORTANT: This skill provides READ-ONLY analysis.** It identifies issues and provides suggestions but does NOT modify code. For implementing fixes, use `codex-task-executor`.

## Invocation

### Recommended: Background Execution

**IMPORTANT**: Code reviews can take several minutes. Use background execution for better UX.

**Use Bash tool with `run_in_background=true`:**

```python
# This allows Claude Code to continue working while review runs
Bash(
    command='~/.claude/skills/codex-review/bin/codex-review-darwin-arm64 "review-123" "..."',
    run_in_background=True,
    description="Running code review in background"
)
# Returns immediately with task_id
```

**Benefits**:
- ✅ Non-blocking: Claude Code can handle other tasks
- ✅ Completion notification: Automatic alert when review finishes
- ✅ Error handling: Catches failures and shows to user
- ✅ Better UX: User sees progress, not frozen terminal

### Foreground Execution (Rare Cases Only)

**Use foreground only when**:
- Follow-up question in existing review session (quick response expected)
- User explicitly requests immediate/interactive review
- Single file, very focused review (< 100 lines)

```bash
~/.claude/skills/codex-review/bin/codex-review-darwin-arm64 "{session-id}" "{review-prompt}"
```

**Warning**: Foreground execution blocks Claude Code for entire review duration (2-5 minutes). User cannot interact during this time.

**Session ID**: Generate unique ID per review using `review-{timestamp}-{random-hex}` format. See Session Management section for generation code.

**Go Implementation Benefits**:
- ✅ 10-100x faster performance
- ✅ 9.5/10 security on macOS/Linux - perfect symlink/TOCTOU protection via openat
- ✅ 5.7MB single binary, zero runtime dependencies
- ✅ 30 minute max timeout for deep analysis
- ✅ For other platforms, see [appendix/BUILD.md](appendix/BUILD.md)

## Context Preparation (Critical)

**Codex operates in headless execution mode and requires complete context upfront.**

### Use Conversation Context

**IMPORTANT: Check the conversation history before asking questions.**

If the user has already provided context in previous messages:
- Files mentioned or read in conversation
- Issues or bugs discussed
- Code snippets shared
- Error messages or logs

**Extract and use this information automatically.**

**Example conversation:**
```
User: "I'm getting SQL injection warnings in auth.ts"
User: "The login function at line 45 looks suspicious"
User: "Can you use codex to review it?"

You (Claude Code):
[Don't ask - you already have context!]
→ File: auth.ts
→ Focus: Security (SQL injection)
→ Specific: login function, line 45
→ Invoke with complete context immediately
```

### When Context is Missing

Before invoking codex-review, YOU (Claude Code) must gather and provide:

### 1. Files to Review (Required)
- Specific file paths (e.g., `src/auth.ts`)
- Use Read tool to preview files if needed
- Identify related files (imports, tests, middleware)

### 2. Focus Area (Required)
Specify what aspects to prioritize:
- 🔒 **Security**: SQL injection, XSS, auth bypass, data exposure
- 🐛 **Bugs**: Logic errors, null references, type issues, edge cases
- ⚡ **Performance**: N+1 queries, algorithm efficiency, memory leaks
- 📝 **Code Quality**: Readability, naming, SOLID principles, duplication
- 🔧 **Refactoring**: Structure improvements, design patterns
- 📋 **Comprehensive**: All aspects (default if unclear)

### 3. Scope (Required)
Define review boundary:
- Single file only
- File + related dependencies
- File + tests
- Entire module/directory

### 4. Context (Optional but Recommended)
- Specific bug or issue user is facing
- Recent changes (git diff if available)
- Production incidents or error logs
- Performance concerns or metrics
- Security vulnerabilities suspected

### 5. External Dependencies Context (Use Context7)

**CRITICAL: When code uses external libraries/frameworks, fetch latest documentation BEFORE invoking.**

Per anti-pattern rule #3: "외부 의존성... Context7나 WebSearch를 통해서 메뉴얼 탐독"

**Workflow:**
```
1. Detect libraries in code (React, FastAPI, Express, etc.)
2. Use Context7 to get latest best practices
3. Include in prompt to Codex

Example:
- Code uses React hooks → Query Context7 for React 19 best practices
- Code uses FastAPI → Query Context7 for FastAPI security guidelines
- Code uses SQL → Query Context7 for SQL injection prevention
```

**Template:**
```bash
~/.claude/skills/codex-review/bin/codex-review-darwin-arm64 "review-123" "
Code Review Request:

FILES:
- src/Component.tsx (React component)

FOCUS: Best practices compliance + Security

EXTERNAL DEPENDENCIES:
- React 19 (latest from Context7):
  * Use new 'use' hook for data fetching
  * Avoid deprecated componentDidMount
  * Server Components best practices: ...

CONTEXT:
- User concerned about following latest React patterns
- Check for deprecated APIs and security issues

PRIORITY: Modern React compliance, then security
"
```

## Invocation Pattern

### ❌ Bad (Vague Context)
```bash
~/.claude/skills/codex-review/bin/codex-review-darwin-arm64 "review-123" "Review auth.ts"
```

### ✅ Good (Complete Context)
```bash
~/.claude/skills/codex-review/bin/codex-review-darwin-arm64 "review-123" "Review src/auth.ts for security vulnerabilities, specifically SQL injection and authentication bypass. Also check related files: src/middleware/auth.ts and src/routes/user.ts. Context: Production incident where user reported unauthorized access. Focus on login and session management logic."
```

### ✅ Better (Structured Context)
```bash
~/.claude/skills/codex-review/bin/codex-review-darwin-arm64 "review-123" "
Code Review Request:

FILES:
- src/auth/login.ts (primary)
- src/middleware/session.ts (related)
- tests/auth.test.ts (tests)

FOCUS: Security (Critical) + Bugs (High)
- SQL injection vulnerabilities
- Authentication bypass risks
- Session management flaws

SCOPE: Login flow and session handling

CONTEXT:
- Recent production incident: unauthorized access via /api/login
- Error logs show suspicious query patterns
- User reported being able to access other accounts

PRIORITY: Critical security issues first, then logic bugs
"
```

## Session Management

Generate unique session ID per review:
```python
import time, random
session_id = f"review-{int(time.time())}-{random.randint(0x1000, 0xffff):04x}"
```

**Reuse same session ID** for follow-up questions in the same review.

**Example workflows:**
```bash
# Initial review
~/.claude/skills/codex-review/bin/codex-review-darwin-arm64 "review-1738224567-a3f9" "[detailed context]"

# Follow-up question (same session)
~/.claude/skills/codex-review/bin/codex-review-darwin-arm64 "review-1738224567-a3f9" "How to fix the SQL injection in login.ts:45?"

# New review = new session
~/.claude/skills/codex-review/bin/codex-review-darwin-arm64 "review-1738224590-b2d1" "[new review context]"
```

## Context Construction Workflow

### Step 1: Check Conversation History

Extract from previous messages:
- **Files**: Any file paths mentioned, code read, or files discussed
- **Issues**: Bugs, errors, performance problems mentioned
- **Focus**: Security concerns, logic errors, performance issues discussed
- **Code locations**: Specific functions, lines, or areas mentioned

### Step 2: Determine if Context is Sufficient

**Sufficient context = Can invoke immediately:**
- ✅ File path identified
- ✅ General focus area clear (even if not specific)
- ✅ Some context about why review is needed

**Insufficient context = Ask questions:**
- ❌ No file identified
- ❌ No indication of what to focus on
- ❌ Zero context

### Step 3A: If Context is Sufficient - Build Rich Prompt

Construct detailed prompt from conversation context:

```python
# Example prompt construction
prompt = f"""
Code Review Request:

FILES:
- {file_from_conversation} (primary)
{related_files_if_discussed}

FOCUS: {inferred_from_conversation}
{specific_issues_mentioned}

CONTEXT:
{summarize_relevant_conversation_context}

{any_specific_concerns_or_requirements}
"""
```

### Step 3B: If Context is Insufficient - Ask First

If user's request is vague **and conversation has no context**, ask clarifying questions:

```
User: "Review auth.ts"

You (Claude Code):
"I can help review auth.ts with Codex. To provide the most valuable analysis, I need:

1. **Focus**: Which aspect is most important?
   - Security vulnerabilities (SQL injection, XSS, auth bypass)
   - Bugs and logic errors
   - Performance issues
   - Code quality and maintainability
   - All aspects (comprehensive review)

2. **Scope**: Should I also review related files?
   - Just auth.ts
   - Include imported dependencies
   - Include tests

3. **Context**: Any specific concerns?
   - Known bugs or issues?
   - Recent production problems?
   - Specific functionality that's failing?"

[Wait for user answers]

[Then build complete context and invoke skill]
```

## Environment

**Required**: `OPENAI_API_KEY`
**Optional**: `REASONING_EFFORT` (low/medium/high/xhigh, default: high)
**Sessions**: `{project}/.codex-sessions/` (project-isolated, auto-cleanup)

## Analysis Framework

Codex analyzes code across 5 dimensions:

- 🐛 **Bugs & Debugging** (Critical): Logic errors, type mismatches, null references, runtime issues
- 🔒 **Security** (High): Injections, XSS, auth flaws, data exposure
- ⚡ **Performance** (Medium): Algorithm efficiency, N+1 queries, memory leaks
- 📝 **Code Quality** (Low): Readability, naming, duplication, SOLID principles
- 🔧 **Refactoring**: Structural improvements, design patterns, abstractions

## Tools Available to Codex

- **Glob**: File pattern search (`src/**/*.ts`)
- **Grep**: Code pattern search
- **Read**: File reading with line ranges

## Complete Workflow Examples

### Example 1: Rich Conversation Context (Background Execution)

```
[Earlier in conversation]
User: "I'm debugging auth.ts"
User: "The login function is throwing SQL errors"
User: "Line 45 looks suspicious - it's building queries with string concat"

[Current request]
User: "Can you use codex to review this?"

You (Claude Code):
[Extract context from conversation]
- File: auth.ts
- Focus: Security (SQL injection from string concatenation)
- Location: Line 45, login function
- Issue: SQL errors in production

[Generate unique session ID]
session_id = f"review-{int(time.time())}-{random.randint(0x1000, 0xffff):04x}"

[Invoke in background with rich context]
Bash(
    command='~/.claude/skills/codex-review/bin/codex-review-darwin-arm64 "review-1738224567-a3f9" "
Code Review Request:

FILES:
- auth.ts (primary - login function)

FOCUS: Security (CRITICAL)
- SQL injection vulnerability
- String concatenation in query building (line 45)

CONTEXT:
- Production SQL errors
- Login function at line 45 uses string concatenation
- Need to check for SQL injection vulnerabilities

PRIORITY: Security vulnerabilities first, then code quality
"',
    run_in_background=True,
    description="Reviewing auth.ts for security"
)

[Tell user immediately]
"Started security review of auth.ts in background, focusing on the SQL injection vulnerability at line 45. Will notify when complete."

[When completion notification arrives]
[Parse output and summarize key findings]
"Code review complete. Found critical SQL injection vulnerability in login function (line 45). The query uses string concatenation instead of parameterized queries. Recommendation: Use prepared statements with parameterized queries. Would you like me to implement the fix?"
```

### Example 2: Minimal Context - Ask First

```
User: "코드 리뷰해줘"

You (Claude Code):
[No context in conversation - ask first]

"I can help with code review using Codex. To provide the most valuable analysis:

1. **Which file(s)** should I review?
2. **What aspect** is most important?
   - Security, Bugs, Performance, Code Quality, or Comprehensive
3. **Any specific concerns** or issues you're facing?"

[Wait for response, then build context]
```

### Example 3: Partial Context - Infer and Proceed

```
[Earlier in conversation]
User: "Can you read src/auth/login.ts?"
[You read the file]

[Current request]
User: "Review this with codex"

You (Claude Code):
[Infer from context]
- File: src/auth/login.ts (just read)
- Focus: Comprehensive (not specified, default to all aspects)
- Context: User wanted to review after reading

[Invoke with reasonable assumptions]
~/.claude/skills/codex-review/bin/codex-review-darwin-arm64 "review-123" "
Code Review Request:

FILES:
- src/auth/login.ts (primary)

FOCUS: Comprehensive review
- Security, bugs, performance, code quality

CONTEXT:
- File was just reviewed in conversation
- Looking for general code quality and potential issues

PRIORITY: Security and bugs first, then performance and quality
"
```

## Background Execution Workflow

**Recommended pattern for all reviews:**

1. **Start review in background**:
```python
Bash(
    command='~/.claude/skills/codex-review/bin/codex-review-darwin-arm64 "review-123" "[context]"',
    run_in_background=True
)
```

2. **Inform user**:
```
"Starting code review in background. This may take 2-5 minutes depending on complexity. I'll notify you when complete."
```

3. **When completion notification arrives**:
   - Parse output for issues found
   - Summarize key findings for user
   - Offer to explain details or fix issues

4. **Handle errors gracefully**:
   - If review fails, check exit code and stderr
   - Provide actionable error message to user

## Best Practices

1. **Use conversation context**: Don't ask if you already know
2. **Run in background**: Default to background execution for better UX
3. **Fetch latest docs with Context7**: When code uses external libraries, query Context7 BEFORE invoking
4. **Preview files**: Use Read tool to check file content and detect dependencies
5. **Identify related files**: Check imports, dependencies, tests
6. **Provide git diff**: If reviewing changes, include diff in context
7. **Be specific**: "Security audit for SQL injection" > "Review this"
8. **Batch related files**: Review login.ts + middleware.ts together rather than separately
9. **Default to comprehensive**: If focus unclear but file is clear, do comprehensive review

### When to Use Context7

**AUTOMATICALLY query Context7 when code uses external libraries.**

Detect and fetch docs for:
- UI frameworks: React, Vue, Angular, Svelte
- Backend frameworks: FastAPI, Express, Django, Rails
- Databases: PostgreSQL, MongoDB, Redis
- Security libraries: JWT, OAuth, bcrypt
- Any external dependency for best practices/security guidelines

**Workflow (AUTOMATIC):**
```bash
# Step 1: Read file and detect imports automatically
Read src/api/auth.ts
# → Detects: import express, jsonwebtoken, bcrypt

# Step 2: Automatically query Context7 for each detected library
Context7: "Express.js security best practices 2026"
Context7: "JWT token validation security guidelines"
Context7: "bcrypt password hashing best practices"

# Step 3: Build enriched prompt with Context7 results
~/.claude/skills/codex-review/bin/codex-review-darwin-arm64 "review-123" "
FILES: src/api/auth.ts

FOCUS: Security

EXTERNAL DEPENDENCIES (from Context7):
- Express.js: Use helmet middleware, validate inputs, prevent injection...
- JWT: Verify signature, check expiration, use strong secret...
- bcrypt: Use saltRounds >= 12, async methods only...

Check code compliance with these latest guidelines.
"
```

**This process should happen automatically** - don't ask user if they want Context7 docs.

## Reference Materials

**Load these when needed for better review quality:**

- **Security**: [references/common-vulnerabilities.md](references/common-vulnerabilities.md) - Security patterns and vulnerability examples
- **Code Quality**: [references/code-quality-patterns.md](references/code-quality-patterns.md) - Anti-patterns and best practices

## Appendix

*Human reference only (not for Claude):*

- Build guide: [appendix/BUILD.md](appendix/BUILD.md)
- Security analysis: [appendix/SECURITY.md](appendix/SECURITY.md)

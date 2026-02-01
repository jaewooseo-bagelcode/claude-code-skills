# Build Guide

This skill includes pre-built binaries for macOS Apple Silicon. For other platforms, follow the build instructions below.

## Pre-built Binary

**Included:**
- `bin/execute-task-darwin-arm64` - macOS Apple Silicon (M1/M2/M3)

**Usage:**
```bash
./bin/execute-task-darwin-arm64 "task-id" "description" "plan.md"
```

---

## Building from Source

For other platforms or custom builds, compile from the included Go source code.

### Requirements

- **Go 1.22 or higher** ([download](https://go.dev/dl/))
- Internet connection (for downloading dependencies on first build)

### Build Instructions

#### 1. Navigate to scripts directory
```bash
cd ~/.claude/skills/codex-task-executor/scripts
```

#### 2. Build for your platform
```bash
# Auto-detect platform
go build -o ../bin/execute-task

# Or specify output name
go build -o ../bin/execute-task-$(uname -s)-$(uname -m)
```

#### 3. Make executable (Unix)
```bash
chmod +x ../bin/execute-task
```

### Platform-Specific Builds

#### macOS Intel
```bash
GOARCH=amd64 go build -o ../bin/execute-task-darwin-amd64
```

#### Linux x86_64
```bash
GOOS=linux GOARCH=amd64 go build -o ../bin/execute-task-linux-amd64
```

#### Linux ARM64 (Raspberry Pi, etc.)
```bash
GOOS=linux GOARCH=arm64 go build -o ../bin/execute-task-linux-arm64
```

#### Windows
```bash
GOOS=windows GOARCH=amd64 go build -o ../bin/execute-task-windows-amd64.exe
```

### Cross-Compilation

Build all platforms at once:
```bash
#!/bin/bash
cd scripts

platforms=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

for platform in "${platforms[@]}"; do
  GOOS=${platform%/*}
  GOARCH=${platform#*/}
  output="../bin/execute-task-${GOOS}-${GOARCH}"

  if [ "$GOOS" = "windows" ]; then
    output="${output}.exe"
  fi

  echo "Building for $GOOS/$GOARCH..."
  GOOS=$GOOS GOARCH=$GOARCH go build -o "$output"

  if [ $? -eq 0 ]; then
    echo "✅ Built: $output"
  else
    echo "❌ Failed: $GOOS/$GOARCH"
  fi
done
```

---

## Dependencies

The build will automatically download:
- `github.com/openai/openai-go/v3` (OpenAI SDK)
- `golang.org/x/sys` (System calls for Unix security)

These are downloaded once and cached in `$GOPATH/pkg/mod`.

---

## Verifying the Build

After building, test the binary:

```bash
# Check binary info
file bin/execute-task
# → should show: "Mach-O 64-bit executable" or "ELF 64-bit executable"

# Test execution (requires OPENAI_API_KEY)
export OPENAI_API_KEY="sk-..."
./bin/execute-task --help 2>&1 || echo "Usage info shown above"
```

Expected: Usage message or error about missing arguments (not "command not found").

---

## Build Troubleshooting

### "go: command not found"
**Solution:** Install Go from https://go.dev/dl/

### "go.mod not found"
**Solution:** You're in the wrong directory. Must be in `scripts/` directory.

### "cannot find package"
**Solution:** Run `go mod download` to fetch dependencies.

### Build fails on old Go version
**Solution:** Upgrade to Go 1.22+
```bash
go version  # Check version
# If < 1.22, download newer version
```

---

## Binary Size

Typical sizes:
- macOS/Linux: **8-9MB**
- Windows: **8-9MB**

The binary is statically linked and includes all dependencies.

---

## Notes for AI Agents

If you're an AI agent (Claude, Codex, etc.) helping a user build this:

1. Check if Go is installed: `go version`
2. If not, guide user to install Go 1.22+
3. Navigate to `scripts/` directory
4. Run `go build -o ../bin/execute-task`
5. Verify binary exists and is executable
6. Test with a simple task

Do not attempt to modify the Go source unless specifically requested. The code is production-ready and thoroughly reviewed.

---

## Security Notes

See `SECURITY.md` for platform-specific security guarantees:
- **Unix platforms**: Perfect security with openat syscalls
- **Windows**: Best-effort security with limitations

For production workloads on Windows, consider using WSL2 to get Unix-level security.

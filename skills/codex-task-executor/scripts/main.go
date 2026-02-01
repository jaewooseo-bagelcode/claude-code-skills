package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

const (
	defaultMaxResults   = 200
	defaultMaxReadLines = 400
	defaultMaxIters     = 50
	maxGrepFileSize     = 2 * 1024 * 1024 // 2MB
)

var (
	// Security patterns
	denyBasenamesRE = regexp.MustCompile(`(^\.env$|^\.env\..+|^id_rsa$|^id_rsa\..+|^known_hosts$|^config$|^credentials$|^\.npmrc$|^\.pypirc$|^\.netrc$|^secrets$|^secrets\..+)`)
	denyExtRE       = regexp.MustCompile(`(?i)(\.pem$|\.key$|\.p12$|\.pfx$|\.cer$|\.crt$|\.der$|\.kdbx$|\.tfstate$|\.tfvars$)`)
	denyPathRE      = regexp.MustCompile(`(^|/)\.git(/|$)|\.docker/config\.json$`)
	safeSessionRE   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
)

// ToolResult represents the result of a tool execution
type ToolResult struct {
	OK      bool                   `json:"ok"`
	Tool    string                 `json:"tool,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Results interface{}            `json:"results,omitempty"`
	Content string                 `json:"content,omitempty"`
	Count   int                    `json:"count,omitempty"`
	Path    string                 `json:"path,omitempty"`
	Extra   map[string]interface{} `json:",inline"`
}

// SessionData stores conversation state
type SessionData struct {
	ConversationID string `json:"conversation_id"`
}

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, `Usage: execute-task "<task-id>" "<task-description>" "<plan-file-path>"`)
		os.Exit(2)
	}

	taskID := os.Args[1]
	taskDesc := os.Args[2]
	planFile := os.Args[3]

	// Validate task ID
	if !safeSessionRE.MatchString(taskID) {
		fmt.Fprintln(os.Stderr, "Invalid task ID: use A-Za-z0-9._- only, max 64 chars, must start with alphanumeric")
		os.Exit(2)
	}

	// Environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "OPENAI_API_KEY is required")
		os.Exit(2)
	}

	model := getEnv("OPENAI_MODEL", "gpt-5.2-codex")
	reasoningEffort := getEnv("REASONING_EFFORT", "medium")
	maxIters := getEnvInt("MAX_ITERS", defaultMaxIters)

	// Detect repo root
	repoRoot, err := detectRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to detect repo root: %v\n", err)
		os.Exit(2)
	}

	// Load plan content
	planContent, err := os.ReadFile(planFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read plan file: %v\n", err)
		os.Exit(2)
	}

	// Load CLAUDE.md if exists
	claudeMD := ""
	claudePath := filepath.Join(repoRoot, "CLAUDE.md")
	if data, err := os.ReadFile(claudePath); err == nil {
		claudeMD = string(data)
	}

	// Session management
	sessionsDir := getEnv("STATE_DIR", filepath.Join(repoRoot, ".codex-sessions", "tasks"))
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create sessions dir: %v\n", err)
		os.Exit(2)
	}

	sessionFile := filepath.Join(sessionsDir, taskID+".json")

	// Load or create conversation
	conversationID, err := loadSession(sessionFile)
	if err != nil || conversationID == "" {
		systemPrompt := buildSystemPrompt(repoRoot, taskID, taskDesc, string(planContent), claudeMD)
		conversationID, err = createConversation(apiKey, systemPrompt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[BLOCKED] Failed to create conversation: %v\n", err)
			os.Exit(3)
		}
		if err := saveSession(sessionFile, conversationID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save session: %v\n", err)
		}
	}

	// Execute task with tool loop
	if err := executeTask(apiKey, model, reasoningEffort, conversationID, taskID, taskDesc, repoRoot, maxIters); err != nil {
		fmt.Fprintf(os.Stderr, "[BLOCKED] %v\n", err)
		os.Exit(3)
	}
}

// Helper functions
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func detectRepoRoot() (string, error) {
	if root := os.Getenv("REPO_ROOT"); root != "" {
		return filepath.Abs(root)
	}

	// Walk up to find .git directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// No git found, use cwd
	return cwd, nil
}


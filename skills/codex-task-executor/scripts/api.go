package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const apiBase = "https://api.openai.com/v1"

// createConversation creates a new OpenAI conversation
func createConversation(apiKey, systemPrompt string) (string, error) {
	payload := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"role":    "developer",
				"content": systemPrompt,
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiBase+"/conversations", bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	conversationID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected response: %s", string(body))
	}

	return conversationID, nil
}

// buildSystemPrompt loads system-prompt.md and substitutes variables
func buildSystemPrompt(repoRoot, taskID, taskDesc, planContent, projectMemory string) string {
	scriptDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	promptPath := filepath.Join(scriptDir, "system-prompt.md")

	template, err := os.ReadFile(promptPath)
	if err != nil {
		// Fallback inline prompt
		return fmt.Sprintf(`You are a coding contractor executing Task #%s.

# Repository Root
%s

# Task Description
%s

# Plan Context
%s

# Project Guidelines
%s

# Your Role
Implement the task using available tools. Report progress with [PROGRESS] markers.

Available Tools: Glob, Grep, Read, Write, Edit
`, taskID, repoRoot, taskDesc, planContent, projectMemory)
	}

	prompt := string(template)
	prompt = strings.ReplaceAll(prompt, "{repo_root}", repoRoot)
	prompt = strings.ReplaceAll(prompt, "{task_id}", taskID)
	prompt = strings.ReplaceAll(prompt, "{task_description}", taskDesc)
	prompt = strings.ReplaceAll(prompt, "{plan_content}", planContent)
	prompt = strings.ReplaceAll(prompt, "{project_memory}", projectMemory)

	return prompt
}

// getToolsSchema returns OpenAI function tool definitions
func getToolsSchema() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"type": "function",
			"name": "Glob",
			"description": "Find repository files matching a glob pattern relative to repo root.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Glob like src/**/*.ts (relative to repo root)",
					},
					"max_results": map[string]interface{}{
						"type":        "integer",
						"description": "Max results (<=200). Default 200.",
					},
				},
				"required": []string{"pattern"},
			},
		},
		{
			"type": "function",
			"name": "Grep",
			"description": "Search for text in repository files; optionally restrict to a glob.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query (text).",
					},
					"glob": map[string]interface{}{
						"type":        "string",
						"description": "Optional file glob scope like src/**/*.ts",
					},
					"max_results": map[string]interface{}{
						"type":        "integer",
						"description": "Max matches (<=200). Default 200.",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"type": "function",
			"name": "Read",
			"description": "Read a file snippet by line range (relative path).",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Relative file path from repo root.",
					},
					"start_line": map[string]interface{}{
						"type":        "integer",
						"description": "1-based start line. Default 1.",
					},
					"end_line": map[string]interface{}{
						"type":        "integer",
						"description": "1-based end line (inclusive).",
					},
					"max_lines": map[string]interface{}{
						"type":        "integer",
						"description": "Max lines to return (<=400). Default 400.",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			"type": "function",
			"name": "Write",
			"description": "Create or overwrite a file with content. Creates parent directories if needed.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Relative file path from repo root.",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Full file content to write.",
					},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			"type": "function",
			"name": "Edit",
			"description": "Edit a file by replacing exact string match. Old string must appear exactly once.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Relative file path from repo root.",
					},
					"old_string": map[string]interface{}{
						"type":        "string",
						"description": "Exact string to replace (must be unique in file).",
					},
					"new_string": map[string]interface{}{
						"type":        "string",
						"description": "New string to replace with.",
					},
				},
				"required": []string{"path", "old_string", "new_string"},
			},
		},
	}
}

// executeTask runs the tool execution loop with Responses API
func executeTask(apiKey, model, reasoningEffort, conversationID, taskID, taskDesc, repoRoot string, maxIters int) error {
	ctx := context.Background()
	tools := getToolsSchema()

	// Initial input
	inputItems := []map[string]interface{}{
		{
			"role":    "user",
			"content": fmt.Sprintf("Execute Task #%s: %s", taskID, taskDesc),
		},
	}

	for iteration := 0; iteration < maxIters; iteration++ {
		// Build payload
		payload := map[string]interface{}{
			"model":                model,
			"conversation":         conversationID,
			"tools":                tools,
			"tool_choice":          "auto",
			"parallel_tool_calls":  false,
			"input":                inputItems,
		}

		if reasoningEffort != "" {
			payload["reasoning"] = map[string]interface{}{
				"effort": reasoningEffort,
			}
		}

		// Call Responses API
		respData, err := callResponsesAPI(ctx, apiKey, payload)
		if err != nil {
			return fmt.Errorf("API error: %w", err)
		}

		// Extract tool calls and text
		toolCalls, outputText := extractCallsAndText(respData)

		// Print output text (includes markers)
		if outputText != "" {
			fmt.Print(outputText)
		}

		if len(toolCalls) == 0 {
			// No tool calls => task complete
			fmt.Printf("\n[CODEX_COMPLETE] Task completed in %d iterations\n", iteration+1)
			return nil
		}

		// Execute tool calls
		outputs := []map[string]interface{}{}
		for _, call := range toolCalls {
			// Safe type assertions
			callID, ok := call["call_id"].(string)
			if !ok {
				continue
			}
			name, ok := call["name"].(string)
			if !ok {
				continue
			}
			argsStr, ok := call["arguments"].(string)
			if !ok {
				argsStr = "{}" // Default to empty args
			}

			// Parse arguments
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
				outputs = append(outputs, map[string]interface{}{
					"type":    "function_call_output",
					"call_id": callID,
					"output":  fmt.Sprintf(`{"ok": false, "error": "Invalid arguments: %v"}`, err),
				})
				continue
			}

			// Execute tool
			result := executeTool(repoRoot, name, args)
			resultJSON, _ := json.Marshal(result)

			fmt.Fprintf(os.Stderr, "[TOOL_CALL] %s(%s...)\n", name, argsStr[:min(100, len(argsStr))])

			outputs = append(outputs, map[string]interface{}{
				"type":    "function_call_output",
				"call_id": callID,
				"output":  string(resultJSON),
			})
		}

		inputItems = outputs
	}

	return fmt.Errorf("reached MAX_ITERS=%d without completion", maxIters)
}

// callResponsesAPI makes HTTP request to Responses API
func callResponsesAPI(ctx context.Context, apiKey string, payload map[string]interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiBase+"/responses", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body[:min(2000, len(body))]))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// extractCallsAndText parses response output
func extractCallsAndText(resp map[string]interface{}) ([]map[string]interface{}, string) {
	calls := []map[string]interface{}{}
	texts := []string{}

	output, ok := resp["output"].([]interface{})
	if !ok {
		return calls, ""
	}

	for _, item := range output {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		itemType, _ := itemMap["type"].(string)

		if itemType == "function_call" {
			calls = append(calls, map[string]interface{}{
				"name":      itemMap["name"],
				"call_id":   itemMap["call_id"],
				"arguments": itemMap["arguments"],
			})
		} else if itemType == "message" {
			content, ok := itemMap["content"].([]interface{})
			if !ok {
				continue
			}
			for _, c := range content {
				cMap, ok := c.(map[string]interface{})
				if !ok {
					continue
				}
				if cMap["type"] == "output_text" {
					if text, ok := cMap["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
		}
	}

	return calls, strings.Join(texts, "")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

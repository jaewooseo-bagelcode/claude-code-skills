package main

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// isDeniedPath checks if a path matches security denylist patterns
func isDeniedPath(relPath string) bool {
	relPath = filepath.ToSlash(relPath)
	base := filepath.Base(relPath)

	if denyBasenamesRE.MatchString(base) {
		return true
	}
	if denyExtRE.MatchString(relPath) {
		return true
	}
	if denyPathRE.MatchString(relPath) {
		return true
	}
	return false
}

// requireSafePath validates that a path is safe (no traversal, no absolute)
func requireSafePath(path string) error {
	if path == "" || strings.ContainsAny(path, "\n\r") {
		return fmt.Errorf("invalid path")
	}
	// Block Windows volume-prefixed paths (C:, D:, UNC)
	if vol := filepath.VolumeName(path); vol != "" {
		return fmt.Errorf("volume paths not allowed")
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed")
	}
	if strings.HasPrefix(path, "~") {
		return fmt.Errorf("home paths not allowed")
	}
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if part == ".." {
			return fmt.Errorf("parent traversal not allowed")
		}
	}
	return nil
}

// confineToRepo ensures path is within repo root and returns absolute path
func confineToRepo(repoRoot, relPath string) (string, error) {
	if err := requireSafePath(relPath); err != nil {
		return "", err
	}

	absRepo, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return "", fmt.Errorf("repo root resolution failed: %w", err)
	}

	targetPath := filepath.Join(repoRoot, relPath)
	absTarget, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		// File might not exist yet - that's OK, just check the parent
		parent := filepath.Dir(targetPath)
		absParent, err2 := filepath.EvalSymlinks(parent)
		if err2 != nil {
			return "", fmt.Errorf("path resolution failed: %w", err)
		}
		// Check parent is in repo
		if !strings.HasPrefix(absParent, absRepo+string(filepath.Separator)) && absParent != absRepo {
			return "", fmt.Errorf("path escapes repo root")
		}
		return targetPath, nil
	}

	// Check resolved path is within repo
	if !strings.HasPrefix(absTarget, absRepo+string(filepath.Separator)) && absTarget != absRepo {
		return "", fmt.Errorf("path escapes repo root")
	}

	return absTarget, nil
}

// isSymlink checks if path is a symlink (using lstat, not stat)
func isSymlink(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.Mode()&fs.ModeSymlink != 0, nil
}

// toolGlob finds files matching a pattern
func toolGlob(repoRoot, pattern string, maxResults int) ToolResult {
	if err := requireSafePath(pattern); err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Glob: %v", err)}
	}

	if maxResults <= 0 || maxResults > defaultMaxResults {
		maxResults = defaultMaxResults
	}

	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)

	if err := os.Chdir(repoRoot); err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Glob: %v", err)}
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Glob: %v", err)}
	}

	results := []string{}
	for _, match := range matches {
		if len(results) >= maxResults {
			break
		}

		// Check if file
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
			continue
		}

		relPath := filepath.ToSlash(match)
		if isDeniedPath(relPath) {
			continue
		}

		// Verify confinement
		if _, err := confineToRepo(repoRoot, relPath); err != nil {
			continue
		}

		results = append(results, relPath)
	}

	return ToolResult{
		OK:      true,
		Tool:    "Glob",
		Results: results,
		Count:   len(results),
		Extra:   map[string]interface{}{"repo_root": repoRoot, "pattern": pattern},
	}
}

// toolRead reads a file with line range
func toolRead(repoRoot, path string, startLine, endLine, maxLines int) ToolResult {
	if err := requireSafePath(path); err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Read: %v", err)}
	}

	if isDeniedPath(path) {
		return ToolResult{OK: false, Error: "Read: access denied"}
	}

	// SECURITY: Use openat-based secure open (perfect on Unix, strict validation on Windows)
	file, err := openSecure(repoRoot, path, os.O_RDONLY, 0)
	if err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Read: %v", err)}
	}
	defer file.Close()

	// Verify it's a regular file
	info, err := file.Stat()
	if err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Read: %v", err)}
	}
	if !info.Mode().IsRegular() {
		return ToolResult{OK: false, Error: "Read: not a regular file"}
	}

	// Read lines
	if maxLines <= 0 || maxLines > defaultMaxReadLines {
		maxLines = defaultMaxReadLines
	}
	if startLine < 1 {
		startLine = 1
	}
	if endLine <= 0 {
		endLine = startLine + maxLines - 1
	}
	if endLine < startLine {
		endLine = startLine
	}
	if endLine-startLine+1 > maxLines {
		endLine = startLine + maxLines - 1
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB line limit
	lines := []string{}
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if lineNum < startLine {
			continue
		}
		if lineNum > endLine {
			break
		}
		lines = append(lines, fmt.Sprintf("%06d\t%s", lineNum, scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Read: %v", err)}
	}

	return ToolResult{
		OK:      true,
		Tool:    "Read",
		Path:    path,
		Content: strings.Join(lines, "\n"),
		Extra: map[string]interface{}{
			"start":     startLine,
			"end":       endLine,
			"repo_root": repoRoot,
		},
	}
}

// toolWrite creates or overwrites a file
func toolWrite(repoRoot, path, content string) ToolResult {
	if err := requireSafePath(path); err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Write: %v", err)}
	}

	if isDeniedPath(path) {
		return ToolResult{OK: false, Error: "Write: access denied"}
	}

	// SECURITY: Create parent directories with symlink protection
	if err := createParentDirs(repoRoot, path); err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Write: mkdir failed: %v", err)}
	}

	// SECURITY: Open with complete protection (openat on Unix, strict validation on Windows)
	file, err := openSecure(repoRoot, path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Write: %v", err)}
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Write: %v", err)}
	}

	return ToolResult{
		OK:   true,
		Tool: "Write",
		Path: path,
		Extra: map[string]interface{}{
			"bytes": len(content),
		},
	}
}

// toolEdit performs precise string replacement
func toolEdit(repoRoot, path, oldString, newString string) ToolResult {
	if err := requireSafePath(path); err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Edit: %v", err)}
	}

	if isDeniedPath(path) {
		return ToolResult{OK: false, Error: "Edit: access denied"}
	}

	if oldString == "" {
		return ToolResult{OK: false, Error: "Edit: old_string must be non-empty"}
	}

	// SECURITY: Read with complete protection
	file, err := openSecure(repoRoot, path, os.O_RDONLY, 0)
	if err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Edit: %v", err)}
	}

	// Verify regular file
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return ToolResult{OK: false, Error: fmt.Sprintf("Edit: %v", err)}
	}
	if !info.Mode().IsRegular() {
		file.Close()
		return ToolResult{OK: false, Error: "Edit: not a regular file"}
	}

	// Read from already-opened FD (no path reopen)
	content, err := io.ReadAll(file)
	file.Close()
	if err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Edit: read failed: %v", err)}
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, oldString) {
		return ToolResult{OK: false, Error: "Edit: old_string not found in file"}
	}

	count := strings.Count(contentStr, oldString)
	if count > 1 {
		return ToolResult{OK: false, Error: fmt.Sprintf("Edit: old_string appears %d times (must be unique)", count)}
	}

	newContent := strings.Replace(contentStr, oldString, newString, 1)

	// SECURITY: Write with complete protection
	file, err = openSecure(repoRoot, path, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Edit: open for write failed: %v", err)}
	}
	defer file.Close()

	if _, err := file.WriteString(newContent); err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Edit: write failed: %v", err)}
	}

	return ToolResult{
		OK:   true,
		Tool: "Edit",
		Path: path,
		Extra: map[string]interface{}{
			"replaced": len(oldString),
			"with":     len(newString),
		},
	}
}


// toolGrep searches for text in files (simplified version)
func toolGrep(repoRoot, query, globFilter string, maxResults int) ToolResult {
	if query == "" {
		return ToolResult{OK: false, Error: "Grep: query required"}
	}

	if maxResults <= 0 || maxResults > defaultMaxResults {
		maxResults = defaultMaxResults
	}

	if globFilter != "" {
		if err := requireSafePath(globFilter); err != nil {
			return ToolResult{OK: false, Error: fmt.Sprintf("Grep: invalid glob: %v", err)}
		}
	}

	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)

	if err := os.Chdir(repoRoot); err != nil {
		return ToolResult{OK: false, Error: fmt.Sprintf("Grep: %v", err)}
	}

	// Walk files
	matches := []string{}
	walkFn := func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			// Skip .git
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		relPath := filepath.ToSlash(path)
		relPath = strings.TrimPrefix(relPath, "./")
		if isDeniedPath(relPath) {
			return nil
		}

		// Apply glob filter
		if globFilter != "" {
			matched, _ := filepath.Match(globFilter, relPath)
			if !matched {
				return nil
			}
		}

		// Skip large files
		if info.Size() > maxGrepFileSize {
			return nil
		}

		// Verify confinement
		if _, err := confineToRepo(repoRoot, relPath); err != nil {
			return nil
		}

		// Skip symlinks
		if info.Mode()&fs.ModeSymlink != 0 {
			return nil
		}

		// Search file
		if len(matches) >= maxResults {
			return fs.SkipAll
		}

		// SECURITY: Open with protection
		file, err := openSecure(repoRoot, relPath, os.O_RDONLY, 0)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB line limit
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			if len(matches) >= maxResults {
				break
			}
			line := scanner.Text()
			if strings.Contains(line, query) {
				matches = append(matches, fmt.Sprintf("%s:%d:%s", relPath, lineNum, line))
			}
		}

		// Ignore scanner errors (file read errors shouldn't stop entire grep)
		_ = scanner.Err()

		return nil
	}

	filepath.Walk(".", walkFn)

	return ToolResult{
		OK:      true,
		Tool:    "Grep",
		Results: matches,
		Count:   len(matches),
		Extra: map[string]interface{}{
			"repo_root": repoRoot,
			"query":     query,
			"glob":      globFilter,
		},
	}
}

// executeTool dispatches tool execution
func executeTool(repoRoot, toolName string, args map[string]interface{}) ToolResult {
	switch toolName {
	case "Glob":
		pattern, _ := args["pattern"].(string)
		maxResults, _ := args["max_results"].(float64)
		return toolGlob(repoRoot, pattern, int(maxResults))

	case "Read":
		path, _ := args["path"].(string)
		startLine, _ := args["start_line"].(float64)
		endLine, _ := args["end_line"].(float64)
		maxLines, _ := args["max_lines"].(float64)
		return toolRead(repoRoot, path, int(startLine), int(endLine), int(maxLines))

	case "Write":
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)
		return toolWrite(repoRoot, path, content)

	case "Edit":
		path, _ := args["path"].(string)
		oldString, _ := args["old_string"].(string)
		newString, _ := args["new_string"].(string)
		return toolEdit(repoRoot, path, oldString, newString)

	default:
		return ToolResult{OK: false, Error: fmt.Sprintf("Unknown tool: %s", toolName)}
	}
}

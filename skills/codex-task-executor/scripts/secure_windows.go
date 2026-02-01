// +build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// openSecure provides strict validation on Windows (no symlink support in stdlib)
func openSecure(repoRoot, relPath string, flags int, perm os.FileMode) (*os.File, error) {
	if err := requireSafePath(relPath); err != nil {
		return nil, err
	}

	// Build path and validate each component
	cleanPath := filepath.Clean(relPath)
	if strings.HasPrefix(cleanPath, "..") {
		return nil, errors.New("path escapes repository")
	}

	fullPath := filepath.Join(repoRoot, cleanPath)

	// Validate each component is not a symlink (best effort on Windows)
	currentPath := repoRoot
	parts := strings.Split(cleanPath, string(filepath.Separator))

	for i, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return nil, errors.New("parent traversal not allowed")
		}

		currentPath = filepath.Join(currentPath, part)

		// Check if this component exists and is not a symlink
		info, err := os.Lstat(currentPath)
		if err != nil {
			// Component doesn't exist - OK if not the last component (will be created)
			if i == len(parts)-1 {
				// Last component can be created by Write
				break
			}
			// Intermediate component doesn't exist - will be created by mkdir
			continue
		}

		// Windows: Check for reparse points (includes symlinks and junctions)
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("symlink not allowed")
		}
	}

	// Open file with validation
	// Note: Windows doesn't have O_NOFOLLOW equivalent, so this is best effort
	file, err := os.OpenFile(fullPath, flags, perm)
	if err != nil {
		return nil, err
	}

	// Verify it's a regular file (for Read/Edit)
	// O_RDONLY is 0, so check for absence of write flags
	if flags&(os.O_WRONLY|os.O_RDWR) == 0 {
		info, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, err
		}
		if !info.Mode().IsRegular() {
			file.Close()
			return nil, errors.New("not a regular file")
		}
	}

	return file, nil
}

// createParentDirs creates parent directories on Windows
func createParentDirs(repoRoot, relPath string) error {
	parent := filepath.Dir(relPath)
	if parent == "." || parent == "" {
		return nil
	}

	parentPath := filepath.Join(repoRoot, parent)

	// Validate no symlinks in path
	currentPath := repoRoot
	parts := strings.Split(parent, string(filepath.Separator))

	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return errors.New("parent traversal in mkdir")
		}

		currentPath = filepath.Join(currentPath, part)
		info, err := os.Lstat(currentPath)
		if err != nil {
			// Doesn't exist, will be created
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("symlink in parent path")
		}
	}

	return os.MkdirAll(parentPath, 0755)
}

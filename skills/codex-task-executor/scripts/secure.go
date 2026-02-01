// +build !windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// openSecure opens a file with complete TOCTOU and symlink protection using openat
func openSecure(repoRoot, relPath string, flags int, perm os.FileMode) (*os.File, error) {
	if err := requireSafePath(relPath); err != nil {
		return nil, err
	}

	// Open repo root directory
	rootFD, err := unix.Open(repoRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open repo root: %w", err)
	}
	defer unix.Close(rootFD)

	// Clean and split path
	cleanPath := filepath.Clean(relPath)
	if strings.HasPrefix(cleanPath, "..") {
		return nil, errors.New("path escapes repository")
	}

	parts := strings.Split(filepath.ToSlash(cleanPath), "/")
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == ".") {
		return nil, errors.New("invalid path")
	}

	// Walk path components with O_NOFOLLOW|O_DIRECTORY
	currentFD := rootFD
	needClose := false

	for i, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if needClose {
				unix.Close(currentFD)
			}
			return nil, errors.New("parent traversal not allowed")
		}

		isLast := i == len(parts)-1

		if isLast {
			// Final component: use requested flags
			finalFlags := flags | unix.O_NOFOLLOW | unix.O_CLOEXEC
			fd, err := unix.Openat(currentFD, part, finalFlags, uint32(perm))
			if needClose {
				unix.Close(currentFD)
			}
			if err != nil {
				return nil, err
			}
			return os.NewFile(uintptr(fd), filepath.Join(repoRoot, relPath)), nil
		} else {
			// Intermediate component: must be directory, no symlinks
			dirFlags := unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC
			fd, err := unix.Openat(currentFD, part, dirFlags, 0)
			if needClose {
				unix.Close(currentFD)
			}
			if err != nil {
				return nil, fmt.Errorf("cannot traverse %s: %w", part, err)
			}
			currentFD = fd
			needClose = true
		}
	}

	if needClose {
		unix.Close(currentFD)
	}
	return nil, errors.New("unexpected: loop completed without opening file")
}

// createParentDirs safely creates parent directories with symlink protection
func createParentDirs(repoRoot, relPath string) error {
	parent := filepath.Dir(relPath)
	if parent == "." || parent == "" {
		return nil // No parent to create
	}

	// Open repo root
	rootFD, err := unix.Open(repoRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(rootFD)

	parts := strings.Split(filepath.ToSlash(parent), "/")
	currentFD := rootFD
	needClose := false

	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if needClose {
				unix.Close(currentFD)
			}
			return errors.New("parent traversal in mkdir")
		}

		// Try to open existing directory
		fd, err := unix.Openat(currentFD, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err == nil {
			// Directory exists
			if needClose {
				unix.Close(currentFD)
			}
			currentFD = fd
			needClose = true
			continue
		}

		// Directory doesn't exist, create it
		if err := unix.Mkdirat(currentFD, part, 0755); err != nil {
			if needClose {
				unix.Close(currentFD)
			}
			return fmt.Errorf("mkdirat %s: %w", part, err)
		}

		// Now open the newly created directory
		fd, err = unix.Openat(currentFD, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if needClose {
			unix.Close(currentFD)
		}
		if err != nil {
			return fmt.Errorf("openat %s after mkdir: %w", part, err)
		}
		currentFD = fd
		needClose = true
	}

	if needClose {
		unix.Close(currentFD)
	}
	return nil
}

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func makeDir(dir string) error {
	_, err := os.Stat(dir)
	if !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(dir, 0755)
}

// isDirEmpty checks if a directory is empty
func isDirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // directory doesn't exist, treat as empty
		}
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil // directory is empty
	}
	return false, err // directory has contents or error occurred
}

// clearDir removes all contents of a directory
func clearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // directory doesn't exist, nothing to clear
		}
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		err = os.RemoveAll(path)
		if err != nil {
			return err
		}
	}
	return nil
}

// validateDestDir checks that the destination directory doesn't exist or is empty
func validateDestDir(dir string, force bool) error {
	empty, err := isDirEmpty(dir)
	if err != nil {
		return err
	}
	if !empty {
		if force {
			// clear the directory contents
			return clearDir(dir)
		}
		return fmt.Errorf("destination directory %q already exists and is not empty", dir)
	}
	return nil
}

func contentsToLines(contents []byte, size int) []string {
	var lines = []string{""}

	for i := 0; i < size; i++ {
		c := contents[i]
		if c != '\n' {
			lines[len(lines)-1] += string(c)
		} else if i+1 != size {
			lines = append(lines, "")
		}
	}

	return lines
}

// getRepoName extracts a clean repository name from the git repository
// It tries git remote URL first, then falls back to the directory name
func getRepoName(repoPath string) (string, error) {
	// First, resolve the actual path if it's relative (e.g., "." -> actual directory)
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return "", err
	}

	// Get the base name of the directory
	repoName := filepath.Base(absPath)

	// Remove .git suffix if present
	repoName = filepath.Clean(repoName)
	if filepath.Ext(repoName) == ".git" {
		repoName = repoName[:len(repoName)-4]
	}

	// Ensure we have a valid name
	if repoName == "" || repoName == "." || repoName == "/" {
		repoName = "repo"
	}

	return repoName, nil
}

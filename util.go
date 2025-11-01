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

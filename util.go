package main

import (
	"fmt"
	"io"
	"os"
)

func makedir(dir string) error {
	_, err := os.Stat(dir)
	if !os.IsNotExist(err) {
		return err
	}
	return os.Mkdir(dir, 0755)
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

// validateDestDir checks that the destination directory doesn't exist or is empty
func validateDestDir(dir string) error {
	empty, err := isDirEmpty(dir)
	if err != nil {
		return err
	}
	if !empty {
		return fmt.Errorf("destination directory %q already exists and is not empty", dir)
	}
	return nil
}

func contentstolines(contents []byte, size int) []string {
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

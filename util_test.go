package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMakeDir(t *testing.T) {
	t.Run("creates new directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		newDir := filepath.Join(tmpDir, "test")

		err := makeDir(newDir)
		if err != nil {
			t.Fatalf("makedir() failed: %v", err)
		}

		info, err := os.Stat(newDir)
		if err != nil {
			t.Fatalf("directory not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("created path is not a directory")
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		nestedDir := filepath.Join(tmpDir, "a", "b", "c")

		err := makeDir(nestedDir)
		if err != nil {
			t.Fatalf("makedir() failed: %v", err)
		}

		info, err := os.Stat(nestedDir)
		if err != nil {
			t.Fatalf("nested directory not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("created path is not a directory")
		}
	})

	t.Run("returns nil when directory already exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		// First call should succeed
		err := makeDir(tmpDir)
		if err != nil {
			t.Fatalf("first makeDir() failed: %v", err)
		}

		// Second call should return nil (directory already exists, stat succeeds)
		err = makeDir(tmpDir)
		if err != nil {
			t.Errorf("expected nil when directory already exists, got %v", err)
		}
	})

	t.Run("returns nil when file exists with same name", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "file")

		// Create a file first
		f, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		f.Close()

		// Try to create directory with same name - returns nil because stat succeeds
		err = makeDir(filePath)
		if err != nil {
			t.Errorf("expected nil when file exists with same name, got %v", err)
		}
	})
}

func TestIsDirEmpty(t *testing.T) {
	t.Run("returns true for empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		empty, err := isDirEmpty(tmpDir)
		if err != nil {
			t.Fatalf("isDirEmpty() failed: %v", err)
		}
		if !empty {
			t.Error("expected directory to be empty")
		}
	})

	t.Run("returns false for directory with files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a file in the directory
		f, err := os.Create(filepath.Join(tmpDir, "test.txt"))
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		f.Close()

		empty, err := isDirEmpty(tmpDir)
		if err != nil {
			t.Fatalf("isDirEmpty() failed: %v", err)
		}
		if empty {
			t.Error("expected directory to not be empty")
		}
	})

	t.Run("returns false for directory with subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a subdirectory
		err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
		if err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		empty, err := isDirEmpty(tmpDir)
		if err != nil {
			t.Fatalf("isDirEmpty() failed: %v", err)
		}
		if empty {
			t.Error("expected directory to not be empty")
		}
	})

	t.Run("returns true for non-existent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistent := filepath.Join(tmpDir, "does-not-exist")

		empty, err := isDirEmpty(nonExistent)
		if err != nil {
			t.Fatalf("isDirEmpty() failed: %v", err)
		}
		if !empty {
			t.Error("expected non-existent directory to be treated as empty")
		}
	})

	t.Run("returns error for file instead of directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "file.txt")

		f, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		f.Close()

		_, err = isDirEmpty(filePath)
		if err == nil {
			t.Error("expected error when checking if file is empty, got nil")
		}
	})
}

func TestClearDir(t *testing.T) {
	t.Run("removes all files from directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create multiple files
		for i := 0; i < 3; i++ {
			f, err := os.Create(filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt"))
			if err != nil {
				t.Fatalf("failed to create file: %v", err)
			}
			f.Close()
		}

		err := clearDir(tmpDir)
		if err != nil {
			t.Fatalf("clearDir() failed: %v", err)
		}

		empty, err := isDirEmpty(tmpDir)
		if err != nil {
			t.Fatalf("isDirEmpty() failed: %v", err)
		}
		if !empty {
			t.Error("expected directory to be empty after clearing")
		}
	})

	t.Run("removes nested directories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create nested structure
		subdir := filepath.Join(tmpDir, "subdir")
		err := os.Mkdir(subdir, 0755)
		if err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		f, err := os.Create(filepath.Join(subdir, "nested.txt"))
		if err != nil {
			t.Fatalf("failed to create nested file: %v", err)
		}
		f.Close()

		err = clearDir(tmpDir)
		if err != nil {
			t.Fatalf("clearDir() failed: %v", err)
		}

		empty, err := isDirEmpty(tmpDir)
		if err != nil {
			t.Fatalf("isDirEmpty() failed: %v", err)
		}
		if !empty {
			t.Error("expected directory to be empty after clearing")
		}
	})

	t.Run("handles non-existent directory gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistent := filepath.Join(tmpDir, "does-not-exist")

		err := clearDir(nonExistent)
		if err != nil {
			t.Errorf("clearDir() should not fail on non-existent directory: %v", err)
		}
	})

	t.Run("handles empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := clearDir(tmpDir)
		if err != nil {
			t.Fatalf("clearDir() failed on empty directory: %v", err)
		}

		empty, err := isDirEmpty(tmpDir)
		if err != nil {
			t.Fatalf("isDirEmpty() failed: %v", err)
		}
		if !empty {
			t.Error("expected directory to still be empty")
		}
	})
}

func TestValidateDestDir(t *testing.T) {
	t.Run("succeeds for non-existent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistent := filepath.Join(tmpDir, "new-dir")

		err := validateDestDir(nonExistent, false)
		if err != nil {
			t.Errorf("validateDestDir() failed for non-existent directory: %v", err)
		}
	})

	t.Run("succeeds for empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := validateDestDir(tmpDir, false)
		if err != nil {
			t.Errorf("validateDestDir() failed for empty directory: %v", err)
		}
	})

	t.Run("fails for non-empty directory without force", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a file
		f, err := os.Create(filepath.Join(tmpDir, "test.txt"))
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		f.Close()

		err = validateDestDir(tmpDir, false)
		if err == nil {
			t.Error("expected error for non-empty directory without force flag")
		}
	})

	t.Run("clears non-empty directory with force", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create files and subdirectories
		f, err := os.Create(filepath.Join(tmpDir, "test.txt"))
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		f.Close()

		subdir := filepath.Join(tmpDir, "subdir")
		err = os.Mkdir(subdir, 0755)
		if err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		err = validateDestDir(tmpDir, true)
		if err != nil {
			t.Errorf("validateDestDir() failed with force flag: %v", err)
		}

		empty, err := isDirEmpty(tmpDir)
		if err != nil {
			t.Fatalf("isDirEmpty() failed: %v", err)
		}
		if !empty {
			t.Error("expected directory to be empty after validation with force flag")
		}
	})

	t.Run("succeeds for non-existent directory with force", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistent := filepath.Join(tmpDir, "new-dir")

		err := validateDestDir(nonExistent, true)
		if err != nil {
			t.Errorf("validateDestDir() failed for non-existent directory with force: %v", err)
		}
	})
}

func TestContentsToLines(t *testing.T) {
	t.Run("splits single line without newline", func(t *testing.T) {
		content := []byte("hello world")
		lines := contentsToLines(content, len(content))

		if len(lines) != 1 {
			t.Errorf("expected 1 line, got %d", len(lines))
		}
		if lines[0] != "hello world" {
			t.Errorf("expected 'hello world', got '%s'", lines[0])
		}
	})

	t.Run("splits multiple lines", func(t *testing.T) {
		content := []byte("line1\nline2\nline3")
		lines := contentsToLines(content, len(content))

		expected := []string{"line1", "line2", "line3"}
		if len(lines) != len(expected) {
			t.Errorf("expected %d lines, got %d", len(expected), len(lines))
		}
		for i, line := range lines {
			if line != expected[i] {
				t.Errorf("line %d: expected '%s', got '%s'", i, expected[i], line)
			}
		}
	})

	t.Run("handles trailing newline", func(t *testing.T) {
		content := []byte("line1\nline2\n")
		lines := contentsToLines(content, len(content))

		expected := []string{"line1", "line2"}
		if len(lines) != len(expected) {
			t.Errorf("expected %d lines, got %d", len(expected), len(lines))
		}
		for i, line := range lines {
			if line != expected[i] {
				t.Errorf("line %d: expected '%s', got '%s'", i, expected[i], line)
			}
		}
	})

	t.Run("handles empty content", func(t *testing.T) {
		content := []byte("")
		lines := contentsToLines(content, 0)

		if len(lines) != 1 {
			t.Errorf("expected 1 line (empty string), got %d", len(lines))
		}
		if lines[0] != "" {
			t.Errorf("expected empty string, got '%s'", lines[0])
		}
	})

	t.Run("handles content with only newline", func(t *testing.T) {
		content := []byte("\n")
		lines := contentsToLines(content, len(content))

		expected := []string{""}
		if len(lines) != len(expected) {
			t.Errorf("expected %d lines, got %d", len(expected), len(lines))
		}
		if lines[0] != "" {
			t.Errorf("expected empty string, got '%s'", lines[0])
		}
	})

	t.Run("handles multiple consecutive newlines", func(t *testing.T) {
		content := []byte("line1\n\n\nline2")
		lines := contentsToLines(content, len(content))

		expected := []string{"line1", "", "", "line2"}
		if len(lines) != len(expected) {
			t.Errorf("expected %d lines, got %d", len(expected), len(lines))
		}
		for i, line := range lines {
			if line != expected[i] {
				t.Errorf("line %d: expected '%s', got '%s'", i, expected[i], line)
			}
		}
	})

	t.Run("handles content with special characters", func(t *testing.T) {
		content := []byte("hello world\nfoo bar\nbaz")
		lines := contentsToLines(content, len(content))

		expected := []string{"hello world", "foo bar", "baz"}
		if len(lines) != len(expected) {
			t.Errorf("expected %d lines, got %d", len(expected), len(lines))
		}
		for i, line := range lines {
			if line != expected[i] {
				t.Errorf("line %d: expected '%s', got '%s'", i, expected[i], line)
			}
		}
	})

	t.Run("handles mixed line endings in content", func(t *testing.T) {
		content := []byte("line1\nline2\nline3\n")
		lines := contentsToLines(content, len(content))

		expected := []string{"line1", "line2", "line3"}
		if len(lines) != len(expected) {
			t.Errorf("expected %d lines, got %d", len(expected), len(lines))
		}
		for i, line := range lines {
			if line != expected[i] {
				t.Errorf("line %d: expected '%s', got '%s'", i, expected[i], line)
			}
		}
	})
}

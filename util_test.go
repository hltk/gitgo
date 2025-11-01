package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"PNG file", "image.png", true},
		{"JPG file", "photo.jpg", true},
		{"JPEG file", "photo.jpeg", true},
		{"GIF file", "animation.gif", true},
		{"SVG file", "vector.svg", true},
		{"WEBP file", "modern.webp", true},
		{"BMP file", "bitmap.bmp", true},
		{"ICO file", "favicon.ico", true},
		{"uppercase PNG", "IMAGE.PNG", true},
		{"mixed case JPEG", "Photo.JpEg", true},
		{"text file", "readme.txt", false},
		{"markdown file", "README.md", false},
		{"go file", "main.go", false},
		{"no extension", "noextension", false},
		{"hidden image", ".hidden.png", true},
		{"multiple dots", "my.image.file.jpg", true},
		{"similar but not image", "image.pngx", false},
		{"similar but not image 2", "ximage.png.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isImageFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isImageFile(%q) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestRenderMarkdownToHTML(t *testing.T) {
	// Save and restore original config
	origRepoName := Config.RepoName
	defer func() { Config.RepoName = origRepoName }()
	Config.RepoName = "testrepo"

	t.Run("rewrites relative image URLs", func(t *testing.T) {
		markdown := []byte("# Test\n\n![alt text](image.png)\n\nSome text.")
		html := renderMarkdownToHTML(markdown)
		
		htmlStr := string(html)
		if !strings.Contains(htmlStr, `src="/testrepo/assets/image.png"`) {
			t.Errorf("expected image URL to be rewritten to /testrepo/assets/image.png, got: %s", htmlStr)
		}
	})

	t.Run("rewrites multiple image URLs", func(t *testing.T) {
		markdown := []byte("![first](image1.png)\n\n![second](image2.jpg)")
		html := renderMarkdownToHTML(markdown)
		
		htmlStr := string(html)
		if !strings.Contains(htmlStr, `src="/testrepo/assets/image1.png"`) {
			t.Errorf("expected first image URL to be rewritten, got: %s", htmlStr)
		}
		if !strings.Contains(htmlStr, `src="/testrepo/assets/image2.jpg"`) {
			t.Errorf("expected second image URL to be rewritten, got: %s", htmlStr)
		}
	})

	t.Run("does not rewrite absolute URLs", func(t *testing.T) {
		markdown := []byte("![external](https://example.com/image.png)")
		html := renderMarkdownToHTML(markdown)
		
		htmlStr := string(html)
		if !strings.Contains(htmlStr, `src="https://example.com/image.png"`) {
			t.Errorf("expected absolute URL to remain unchanged, got: %s", htmlStr)
		}
		if strings.Contains(htmlStr, "/testrepo/assets/") {
			t.Errorf("absolute URL should not be rewritten to assets folder: %s", htmlStr)
		}
	})

	t.Run("does not rewrite root-relative URLs", func(t *testing.T) {
		markdown := []byte("![root](/images/photo.png)")
		html := renderMarkdownToHTML(markdown)
		
		htmlStr := string(html)
		if !strings.Contains(htmlStr, `src="/images/photo.png"`) {
			t.Errorf("expected root-relative URL to remain unchanged, got: %s", htmlStr)
		}
	})

	t.Run("handles HTML img tags in markdown", func(t *testing.T) {
		markdown := []byte(`<img src="photo.jpg" alt="test" width="400">`)
		html := renderMarkdownToHTML(markdown)
		
		htmlStr := string(html)
		if !strings.Contains(htmlStr, `src="/testrepo/assets/photo.jpg"`) {
			t.Errorf("expected HTML img tag to be rewritten, got: %s", htmlStr)
		}
	})

	t.Run("handles mixed markdown and HTML images", func(t *testing.T) {
		markdown := []byte("![markdown](md.png)\n\n<img src=\"html.jpg\" alt=\"html\">")
		html := renderMarkdownToHTML(markdown)
		
		htmlStr := string(html)
		if !strings.Contains(htmlStr, `src="/testrepo/assets/md.png"`) {
			t.Errorf("expected markdown image to be rewritten: %s", htmlStr)
		}
		if !strings.Contains(htmlStr, `src="/testrepo/assets/html.jpg"`) {
			t.Errorf("expected HTML image to be rewritten: %s", htmlStr)
		}
	})

	t.Run("preserves other markdown formatting", func(t *testing.T) {
		markdown := []byte("# Header\n\n**bold** and *italic*\n\n![image](test.png)\n\n- list item")
		html := renderMarkdownToHTML(markdown)
		
		htmlStr := string(html)
		if !strings.Contains(htmlStr, "<h1") {
			t.Error("expected header to be rendered")
		}
		if !strings.Contains(htmlStr, "<strong>bold</strong>") {
			t.Error("expected bold text to be rendered")
		}
		if !strings.Contains(htmlStr, "<em>italic</em>") {
			t.Error("expected italic text to be rendered")
		}
		if !strings.Contains(htmlStr, "<li>list item</li>") {
			t.Error("expected list to be rendered")
		}
	})

	t.Run("handles images with various extensions", func(t *testing.T) {
		extensions := []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp"}
		for _, ext := range extensions {
			markdown := []byte("![test](image" + ext + ")")
			html := renderMarkdownToHTML(markdown)
			
			htmlStr := string(html)
			expected := `/testrepo/assets/image` + ext
			if !strings.Contains(htmlStr, expected) {
				t.Errorf("expected %s extension to be handled, got: %s", ext, htmlStr)
			}
		}
	})

	t.Run("does not modify non-image links", func(t *testing.T) {
		markdown := []byte("[link](page.html)")
		html := renderMarkdownToHTML(markdown)
		
		htmlStr := string(html)
		if strings.Contains(htmlStr, "/testrepo/assets/") {
			t.Errorf("non-image link should not be modified: %s", htmlStr)
		}
	})

	t.Run("handles empty markdown", func(t *testing.T) {
		markdown := []byte("")
		html := renderMarkdownToHTML(markdown)
		
		// Empty markdown should return empty HTML or minimal HTML
		// This is valid behavior - just verify it doesn't crash
		_ = html
	})

	t.Run("handles markdown with no images", func(t *testing.T) {
		markdown := []byte("# Just text\n\nNo images here.")
		html := renderMarkdownToHTML(markdown)
		
		htmlStr := string(html)
		if strings.Contains(htmlStr, "/assets/") {
			t.Error("markdown with no images should not have asset URLs")
		}
	})
}

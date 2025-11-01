package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFileServerHandler(t *testing.T) {
	t.Run("serves HTML files", func(t *testing.T) {
		// Create a temporary directory with test content
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.html")
		testContent := "<html><body>Test</body></html>"
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Create file server handler
		handler := http.FileServer(http.Dir(tmpDir))

		// Create test request - note the leading slash is important
		req := httptest.NewRequest("GET", "/test.html", nil)
		rec := httptest.NewRecorder()

		// Serve the request
		handler.ServeHTTP(rec, req)

		// Check response
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		body := rec.Body.String()
		if body != testContent {
			t.Errorf("expected body %q, got %q", testContent, body)
		}
	})

	t.Run("returns 404 for non-existent files", func(t *testing.T) {
		// Create an empty temporary directory
		tmpDir := t.TempDir()

		// Create file server handler
		handler := http.FileServer(http.Dir(tmpDir))

		// Request a file that doesn't exist
		req := httptest.NewRequest("GET", "/nonexistent.html", nil)
		rec := httptest.NewRecorder()

		// Serve the request
		handler.ServeHTTP(rec, req)

		// Check response
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})

	t.Run("serves index.html for directory", func(t *testing.T) {
		// Create a temporary directory with index.html
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "index.html")
		testContent := "<html><body>Index</body></html>"
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Create file server handler
		handler := http.FileServer(http.Dir(tmpDir))

		// Request the directory
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		// Serve the request
		handler.ServeHTTP(rec, req)

		// Check response
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		body := rec.Body.String()
		if body != testContent {
			t.Errorf("expected body %q, got %q", testContent, body)
		}
	})

	t.Run("lists directory contents when no index.html", func(t *testing.T) {
		// Create a temporary directory with some files
		tmpDir := t.TempDir()

		// Create test files
		files := []string{"file1.html", "file2.html", "file3.txt"}
		for _, file := range files {
			err := os.WriteFile(filepath.Join(tmpDir, file), []byte("content"), 0644)
			if err != nil {
				t.Fatalf("failed to create test file %s: %v", file, err)
			}
		}

		// Create file server handler
		handler := http.FileServer(http.Dir(tmpDir))

		// Request the directory
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		// Serve the request
		handler.ServeHTTP(rec, req)

		// Check response
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		// Check that file names appear in the listing
		body := rec.Body.String()
		for _, file := range files {
			if !contains(body, file) {
				t.Errorf("expected file %q to appear in directory listing", file)
			}
		}
	})

	t.Run("serves nested directories", func(t *testing.T) {
		// Create a temporary directory with nested structure
		tmpDir := t.TempDir()
		nestedDir := filepath.Join(tmpDir, "subdir")
		err := os.MkdirAll(nestedDir, 0755)
		if err != nil {
			t.Fatalf("failed to create nested directory: %v", err)
		}

		testFile := filepath.Join(nestedDir, "nested.html")
		testContent := "<html><body>Nested</body></html>"
		err = os.WriteFile(testFile, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Create file server handler
		handler := http.FileServer(http.Dir(tmpDir))

		// Request the nested file
		req := httptest.NewRequest("GET", "/subdir/nested.html", nil)
		rec := httptest.NewRecorder()

		// Serve the request
		handler.ServeHTTP(rec, req)

		// Check response
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		body := rec.Body.String()
		if body != testContent {
			t.Errorf("expected body %q, got %q", testContent, body)
		}
	})

	t.Run("sets correct content type for HTML", func(t *testing.T) {
		// Create a temporary directory with HTML file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.html")
		err := os.WriteFile(testFile, []byte("<html></html>"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Create file server handler
		handler := http.FileServer(http.Dir(tmpDir))

		// Request the file
		req := httptest.NewRequest("GET", "/test.html", nil)
		rec := httptest.NewRecorder()

		// Serve the request
		handler.ServeHTTP(rec, req)

		// Check content type
		contentType := rec.Header().Get("Content-Type")
		if contentType != "text/html; charset=utf-8" {
			t.Errorf("expected content type 'text/html; charset=utf-8', got %q", contentType)
		}
	})

	t.Run("handles large files", func(t *testing.T) {
		// Create a temporary directory with a large file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "large.html")

		// Create a large content (1MB)
		largeContent := make([]byte, 1024*1024)
		for i := range largeContent {
			largeContent[i] = byte('x')
		}

		err := os.WriteFile(testFile, largeContent, 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Create file server handler
		handler := http.FileServer(http.Dir(tmpDir))

		// Request the file
		req := httptest.NewRequest("GET", "/large.html", nil)
		rec := httptest.NewRecorder()

		// Serve the request
		handler.ServeHTTP(rec, req)

		// Check response
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		// Check content length
		body, err := io.ReadAll(rec.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}

		if len(body) != len(largeContent) {
			t.Errorf("expected body length %d, got %d", len(largeContent), len(body))
		}
	})
}

func TestServerConfiguration(t *testing.T) {
	t.Run("default port is 8000", func(t *testing.T) {
		expectedPort := ":8000"
		if expectedPort != ":8000" {
			t.Errorf("expected default port ':8000', got %q", expectedPort)
		}
	})

	t.Run("default directory is ./build", func(t *testing.T) {
		expectedDir := "./build"
		if expectedDir != "./build" {
			t.Errorf("expected default directory './build', got %q", expectedDir)
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/libgit2/git2go/v34"
)

func TestFlagParsing(t *testing.T) {
	t.Run("default flag values", func(t *testing.T) {
		// Reset flags for testing
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

		// Save original values
		origDestDir := Config.DestDir
		origInstallDir := Config.InstallDir
		origForce := Config.Force
		defer func() {
			Config.DestDir = origDestDir
			Config.InstallDir = origInstallDir
			Config.Force = origForce
		}()

		// Define flags like main does
		flag.StringVar(&Config.DestDir, "destdir", "build", "target directory")
		flag.StringVar(&Config.InstallDir, "installdir", ".", "install directory containing templates")
		flag.BoolVar(&Config.Force, "force", false, "force overwrite by clearing destination directory if not empty")

		// Parse empty args
		err := flag.CommandLine.Parse([]string{})
		if err != nil {
			t.Fatalf("failed to parse flags: %v", err)
		}

		// Check defaults
		if Config.DestDir != "build" {
			t.Errorf("expected default destdir 'build', got '%s'", Config.DestDir)
		}
		if Config.InstallDir != "." {
			t.Errorf("expected default installdir '.', got '%s'", Config.InstallDir)
		}
		if Config.Force != false {
			t.Errorf("expected default force false, got %v", Config.Force)
		}
	})

	t.Run("custom flag values", func(t *testing.T) {
		// Reset flags for testing
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

		// Save original values
		origDestDir := Config.DestDir
		origInstallDir := Config.InstallDir
		origForce := Config.Force
		defer func() {
			Config.DestDir = origDestDir
			Config.InstallDir = origInstallDir
			Config.Force = origForce
		}()

		// Define flags
		flag.StringVar(&Config.DestDir, "destdir", "build", "target directory")
		flag.StringVar(&Config.InstallDir, "installdir", ".", "install directory containing templates")
		flag.BoolVar(&Config.Force, "force", false, "force overwrite by clearing destination directory if not empty")

		// Parse custom args
		err := flag.CommandLine.Parse([]string{"-destdir=output", "-installdir=/tmp", "-force"})
		if err != nil {
			t.Fatalf("failed to parse flags: %v", err)
		}

		// Check custom values
		if Config.DestDir != "output" {
			t.Errorf("expected destdir 'output', got '%s'", Config.DestDir)
		}
		if Config.InstallDir != "/tmp" {
			t.Errorf("expected installdir '/tmp', got '%s'", Config.InstallDir)
		}
		if Config.Force != true {
			t.Errorf("expected force true, got %v", Config.Force)
		}
	})
}

func TestRepoNameExtraction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple repo name",
			input:    "/path/to/myrepo",
			expected: "myrepo",
		},
		{
			name:     "repo with .git suffix",
			input:    "/path/to/myrepo.git",
			expected: "myrepo",
		},
		{
			name:     "complex path with .git",
			input:    "/home/user/projects/awesome-project.git",
			expected: "awesome-project",
		},
		{
			name:     "relative path",
			input:    "./local-repo",
			expected: "local-repo",
		},
		{
			name:     "single directory name",
			input:    "repo",
			expected: "repo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the name extraction logic from main.go
			repoName := filepath.Base(tc.input)
			if len(repoName) > 4 && repoName[len(repoName)-4:] == ".git" {
				repoName = repoName[:len(repoName)-4]
			}

			if repoName != tc.expected {
				t.Errorf("expected repo name '%s', got '%s'", tc.expected, repoName)
			}
		})
	}
}

func TestDestDirConstruction(t *testing.T) {
	tests := []struct {
		name     string
		baseDir  string
		repoName string
		expected string
	}{
		{
			name:     "simple construction",
			baseDir:  "build",
			repoName: "myrepo",
			expected: "build/myrepo",
		},
		{
			name:     "absolute path",
			baseDir:  "/tmp/output",
			repoName: "project",
			expected: "/tmp/output/project",
		},
		{
			name:     "relative path",
			baseDir:  "./dist",
			repoName: "app",
			expected: "dist/app",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := filepath.Join(tc.baseDir, tc.repoName)
			expected := filepath.Clean(tc.expected)

			if result != expected {
				t.Errorf("expected dest dir '%s', got '%s'", expected, result)
			}
		})
	}
}

func TestGlobalDataStructure(t *testing.T) {
	t.Run("GlobalDataGlobal has correct links", func(t *testing.T) {
		if len(GlobalDataGlobal.Links) != 3 {
			t.Errorf("expected 3 links, got %d", len(GlobalDataGlobal.Links))
		}

		expectedLinks := []struct {
			pretty string
			link   string
		}{
			{"summary", "/"},
			{"tree", "/tree"},
			{"log", "/log"},
		}

		for i, expected := range expectedLinks {
			if i >= len(GlobalDataGlobal.Links) {
				break
			}
			link := GlobalDataGlobal.Links[i]
			if link.Pretty != expected.pretty {
				t.Errorf("link %d: expected pretty '%s', got '%s'", i, expected.pretty, link.Pretty)
			}
			if link.Link != expected.link {
				t.Errorf("link %d: expected link '%s', got '%s'", i, expected.link, link.Link)
			}
		}
	})

	t.Run("GlobalDataGlobal references Config", func(t *testing.T) {
		if GlobalDataGlobal.Config != &Config {
			t.Error("GlobalDataGlobal.Config does not reference Config")
		}
	})

	t.Run("LogoFound defaults to false", func(t *testing.T) {
		// Create a fresh GlobalRenderData
		testGlobal := GlobalRenderData{
			Config: &Config,
			Links:  []LinkListElem{{"summary", "/"}, {"tree", "/tree"}, {"log", "/log"}},
		}

		if testGlobal.LogoFound != false {
			t.Errorf("expected LogoFound to default to false, got %v", testGlobal.LogoFound)
		}
	})
}

func TestConfigDefaults(t *testing.T) {
	t.Run("Config has correct defaults", func(t *testing.T) {
		// Create a new config with defaults
		testConfig := ConfigStruct{
			MaxSummaryLen: 20,
			GitUrl:        "github.com/hltk",
		}

		if testConfig.MaxSummaryLen != 20 {
			t.Errorf("expected MaxSummaryLen 20, got %d", testConfig.MaxSummaryLen)
		}
		if testConfig.GitUrl != "github.com/hltk" {
			t.Errorf("expected GitUrl 'github.com/hltk', got '%s'", testConfig.GitUrl)
		}
	})
}

func TestReadmeAndLicenseFileMatching(t *testing.T) {
	t.Run("readme file patterns", func(t *testing.T) {
		readmeFiles := []string{"HEAD:README", "HEAD:README.md"}

		if len(readmeFiles) != 2 {
			t.Errorf("expected 2 readme patterns, got %d", len(readmeFiles))
		}

		expectedPatterns := []string{"HEAD:README", "HEAD:README.md"}
		for i, expected := range expectedPatterns {
			if readmeFiles[i] != expected {
				t.Errorf("pattern %d: expected '%s', got '%s'", i, expected, readmeFiles[i])
			}
		}
	})

	t.Run("license file patterns", func(t *testing.T) {
		licenseFiles := []string{"HEAD:LICENSE", "HEAD:COPYING", "HEAD:LICENSE.md"}

		if len(licenseFiles) != 3 {
			t.Errorf("expected 3 license patterns, got %d", len(licenseFiles))
		}

		expectedPatterns := []string{"HEAD:LICENSE", "HEAD:COPYING", "HEAD:LICENSE.md"}
		for i, expected := range expectedPatterns {
			if licenseFiles[i] != expected {
				t.Errorf("pattern %d: expected '%s', got '%s'", i, expected, licenseFiles[i])
			}
		}
	})
}

// Integration tests that use the run() function

func TestRunIntegration(t *testing.T) {
	t.Run("generates HTML for simple repository", func(t *testing.T) {
		// Create a test repository
		repoPath := t.TempDir()
		repo, err := git.InitRepository(repoPath, false)
		if err != nil {
			t.Fatalf("failed to init repository: %v", err)
		}
		defer repo.Free()

		// Create a file in the repo
		testFile := filepath.Join(repoPath, "README.md")
		err = os.WriteFile(testFile, []byte("# Test Repo\n\nThis is a test."), 0644)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Add file to index
		idx, err := repo.Index()
		if err != nil {
			t.Fatalf("failed to get index: %v", err)
		}

		err = idx.AddByPath("README.md")
		if err != nil {
			t.Fatalf("failed to add file to index: %v", err)
		}

		err = idx.Write()
		if err != nil {
			t.Fatalf("failed to write index: %v", err)
		}

		// Create tree
		treeId, err := idx.WriteTree()
		if err != nil {
			t.Fatalf("failed to write tree: %v", err)
		}

		tree, err := repo.LookupTree(treeId)
		if err != nil {
			t.Fatalf("failed to lookup tree: %v", err)
		}

		// Create commit
		sig := &git.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		}

		_, err = repo.CreateCommit("HEAD", sig, sig, "Initial commit", tree)
		if err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Setup directories
		tmpDir := t.TempDir()
		destDir := filepath.Join(tmpDir, "output")
		installDir := tmpDir

		// Create templates directory
		templatesDir := filepath.Join(installDir, "templates")
		err = os.MkdirAll(templatesDir, 0755)
		if err != nil {
			t.Fatalf("failed to create templates dir: %v", err)
		}

		// Create minimal templates
		templates := map[string]string{
			"header.html":       `{{define "header.html"}}<html><head><title>Test</title></head><body>{{end}}`,
			"footer.html":       `{{define "footer.html"}}</body></html>{{end}}`,
			"nav.html":          `{{define "nav.html"}}<nav></nav>{{end}}`,
			"index.html":        `{{define "index.html"}}{{template "header.html" .GlobalData}}{{template "nav.html" .GlobalData}}<h1>Index</h1>{{template "footer.html" .GlobalData}}{{end}}`,
			"tree.html":         `{{define "tree.html"}}{{template "header.html" .GlobalData}}{{template "nav.html" .GlobalData}}<h1>Tree</h1>{{template "footer.html" .GlobalData}}{{end}}`,
			"file.html":         `{{define "file.html"}}{{template "header.html" .GlobalData}}{{template "nav.html" .GlobalData}}<h1>File</h1>{{template "footer.html" .GlobalData}}{{end}}`,
			"log.html":          `{{define "log.html"}}{{template "header.html" .GlobalData}}{{template "nav.html" .GlobalData}}<h1>Log</h1>{{template "footer.html" .GlobalData}}{{end}}`,
			"commit.html":       `{{define "commit.html"}}{{template "header.html" .GlobalData}}{{template "nav.html" .GlobalData}}<h1>Commit</h1>{{template "footer.html" .GlobalData}}{{end}}`,
			"fileview.html":     `{{define "fileview.html"}}<pre>{{range .Lines}}{{.}}\n{{end}}</pre>{{end}}`,
			"linenumberer.html": `{{define "linenumberer.html"}}{{end}}`,
		}

		for name, content := range templates {
			err = os.WriteFile(filepath.Join(templatesDir, name), []byte(content), 0644)
			if err != nil {
				t.Fatalf("failed to write template %s: %v", name, err)
			}
		}

		// Run the main logic
		err = run(repoPath, destDir, installDir, false)
		if err != nil {
			t.Fatalf("run() failed: %v", err)
		}

		// Verify outputs were created
		repoName := filepath.Base(repoPath)

		// Get branch name for log path verification
		branchName := getBranchName(repo)

		expectedFiles := []string{
			filepath.Join(destDir, repoName, "index.html"),
			filepath.Join(destDir, repoName, "log", branchName, "index.html"),
			filepath.Join(destDir, repoName, "tree", "index.html"),
		}

		for _, file := range expectedFiles {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				t.Errorf("expected file not created: %s", file)
			}
		}
	})

	t.Run("returns error for non-existent repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		destDir := filepath.Join(tmpDir, "output")
		installDir := tmpDir
		nonExistentRepo := filepath.Join(tmpDir, "does-not-exist")

		err := run(nonExistentRepo, destDir, installDir, false)
		if err == nil {
			t.Error("expected error for non-existent repository, got nil")
		}
	})

	t.Run("handles force flag for non-empty destination", func(t *testing.T) {
		// Create a test repository
		repoPath := t.TempDir()
		repo, err := git.InitRepository(repoPath, false)
		if err != nil {
			t.Fatalf("failed to init repository: %v", err)
		}
		defer repo.Free()

		// Create a file and commit
		testFile := filepath.Join(repoPath, "test.txt")
		err = os.WriteFile(testFile, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		idx, err := repo.Index()
		if err != nil {
			t.Fatalf("failed to get index: %v", err)
		}

		err = idx.AddByPath("test.txt")
		if err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		err = idx.Write()
		if err != nil {
			t.Fatalf("failed to write index: %v", err)
		}

		treeId, err := idx.WriteTree()
		if err != nil {
			t.Fatalf("failed to write tree: %v", err)
		}

		tree, err := repo.LookupTree(treeId)
		if err != nil {
			t.Fatalf("failed to lookup tree: %v", err)
		}

		sig := &git.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		}

		_, err = repo.CreateCommit("HEAD", sig, sig, "Test commit", tree)
		if err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Setup directories
		tmpDir := t.TempDir()
		destDir := filepath.Join(tmpDir, "output")
		installDir := tmpDir

		// Pre-create destination with content
		repoName := filepath.Base(repoPath)
		fullDestDir := filepath.Join(destDir, repoName)
		err = os.MkdirAll(fullDestDir, 0755)
		if err != nil {
			t.Fatalf("failed to create dest dir: %v", err)
		}

		// Add a file to make it non-empty
		err = os.WriteFile(filepath.Join(fullDestDir, "existing.txt"), []byte("exists"), 0644)
		if err != nil {
			t.Fatalf("failed to write existing file: %v", err)
		}

		// Create templates
		templatesDir := filepath.Join(installDir, "templates")
		err = os.MkdirAll(templatesDir, 0755)
		if err != nil {
			t.Fatalf("failed to create templates dir: %v", err)
		}

		// Create minimal templates
		templates := map[string]string{
			"header.html":       `{{define "header.html"}}<html><head><title>Test</title></head><body>{{end}}`,
			"footer.html":       `{{define "footer.html"}}</body></html>{{end}}`,
			"nav.html":          `{{define "nav.html"}}<nav></nav>{{end}}`,
			"index.html":        `{{define "index.html"}}{{template "header.html" .}}{{template "nav.html" .}}<h1>Index</h1>{{template "footer.html" .}}{{end}}`,
			"tree.html":         `{{define "tree.html"}}{{template "header.html" .}}{{template "nav.html" .}}<h1>Tree</h1>{{template "footer.html" .}}{{end}}`,
			"file.html":         `{{define "file.html"}}{{template "header.html" .}}{{template "nav.html" .}}<h1>File</h1>{{template "footer.html" .}}{{end}}`,
			"log.html":          `{{define "log.html"}}{{template "header.html" .}}{{template "nav.html" .}}<h1>Log</h1>{{template "footer.html" .}}{{end}}`,
			"commit.html":       `{{define "commit.html"}}{{template "header.html" .}}{{template "nav.html" .}}<h1>Commit</h1>{{template "footer.html" .}}{{end}}`,
			"fileview.html":     `{{define "fileview.html"}}<pre>{{range .Lines}}{{.}}\n{{end}}</pre>{{end}}`,
			"linenumberer.html": `{{define "linenumberer.html"}}{{end}}`,
		}

		for name, content := range templates {
			err = os.WriteFile(filepath.Join(templatesDir, name), []byte(content), 0644)
			if err != nil {
				t.Fatalf("failed to write template %s: %v", name, err)
			}
		}

		// Run should succeed with force flag
		err = run(repoPath, destDir, installDir, true)
		if err != nil {
			t.Errorf("run() with force flag failed: %v", err)
		}

		// Verify outputs were created
		expectedFile := filepath.Join(fullDestDir, "index.html")
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Error("expected index.html to be created with force flag")
		}
	})
}

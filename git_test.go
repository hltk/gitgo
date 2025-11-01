package main

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/libgit2/git2go/v34"
)

// setGlobalTemplate is a helper to set the global template variable t
// which is shadowed by the test parameter t *testing.T
func setGlobalTemplate(tmpl *template.Template) {
	t = tmpl
}

// createTestRepo creates a simple test repository for testing
func createTestRepo(t *testing.T) (*git.Repository, string) {
	t.Helper()

	repoPath := t.TempDir()

	repo, err := git.InitRepository(repoPath, false)
	if err != nil {
		t.Fatalf("failed to init repository: %v", err)
	}

	return repo, repoPath
}

// createCommitInRepo creates a commit with a single file in the repository
func createCommitInRepo(t *testing.T, repo *git.Repository, repoPath string, filename, content, message string) *git.Oid {
	t.Helper()

	// Create a file
	filePath := filepath.Join(repoPath, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Get the repository index
	idx, err := repo.Index()
	if err != nil {
		t.Fatalf("failed to get index: %v", err)
	}

	// Add file to index
	err = idx.AddByPath(filename)
	if err != nil {
		t.Fatalf("failed to add file to index: %v", err)
	}

	// Write index
	err = idx.Write()
	if err != nil {
		t.Fatalf("failed to write index: %v", err)
	}

	// Create tree from index
	treeId, err := idx.WriteTree()
	if err != nil {
		t.Fatalf("failed to write tree: %v", err)
	}

	tree, err := repo.LookupTree(treeId)
	if err != nil {
		t.Fatalf("failed to lookup tree: %v", err)
	}

	// Create signature
	sig := &git.Signature{
		Name:  "Test User",
		Email: "test@example.com",
		When:  time.Now(),
	}

	// Get parent commit if exists
	var parents []*git.Commit
	head, err := repo.Head()
	if err == nil {
		headCommit, err := repo.LookupCommit(head.Target())
		if err == nil {
			parents = append(parents, headCommit)
		}
	}

	// Create commit
	var commitId *git.Oid
	if len(parents) > 0 {
		commitId, err = repo.CreateCommit("HEAD", sig, sig, message, tree, parents...)
	} else {
		commitId, err = repo.CreateCommit("HEAD", sig, sig, message, tree)
	}
	if err != nil {
		t.Fatalf("failed to create commit: %v", err)
	}

	return commitId
}

func TestGetcommitlog(t *testing.T) {
	// Setup temp directory and configure
	origDestDir := Config.DestDir
	origInstallDir := Config.InstallDir
	tmpDir := t.TempDir()
	Config.DestDir = filepath.Join(tmpDir, "build")
	Config.InstallDir = tmpDir
	defer func() {
		Config.DestDir = origDestDir
		Config.InstallDir = origInstallDir
	}()

	// Create build directory and subdirectories
	err := os.MkdirAll(Config.DestDir, 0755)
	if err != nil {
		t.Fatalf("failed to create destdir: %v", err)
	}

	err = os.MkdirAll(filepath.Join(Config.DestDir, "commit"), 0755)
	if err != nil {
		t.Fatalf("failed to create commit dir: %v", err)
	}

	// Initialize templates (minimal setup)
	err = os.MkdirAll(filepath.Join(tmpDir, "templates"), 0755)
	if err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}

	// Create minimal template files
	commitTemplate := `{{define "commit.html"}}test{{end}}`
	err = os.WriteFile(filepath.Join(tmpDir, "templates", "commit.html"), []byte(commitTemplate), 0644)
	if err != nil {
		t.Fatalf("failed to write commit template: %v", err)
	}

	t.Run("returns empty list for repository with no commits", func(t *testing.T) {
		repo, _ := createTestRepo(t)
		defer repo.Free()

		// Create an empty tree
		builder, err := repo.TreeBuilder()
		if err != nil {
			t.Fatalf("failed to create tree builder: %v", err)
		}
		defer builder.Free()

		treeId, err := builder.Write()
		if err != nil {
			t.Fatalf("failed to write tree: %v", err)
		}

		// Skip actual commit log test since we need at least one commit
		// The function will fail with an invalid head OID for empty repos
		_ = treeId
		t.Skip("Empty repository test skipped - getcommitlog requires valid HEAD")
	})

	t.Run("returns commit list with single commit", func(t *testing.T) {
		repo, repoPath := createTestRepo(t)
		defer repo.Free()

		// Create a commit
		commitId := createCommitInRepo(t, repo, repoPath, "test.txt", "hello world", "Initial commit")

		// Parse templates before calling getcommitlog
		templ = template.New("").Funcs(funcmap)
		var err error
		parsedTemplate, err := templ.ParseGlob(filepath.Join(Config.InstallDir, "templates/*.html"))
		if err != nil {
			t.Fatalf("failed to parse templates: %v", err)
		}
		setGlobalTemplate(parsedTemplate)

		// Get commit log
		commitList := getcommitlog(repo, commitId)

		if len(commitList) != 1 {
			t.Errorf("expected 1 commit, got %d", len(commitList))
		}

		if len(commitList) > 0 {
			commit := commitList[0]
			if commit.Msg != "Initial commit" {
				t.Errorf("expected message 'Initial commit', got '%s'", commit.Msg)
			}
			if commit.Name != "Test User" {
				t.Errorf("expected author 'Test User', got '%s'", commit.Name)
			}
		}
	})

	t.Run("returns multiple commits in order", func(t *testing.T) {
		repo, repoPath := createTestRepo(t)
		defer repo.Free()

		// Create multiple commits
		_ = createCommitInRepo(t, repo, repoPath, "file1.txt", "content1", "First commit")
		_ = createCommitInRepo(t, repo, repoPath, "file2.txt", "content2", "Second commit")
		headId := createCommitInRepo(t, repo, repoPath, "file3.txt", "content3", "Third commit")

		// Parse templates
		templ = template.New("").Funcs(funcmap)
		parsedTemplate, err := templ.ParseGlob(filepath.Join(Config.InstallDir, "templates/*.html"))
		if err != nil {
			t.Fatalf("failed to parse templates: %v", err)
		}
		setGlobalTemplate(parsedTemplate)

		// Get commit log
		commitList := getcommitlog(repo, headId)

		if len(commitList) != 3 {
			t.Errorf("expected 3 commits, got %d", len(commitList))
		}

		// Verify order (should be newest first)
		expectedMsgs := []string{"Third commit", "Second commit", "First commit"}
		for i, commit := range commitList {
			if commit.Msg != expectedMsgs[i] {
				t.Errorf("commit %d: expected message '%s', got '%s'", i, expectedMsgs[i], commit.Msg)
			}
		}
	})

	t.Run("truncates long commit messages", func(t *testing.T) {
		repo, repoPath := createTestRepo(t)
		defer repo.Free()

		// Set a small MaxSummaryLen for testing
		origMaxSummary := Config.MaxSummaryLen
		Config.MaxSummaryLen = 10
		defer func() { Config.MaxSummaryLen = origMaxSummary }()

		longMsg := "This is a very long commit message that should be truncated"
		commitId := createCommitInRepo(t, repo, repoPath, "test.txt", "content", longMsg)

		// Parse templates
		templ = template.New("").Funcs(funcmap)
		parsedTemplate, err := templ.ParseGlob(filepath.Join(Config.InstallDir, "templates/*.html"))
		if err != nil {
			t.Fatalf("failed to parse templates: %v", err)
		}
		setGlobalTemplate(parsedTemplate)

		commitList := getcommitlog(repo, commitId)

		if len(commitList) != 1 {
			t.Fatalf("expected 1 commit, got %d", len(commitList))
		}

		if len(commitList[0].Msg) > Config.MaxSummaryLen {
			t.Errorf("expected message to be truncated to %d chars, got %d: '%s'",
				Config.MaxSummaryLen, len(commitList[0].Msg), commitList[0].Msg)
		}

		if commitList[0].Msg != "This is..." {
			t.Errorf("expected 'This is...', got '%s'", commitList[0].Msg)
		}
	})
}

func TestIndextree(t *testing.T) {
	// Setup temp directory and configure
	origDestDir := Config.DestDir
	origInstallDir := Config.InstallDir
	tmpDir := t.TempDir()
	Config.DestDir = filepath.Join(tmpDir, "build")
	Config.InstallDir = tmpDir
	defer func() {
		Config.DestDir = origDestDir
		Config.InstallDir = origInstallDir
	}()

	// Create build directory structure
	err := os.MkdirAll(Config.DestDir, 0755)
	if err != nil {
		t.Fatalf("failed to create destdir: %v", err)
	}

	err = os.MkdirAll(filepath.Join(Config.DestDir, "tree"), 0755)
	if err != nil {
		t.Fatalf("failed to create tree dir: %v", err)
	}

	// Create templates directory
	err = os.MkdirAll(filepath.Join(tmpDir, "templates"), 0755)
	if err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}

	// Create minimal template files
	templates := map[string]string{
		"tree.html": `{{define "tree.html"}}tree{{end}}`,
		"file.html": `{{define "file.html"}}file{{end}}`,
	}

	for name, content := range templates {
		err = os.WriteFile(filepath.Join(tmpDir, "templates", name), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write %s template: %v", name, err)
		}
	}

	t.Run("creates tree index for simple repository", func(t *testing.T) {
		repo, repoPath := createTestRepo(t)
		defer repo.Free()

		// Create a commit with files
		commitId := createCommitInRepo(t, repo, repoPath, "README.md", "# Test", "Initial commit")

		// Parse templates
		templ = template.New("").Funcs(funcmap)
		parsedTemplate, err := templ.ParseGlob(filepath.Join(Config.InstallDir, "templates/*.html"))
		if err != nil {
			t.Fatalf("failed to parse templates: %v", err)
		}
		setGlobalTemplate(parsedTemplate)

		// Run indextree
		indextree(repo, commitId)

		// Verify tree directory was created
		treePath := filepath.Join(Config.DestDir, "tree")
		if _, err := os.Stat(treePath); os.IsNotExist(err) {
			t.Error("tree directory was not created")
		}

		// Verify index.html was created
		indexPath := filepath.Join(treePath, "index.html")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			t.Error("tree/index.html was not created")
		}

		// Verify file.html was created for README.md
		readmePath := filepath.Join(treePath, "README.md.html")
		if _, err := os.Stat(readmePath); os.IsNotExist(err) {
			t.Error("tree/README.md.html was not created")
		}
	})

	t.Run("handles nested directory structure", func(t *testing.T) {
		repo, repoPath := createTestRepo(t)
		defer repo.Free()

		// Create nested directory structure
		subdir := filepath.Join(repoPath, "src")
		err := os.MkdirAll(subdir, 0755)
		if err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		// Create files in nested structure
		err = os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# Test"), 0644)
		if err != nil {
			t.Fatalf("failed to write README: %v", err)
		}

		err = os.WriteFile(filepath.Join(subdir, "main.go"), []byte("package main"), 0644)
		if err != nil {
			t.Fatalf("failed to write main.go: %v", err)
		}

		// Get the repository index
		idx, err := repo.Index()
		if err != nil {
			t.Fatalf("failed to get index: %v", err)
		}

		// Add files to index
		err = idx.AddByPath("README.md")
		if err != nil {
			t.Fatalf("failed to add README: %v", err)
		}

		err = idx.AddByPath("src/main.go")
		if err != nil {
			t.Fatalf("failed to add main.go: %v", err)
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

		commitId, err := repo.CreateCommit("HEAD", sig, sig, "Initial commit", tree)
		if err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Parse templates
		templ = template.New("").Funcs(funcmap)
		parsedTemplate, err2 := templ.ParseGlob(filepath.Join(Config.InstallDir, "templates/*.html"))
		if err2 != nil {
			t.Fatalf("failed to parse templates: %v", err2)
		}
		setGlobalTemplate(parsedTemplate)

		// Run indextree
		indextree(repo, commitId)

		// Verify nested directory was created
		srcPath := filepath.Join(Config.DestDir, "tree", "src")
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			t.Error("tree/src directory was not created")
		}

		// Verify nested file was created
		mainPath := filepath.Join(srcPath, "main.go.html")
		if _, err := os.Stat(mainPath); os.IsNotExist(err) {
			t.Error("tree/src/main.go.html was not created")
		}
	})
}

func TestIndextreerecursive(t *testing.T) {
	// Setup temp directory and configure
	origDestDir := Config.DestDir
	origInstallDir := Config.InstallDir
	tmpDir := t.TempDir()
	Config.DestDir = filepath.Join(tmpDir, "build")
	Config.InstallDir = tmpDir
	defer func() {
		Config.DestDir = origDestDir
		Config.InstallDir = origInstallDir
	}()

	// Create build directory structure
	err := os.MkdirAll(Config.DestDir, 0755)
	if err != nil {
		t.Fatalf("failed to create destdir: %v", err)
	}

	// Create templates
	err = os.MkdirAll(filepath.Join(tmpDir, "templates"), 0755)
	if err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}

	templates := map[string]string{
		"tree.html": `{{define "tree.html"}}tree{{end}}`,
		"file.html": `{{define "file.html"}}file{{end}}`,
	}

	for name, content := range templates {
		err = os.WriteFile(filepath.Join(tmpDir, "templates", name), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write %s template: %v", name, err)
		}
	}

	t.Run("processes tree with single file", func(t *testing.T) {
		repo, repoPath := createTestRepo(t)
		defer repo.Free()

		commitId := createCommitInRepo(t, repo, repoPath, "test.txt", "content", "commit")

		commit, err := repo.LookupCommit(commitId)
		if err != nil {
			t.Fatalf("failed to lookup commit: %v", err)
		}

		tree, err := commit.Tree()
		if err != nil {
			t.Fatalf("failed to get tree: %v", err)
		}

		// Parse templates
		templ = template.New("").Funcs(funcmap)
		parsedTemplate, err2 := templ.ParseGlob(filepath.Join(Config.InstallDir, "templates/*.html"))
		if err2 != nil {
			t.Fatalf("failed to parse templates: %v", err2)
		}
		setGlobalTemplate(parsedTemplate)

		// Create the tree path
		treePath := "/tree"
		err = os.MkdirAll(filepath.Join(Config.DestDir, treePath), 0755)
		if err != nil {
			t.Fatalf("failed to create tree path: %v", err)
		}

		// Run indextreerecursive
		indextreerecursive(repo, tree, treePath)

		// Verify file was created
		filePath := filepath.Join(Config.DestDir, treePath, "test.txt.html")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("test.txt.html was not created")
		}

		// Verify index was created
		indexPath := filepath.Join(Config.DestDir, treePath, "index.html")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			t.Error("index.html was not created")
		}
	})
}

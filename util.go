package main

import (
	"bytes"
	"fmt"
	htmlpkg "html"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
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

// highlightFileContents applies syntax highlighting to file contents
// Returns an array of HTML strings, one per line
func highlightFileContents(filename string, contents []byte) []template.HTML {
	// Detect lexer from filename
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	// Use HTML formatter with classes (not inline styles)
	formatter := chromahtml.New(chromahtml.WithClasses(true), chromahtml.WithLineNumbers(false), chromahtml.PreventSurroundingPre(true))

	// Get the style (we'll use github style, CSS will be generated separately)
	style := styles.Get("github")
	if style == nil {
		style = styles.Fallback
	}

	// Tokenize the code
	iterator, err := lexer.Tokenise(nil, string(contents))
	if err != nil {
		// If tokenization fails, return plain text lines
		return contentsToLinesHTML(contents, len(contents))
	}

	// Format to HTML
	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		// If formatting fails, return plain text lines
		return contentsToLinesHTML(contents, len(contents))
	}

	// Split the HTML into lines
	htmlStr := buf.String()
	lines := strings.Split(strings.TrimRight(htmlStr, "\n"), "\n")

	// Convert to template.HTML to prevent escaping
	result := make([]template.HTML, len(lines))
	for i, line := range lines {
		result[i] = template.HTML(line)
	}

	return result
}

// highlightDiffLines applies basic coloring to diff lines
// Lines starting with + are wrapped in diff-add span, - in diff-del span
func highlightDiffLines(diffText string) []template.HTML {
	lines := strings.Split(strings.TrimRight(diffText, "\n"), "\n")
	result := make([]template.HTML, len(lines))

	for i, line := range lines {
		// Escape HTML in the line first
		escapedLine := htmlpkg.EscapeString(line)

		if len(line) > 0 {
			if line[0] == '+' {
				result[i] = template.HTML(`<span class="diff-add">` + escapedLine + `</span>`)
			} else if line[0] == '-' {
				result[i] = template.HTML(`<span class="diff-del">` + escapedLine + `</span>`)
			} else {
				result[i] = template.HTML(escapedLine)
			}
		} else {
			result[i] = template.HTML(escapedLine)
		}
	}

	return result
}

// contentsToLinesHTML is a fallback that converts plain text to HTML lines
// Used when syntax highlighting fails
func contentsToLinesHTML(contents []byte, size int) []template.HTML {
	lines := contentsToLines(contents, size)
	result := make([]template.HTML, len(lines))
	for i, line := range lines {
		result[i] = template.HTML(htmlpkg.EscapeString(line))
	}
	return result
}

// generateChromaCSS generates the CSS stylesheet for syntax highlighting
// Returns the CSS as a string
func generateChromaCSS() (string, error) {
	style := styles.Get("github")
	if style == nil {
		style = styles.Fallback
	}

	formatter := chromahtml.New(chromahtml.WithClasses(true))

	var buf bytes.Buffer
	err := formatter.WriteCSS(&buf, style)
	if err != nil {
		return "", err
	}

	// Add custom diff colors
	customCSS := `
/* Diff highlighting */
.diff-add {
	background-color: #e6ffed;
	color: #24292e;
}
.diff-del {
	background-color: #ffeef0;
	color: #24292e;
}
`

	return buf.String() + customCSS, nil
}

// isImageFile checks if a filename has an image extension
func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".bmp", ".ico":
		return true
	}
	return false
}

// renderMarkdownToHTML converts markdown bytes to HTML using goldmark
// Returns the HTML as template.HTML to prevent escaping
// Also rewrites relative image URLs to point to the assets directory
func renderMarkdownToHTML(contents []byte) template.HTML {
	// Configure goldmark to allow unsafe HTML (so img tags in markdown are preserved)
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // Allow raw HTML in markdown
		),
	)

	var buf bytes.Buffer
	if err := md.Convert(contents, &buf); err != nil {
		// If conversion fails, return plain text (escaped)
		return template.HTML("<pre>" + htmlpkg.EscapeString(string(contents)) + "</pre>")
	}

	// Rewrite relative image URLs to point to the assets directory
	htmlStr := buf.String()

	// Find and replace all img src attributes
	// We look for patterns like: <img src="filename.ext" where filename doesn't start with / or http
	lines := strings.Split(htmlStr, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "<img") || !strings.Contains(line, "src=") {
			continue
		}

		// Find src=" and extract the value
		for {
			start := strings.Index(line, `src="`)
			if start == -1 {
				break
			}

			end := strings.Index(line[start+5:], `"`)
			if end == -1 {
				break
			}

			srcValue := line[start+5 : start+5+end]

			// Only rewrite if it's a relative image path
			if !strings.HasPrefix(srcValue, "/") &&
				!strings.HasPrefix(srcValue, "http://") &&
				!strings.HasPrefix(srcValue, "https://") &&
				isImageFile(srcValue) {
				newSrc := "/" + Config.RepoName + "/assets/" + filepath.Base(srcValue)
				line = line[:start+5] + newSrc + line[start+5+end:]
				lines[i] = line
			}

			// Move past this occurrence to find the next one
			if start+5+end+1 < len(line) {
				line = line[start+5+end+1:]
			} else {
				break
			}
		}
	}
	htmlStr = strings.Join(lines, "\n")

	return template.HTML(htmlStr)
}

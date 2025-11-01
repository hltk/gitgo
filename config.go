package main

import (
	"html/template"
	"time"
)

var (
	funcmap = template.FuncMap{
		"now": time.Now,
		"formatDate": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("2006-01-02 15:04")
		},
		"timeAgo": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			duration := time.Since(t)

			seconds := int(duration.Seconds())
			minutes := int(duration.Minutes())
			hours := int(duration.Hours())
			days := hours / 24
			weeks := days / 7
			months := days / 30
			years := days / 365

			if years > 0 {
				if years == 1 {
					return "1 year ago"
				}
				return fmt.Sprintf("%d years ago", years)
			}
			if months > 0 {
				if months == 1 {
					return "1 month ago"
				}
				return fmt.Sprintf("%d months ago", months)
			}
			if weeks > 0 {
				if weeks == 1 {
					return "1 week ago"
				}
				return fmt.Sprintf("%d weeks ago", weeks)
			}
			if days > 0 {
				if days == 1 {
					return "1 day ago"
				}
				return fmt.Sprintf("%d days ago", days)
			}
			if hours > 0 {
				if hours == 1 {
					return "1 hour ago"
				}
				return fmt.Sprintf("%d hours ago", hours)
			}
			if minutes > 0 {
				if minutes == 1 {
					return "1 minute ago"
				}
				return fmt.Sprintf("%d minutes ago", minutes)
			}
			if seconds > 0 {
				if seconds == 1 {
					return "1 second ago"
				}
				return fmt.Sprintf("%d seconds ago", seconds)
			}
			return "just now"
		},
	}
	templ *template.Template
	t     *template.Template
)

type ConfigStruct struct {
	// the following are configured below:
	MaxSummaryLen int
	GitUrl        string
	// the following are received from the command line arguments and flags:
	RepoName   string
	InstallDir string
	DestDir    string
	Force      bool
}

var Config = ConfigStruct{MaxSummaryLen: 20, GitUrl: "github.com/hltk"}

type GlobalRenderData struct {
	Config      *ConfigStruct
	Links       []LinkListElem
	LogoFound   bool
	CommitCount int
	BranchName  string
	BranchCount int
	TagCount    int
}

var GlobalDataGlobal = GlobalRenderData{Config: &Config,
	Links: []LinkListElem{{"branches", "/branches.html"}, {"tags", "/tags.html"}, {"tree", "/tree"}, {"log", "/log"}}}

// GlobalFullTree stores the complete repository tree structure (flattened)
var GlobalFullTree []FlatTreeItem

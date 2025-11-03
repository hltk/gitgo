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
		"mul": func(a int, b float64) float64 {
			return float64(a) * b
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

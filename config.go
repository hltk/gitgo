package main

import (
	"html/template"
	"time"
)

var (
	funcmap = template.FuncMap{
		"now": time.Now,
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
	Config    *ConfigStruct
	Links     []LinkListElem
	LogoFound bool
}

var GlobalDataGlobal = GlobalRenderData{Config: &Config,
	Links: []LinkListElem{{"summary", "/"}, {"tree", "/tree"}, {"log", "/log"}}}

package main

import (
	"html/template"
	"time"
)

type LinkListElem struct {
	Pretty string
	Link   string
}

type CommitListElem struct {
	Link       string
	Msg        string
	Name       string
	Date       time.Time
	AbbrevHash string
}

type FileListElem struct {
	Name           string
	Link           string
	IsFile         bool
	Mode           string
	Size           int
	LastModified   time.Time
	LastCommitMsg  string
	LastCommitLink string
}

type Contributor struct {
	Name  string
	Email string
}

type IndexRenderData struct {
	GlobalData     *GlobalRenderData
	ReadmeFile     FileViewRenderData
	ReadmeFound    bool
	LicenseFile    FileViewRenderData
	LicenseFound   bool
	LatestCommit   CommitListElem
	CommitFound    bool
	RootTree       []FileListElem
	TreeFound      bool
	Contributors   []Contributor
	ContributorsCt int
}

type LogRenderData struct {
	GlobalData *GlobalRenderData
	Commits    []CommitListElem
}

type TreeRenderData struct {
	GlobalData  *GlobalRenderData
	Files       []FileListElem
	CurrentPath string
	ParentPath  string
	HasParent   bool
}

type FileViewRenderData struct {
	Name  string
	Lines []template.HTML
}

type FileRenderData struct {
	GlobalData   *GlobalRenderData
	FileViewData FileViewRenderData
}

type CommitRenderData struct {
	GlobalData    *GlobalRenderData
	Id            string
	Author        string
	Mail          string
	Parents       []string
	HasAnyParents bool
	Date          time.Time
	MsgLines      []string
	DiffStatLines []template.HTML
}

type RefListElem struct {
	Name       string
	Type       string // "branch" or "tag"
	CommitHash string
	LogLink    string
}

type RefsRenderData struct {
	GlobalData *GlobalRenderData
	Branches   []RefListElem
	Tags       []RefListElem
}

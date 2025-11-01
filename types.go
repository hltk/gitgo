package main

import (
	"time"
)

type LinkListElem struct {
	Pretty string
	Link   string
}

type CommitListElem struct {
	Link string
	Msg  string
	Name string
	Date time.Time
}

type FileListElem struct {
	Name   string
	Link   string
	IsFile bool
	Mode   string
	Size   int
}

type IndexRenderData struct {
	GlobalData   *GlobalRenderData
	ReadmeFile   FileViewRenderData
	ReadmeFound  bool
	LicenseFile  FileViewRenderData
	LicenseFound bool
}

type LogRenderData struct {
	GlobalData *GlobalRenderData
	Commits    []CommitListElem
}

type TreeRenderData struct {
	GlobalData *GlobalRenderData
	Files      []FileListElem
}

type FileViewRenderData struct {
	Name  string
	Lines []string
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
	DiffStatLines []string
}

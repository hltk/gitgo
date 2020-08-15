package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"strings"
	"time"
	"os"

	"github.com/libgit2/git2go/v30"
)

var (
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
}

var Config = ConfigStruct{MaxSummaryLen: 20, GitUrl: "git.hltk.fi"}

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
	Date   time.Time
}

type GlobalRenderData struct {
	Config *ConfigStruct
	Links  []LinkListElem
}

var GlobalDataGlobal = GlobalRenderData{Config: &Config,
	Links: []LinkListElem{{"summary", "/"}, {"tree", "/tree"}, {"log", "/log"}}}

type IndexRenderData struct {
	GlobalData  *GlobalRenderData
	ReadmeFile  FileViewRenderData
	ReadmeFound bool
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

func getcommitlog(repo *git.Repository, head *git.Oid) []CommitListElem {
	var commitlist []CommitListElem

	walk, err := repo.Walk()
	if err != nil {
		log.Fatal(err)
	}
	if err = walk.Push(head); err != nil {
		log.Fatal(err)
	}
	walk.SimplifyFirstParent()

	id := git.Oid{}

	for {
		if err = walk.Next(&id); err != nil {
			// at the end of the commit history
			break
		}

		commit, err := repo.LookupCommit(&id)
		if err != nil {
			log.Fatal(err)
		}

		var parents []string

		parentcount := int(commit.ParentCount())
		parentcountispositive := parentcount > 0

		for i := 0; i < parentcount; i++ {
			parents = append(parents, commit.Parent(uint(i)).TreeId().String())
		}

		var diffstat = ""

		if parentcountispositive {
			opts, err := git.DefaultDiffOptions()
			if err != nil {
				log.Fatal(err)
			}
			opts.Flags |= git.DiffDisablePathspecMatch | git.DiffIgnoreSubmodules | git.DiffIncludeTypeChange

			parenttree, err := commit.Parent(0).Tree()
			if err != nil {
				log.Fatal(err)
			}
			tree, err := commit.Tree()
			if err != nil {
				log.Fatal(err)
			}

			diff, err := repo.DiffTreeToTree(parenttree, tree, &opts)
			if err != nil {
				log.Fatal(err)
			}
			fopts, err := git.DefaultDiffFindOptions()
			if err != nil {
				log.Fatal(err)
			}
			fopts.Flags |= git.DiffFindRenames | git.DiffFindCopies | git.DiffFindExactMatchOnly
			err = diff.FindSimilar(&fopts)
			if err != nil {
				log.Fatal(err)
			}
			numdeltas, err := diff.NumDeltas()

			for i := 0; i < numdeltas; i++ {
				delta, err := diff.GetDelta(i)
				if err != nil {
					log.Fatal(err)
				}
				patch, err := diff.Patch(i)
				if err != nil {
					log.Fatal(err)
				}
				if (delta.Flags & git.DiffFlagBinary) > 0 {
					continue
				}
				str, err := patch.String()
				if err != nil {
					log.Fatal(err)
				}

				diffstat += str + "\n"
			}
		}

		commitfilename := Config.DestDir + "commit/" + commit.TreeId().String() + ".html"
		commitfile, err := os.Create(commitfilename)
		if err != nil {
			log.Fatal(err)
		}
		err = t.ExecuteTemplate(commitfile, "commit.html", CommitRenderData{GlobalData: &GlobalDataGlobal,
			Author:        commit.Author().Name,
			Mail:          commit.Author().Email,
			Date:          commit.Author().When,
			Id:            commit.TreeId().String(),
			Parents:       parents,
			HasAnyParents: parentcountispositive,
			MsgLines:      strings.Split(strings.TrimRight(commit.Message(), "\n"), "\n"),
			DiffStatLines: strings.Split(strings.TrimRight(diffstat, "\n"), "\n")})

		if err != nil {
			log.Print("execute:", err)
		}
		commitfile.Sync()
		defer commitfile.Close()

		link := "/commit/" + commit.TreeId().String() + ".html"
		msg := capcommitsummary(commit.Summary())
		name := commit.Author().Name
		date := commit.Author().When

		commitlist = append(commitlist, CommitListElem{link, msg, name, date})
	}

	return commitlist

}

func indextreerecursive(repo *git.Repository, tree *git.Tree, path string) {
	var filelist []FileListElem
	count := int(tree.EntryCount())
	for i := 0; i < count; i++ {
		entry := tree.EntryByIndex(uint64(i))
		if entry.Type == git.ObjectTree {
			// possibly very slow?
			nexttree, err := repo.LookupTree(entry.Id)
			if err != nil {
				log.Fatal()
			}

			newpath := path + entry.Name + "/"

			makedir(Config.DestDir + newpath)

			filelist = append(filelist, FileListElem{entry.Name + "/", "/" + newpath, false, time.Now()})

			indextreerecursive(repo, nexttree, newpath)
		}
		if entry.Type == git.ObjectBlob {
			blob, err := repo.LookupBlob(entry.Id)
			if err != nil {
				log.Fatal()
			}

			newpath := path + entry.Name
			file, err := os.Create(Config.DestDir + newpath + ".html")

			if err != nil {
				log.Fatal(err)
			}

			lines := contentstolines(blob.Contents(), int(blob.Size()))

			err = t.ExecuteTemplate(file, "file.html", FileRenderData{&GlobalDataGlobal, FileViewRenderData{entry.Name, lines}})
			if err != nil {
				log.Print("execute:", err)
			}
			file.Sync()
			defer file.Close()

			filelist = append(filelist, FileListElem{entry.Name, "/" + newpath + ".html", true, time.Now()})
		}
		if entry.Type == git.ObjectCommit {
			log.Print("FATAL: submodules not implemented")
			log.Fatal()
		}
	}
	treefile, err := os.Create(Config.DestDir + path + "index.html")
	if err != nil {
		log.Fatal(err)
	}
	err = t.ExecuteTemplate(treefile, "tree.html", TreeRenderData{GlobalData: &GlobalDataGlobal, Files: filelist})
	if err != nil {
		log.Print("execute:", err)
	}
	treefile.Sync()
	defer treefile.Close()
}

func indextree(repo *git.Repository, head *git.Oid) {
	commit, err := repo.LookupCommit(head)
	if err != nil {
		log.Fatal(err)
	}
	tree, err := commit.Tree()
	if err != nil {
		log.Fatal(err)
	}
	indextreerecursive(repo, tree, "tree/")
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.StringVar(&Config.DestDir, "destdir", ".", "target directory")
	flag.StringVar(&Config.InstallDir, "installdir", "/usr/share/gitgo", "install directory")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: gitgo [options] <git repo>\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		flag.Usage()
		return
	}

	fixpath(&Config.DestDir)
	fixpath(&Config.InstallDir)

	repo, err := git.OpenRepositoryExtended(args[0], git.RepositoryOpenNoSearch, "")
	if err != nil {
		log.Fatal(err)
	}

	obj, _, err := repo.RevparseExt("HEAD")
	if err != nil {
		log.Fatal(err)
	}

	head := obj.Id()

	Config.RepoName = cleanname(args[0])

	Config.DestDir += Config.RepoName + "/"

	makedir(Config.DestDir)

	templ = template.New("")

	t, err = templ.ParseGlob(Config.InstallDir + "templates/*.html")

	if err != nil {
		log.Print("parse:", err)
	}

	var (
		readmefile  FileViewRenderData
		readmefiles = [...]string{"HEAD:README", "HEAD:README.md"}
		readmefound = false
	)

	for _, file := range readmefiles {
		fileobj, _, err := repo.RevparseExt(file)
		if err == nil && fileobj.Type() == git.ObjectBlob {
			blob, err := fileobj.AsBlob()

			if err != nil {
				log.Fatal(err)
			}

			lines := contentstolines(blob.Contents(), int(blob.Size()))

			readmefile.Name = strings.TrimPrefix(file, "HEAD:")
			readmefile.Lines = lines
			readmefound = true
			break
		}
	}

	// var licensefiles = [...]string{"HEAD:LICENSE", "HEAD:COPYING", "HEAD:LICENSE.md"}
	// TODO: make the LICENSE file easily accessible (the same way as README)

	indexfile, err := os.Create(Config.DestDir + "index.html")
	if err != nil {
		log.Fatal(err)
	}
	err = t.ExecuteTemplate(indexfile, "index.html", IndexRenderData{&GlobalDataGlobal, readmefile, readmefound})
	if err != nil {
		log.Print("execute:", err)
	}
	indexfile.Sync()
	defer indexfile.Close()

	// TODO: submodules are listed in .submodules

	makedir(Config.DestDir + "commit")
	makedir(Config.DestDir + "tree")
	makedir(Config.DestDir + "log")

	commitlist := getcommitlog(repo, head)

	logfile, err := os.Create(Config.DestDir + "log/index.html")
	if err != nil {
		log.Fatal(err)
	}
	err = t.ExecuteTemplate(logfile, "log.html", LogRenderData{GlobalData: &GlobalDataGlobal, Commits: commitlist})
	if err != nil {
		log.Print("execute:", err)
	}
	logfile.Sync()
	defer logfile.Close()

	indextree(repo, head)

	// TODO: add refs.html for branches and tags
}

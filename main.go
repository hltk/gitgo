package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"strings"

	"github.com/libgit2/git2go/v30"
)

var (
	templ *template.Template
	t     *template.Template
)

type ConfigStruct struct {
	InstallDir string
	DestDir    string
}

var Config ConfigStruct

type LinkListElem struct {
	Pretty string
	Link   string
}

type CommitListElem struct {
	Link string
	Msg  string
	Name string
	Date string
}

type FileListElem struct {
	Name string
	Link string
	Date string
}

type GlobalRenderData struct {
	RepoName string
	Links    []LinkListElem
}

var GlobalDataGlobal GlobalRenderData

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

func writetofile(file *os.File, str string) {
	_, err := io.WriteString(file, str)
	if err != nil {
		log.Fatal(err)
	}
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

		commitfile_name := Config.DestDir + "commit/" + commit.TreeId().String() + ".html"
		commitfile := openfile(commitfile_name)
		// TODO: write commit info to file
		closefile(commitfile)

		link := "/commit/" + commit.TreeId().String() + ".html"
		msg := commit.Summary()
		name := commit.Author().Name
		date := commit.Author().When.Format("15:04:05 2006-01-02")

		commitlist = append(commitlist, CommitListElem{link, msg, name, date})
	}

	return commitlist

}

func contentstolines(contents []byte, size int) []string {
	var lines = []string{""}

	for i := 0; i < size; i++ {
		c := contents[i]
		if c != '\n' {
			lines[len(lines)-1] += string(c)
		} else if i + 1 != size {
			lines = append(lines, "")
		}
	}

	return lines
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

			filelist = append(filelist, FileListElem{entry.Name + "/", "/" + newpath, "TODO"})

			indextreerecursive(repo, nexttree, newpath)
		}
		if entry.Type == git.ObjectBlob {
			blob, err := repo.LookupBlob(entry.Id)
			if err != nil {
				log.Fatal()
			}

			newpath := path + entry.Name
			file := openfile(Config.DestDir + newpath + ".html")

			lines := contentstolines(blob.Contents(), int(blob.Size()))

			err = t.ExecuteTemplate(file, "file.html", FileRenderData{&GlobalDataGlobal, FileViewRenderData{entry.Name, lines}})
			if err != nil {
				log.Print("execute:", err)
			}
			closefile(file)

			filelist = append(filelist, FileListElem{entry.Name, "/" + newpath + ".html", "TODO"})
		}
		if entry.Type == git.ObjectCommit {
			log.Print("FATAL: submodules not implemented")
			log.Fatal()
		}
	}
	treefile := openfile(Config.DestDir + path + "index.html")
	err := t.ExecuteTemplate(treefile, "tree.html", TreeRenderData{GlobalData: &GlobalDataGlobal, Files: filelist})
	if err != nil {
		log.Print("execute:", err)
	}
	closefile(treefile)
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

	GlobalDataGlobal.RepoName = cleanname(args[0])
	GlobalDataGlobal.Links = []LinkListElem{{"summary", "/"}, {"tree", "/tree"}, {"log", "/log"}}

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

	indexfile := openfile(Config.DestDir + "index.html")
	err = t.ExecuteTemplate(indexfile, "index.html", IndexRenderData{&GlobalDataGlobal, readmefile, readmefound})
	if err != nil {
		log.Print("execute:", err)
	}
	closefile(indexfile)

	// TODO: submodules are listed in .submodules

	makedir(Config.DestDir + "commit")
	makedir(Config.DestDir + "tree")
	makedir(Config.DestDir + "log")

	commitlist := getcommitlog(repo, head)

	logfile := openfile(Config.DestDir + "log/index.html")
	err = t.ExecuteTemplate(logfile, "log.html", LogRenderData{GlobalData: &GlobalDataGlobal, Commits: commitlist})
	if err != nil {
		log.Print("execute:", err)
	}
	closefile(logfile)

	indextree(repo, head)

	// TODO: add refs.html for branches and tags
}

package main

import (
"flag"
"fmt"
"io"
"log"
"os"
"strings"

"github.com/libgit2/git2go/v30"
)

var licensefiles = [...]string{"HEAD:LICENSE", "HEAD:LICENSE.md", "HEAD:COPYING"}
var readmefiles = [...]string{"HEAD:README", "HEAD:README.md"}
var mainfiles = [...]string{"index.html", "tree.html", "log.html"}

func writetofile(file *os.File, str string) {
_, err := io.WriteString(file, str)
if err != nil {
	log.Fatal(err)
}
}

func openfile(str string) *os.File {
file, err := os.Create(str)
if err != nil {
		log.Fatal(err)
	}
	return file

}

func closefile(file *os.File) {
	if err := file.Sync(); err != nil {
		log.Fatal(err)
	}

	if err := file.Close(); err != nil {
		log.Fatal(err)
	}
}

func writelogline(commit *git.Commit, logfile *os.File) {
	commit_msg := commit.Summary()
	name := commit.Author().Name
	date := commit.Author().When.Format("15:04:05 2006-01-02")
	link := commit.TreeId().String() + ".html"
	writetofile(logfile, "<a href=\"/commit/" + link + "\">")
	writetofile(logfile, commit_msg + "<br>")
	writetofile(logfile, "</a>")
	writetofile(logfile, name + "<br>")
	writetofile(logfile, date + "<br>")
	writetofile(logfile, "<br>")
}

func writelogtofile(repo *git.Repository, head *git.Oid, logfile *os.File) {
	writetofile(logfile, "commit log:\n<br><br>\n")
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

		commitfile_name := "commit/" + commit.TreeId().String() + ".html"

		commitfile := openfile(commitfile_name)

		closefile(commitfile)


		if err != nil {
			log.Fatal(err)
		}

		writelogline(commit, logfile)
	}

}

func writetreetofile(repo *git.Repository, head *git.Oid, logfile *os.File) {
	commit, err := repo.LookupCommit(head)
	if err != nil {
		log.Fatal(err)
	}
	tree, err := commit.Tree()
	if err != nil {
		log.Fatal(err)
	}

}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: FIXME [options...] <git repo>\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	args := flag.Args()

	if len(args) != 1 {
		flag.Usage()
		return
	}

	reponame := args[0]

	repo, err := git.OpenRepositoryExtended(reponame, git.RepositoryOpenNoSearch, "")
	if err != nil {
		log.Fatal(err)
	}

	// ignore the reference
	obj, _, err := repo.RevparseExt("HEAD")
	if err != nil {
		log.Fatal(err)
	}
	head := obj.Id()

	indexfile := openfile("index.html")
	for _, file := range licensefiles {
		fileObj, _, err := repo.RevparseExt(file)
		if err == nil && fileObj.Type() == git.ObjectBlob {
			realname := strings.TrimPrefix(file, "HEAD:")
			writetofile(indexfile, "<a href=\"/" + "file/" + realname + "\">LICENSE</a><br>")
			break
		}
	}

	for _, file := range readmefiles {
		fileObj, _, err := repo.RevparseExt(file)
		if err == nil && fileObj.Type() == git.ObjectBlob {
			realname := strings.TrimPrefix(file, "HEAD:")
			writetofile(indexfile, "<a href=\"/" + "file/" + realname + "\">README</a><br>")
			break
		}
	}

	for _, file := range mainfiles {
		cleanname := strings.TrimSuffix(file, ".html")
		writetofile(indexfile, "<a href=\"/" + file + "\">" + cleanname + "</a><br>")
	}
	closefile(indexfile)

	// TODO: submodules are listed in .submodules

	if _, err := os.Stat("commit"); os.IsNotExist(err) {
		os.Mkdir("commit", 755)
	}

	logfile := openfile("log.html")
	writelogtofile(repo, head, logfile)
	closefile(logfile)

	treefile := openfile("tree.html")
	writetreetofile(repo, head, treefile)
	closefile(treefile)

	// TODO: add refs.html for branches and tags
	// refsfile := openfile("refs.html")
	// closefile(refsfile)
}

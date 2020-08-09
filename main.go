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

// var licensefiles = [...]string{"HEAD:LICENSE", "HEAD:LICENSE.md", "HEAD:COPYING"}
var readmefiles = [...]string{"HEAD:README", "HEAD:README.md"}
var mainfiles = [...]string{"index.html", "tree", "log"}

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
	writetofile(logfile, "<a href=\"/commit/"+link+"\">")
	writetofile(logfile, commit_msg+"<br>")
	writetofile(logfile, "</a>")
	writetofile(logfile, name+"<br>")
	writetofile(logfile, date+"<br>")
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

func writetreetofilerecursive(repo *git.Repository, tree *git.Tree, treefile *os.File, path string) {
	count := tree.EntryCount()
	for i := uint64(0); i < count; i++ {
		entry := tree.EntryByIndex(i)
		if entry.Type == git.ObjectTree {
			// possibly very slow?
			nexttree, err := repo.LookupTree(entry.Id)
			if err != nil {
				log.Fatal()
			}
			newpath := path + entry.Name + "/"
			if _, err := os.Stat(newpath); os.IsNotExist(err) {
				os.Mkdir(newpath, 755)
			}
			writetofile(treefile, "<a href=\"/"+newpath+"\">"+entry.Name+"/</a><br>")
			newtreefile := openfile(newpath + "index.html")
			writetreetofilerecursive(repo, nexttree, newtreefile, newpath)
			closefile(newtreefile)
		}
		if entry.Type == git.ObjectBlob {
			blob, err := repo.LookupBlob(entry.Id)
			if err != nil {
				log.Fatal()
			}
			size := blob.Size()
			contents := blob.Contents()

			newpath := path + entry.Name
			file := openfile(newpath + ".html")
			writtensize, err := file.Write(contents)
			if int64(writtensize) != size {
				log.Fatal()
			}
			if err != nil {
				log.Fatal(err)
			}
			closefile(file)
			writetofile(treefile, "<a href=\"/"+newpath+".html\">"+entry.Name+"</a><br>")
		}
		if entry.Type == git.ObjectCommit {
			log.Print("FATAL: submodules not implemented")
			log.Fatal()
		}
	}
}

func writetreetofile(repo *git.Repository, head *git.Oid, treefile *os.File) {
	commit, err := repo.LookupCommit(head)
	if err != nil {
		log.Fatal(err)
	}
	tree, err := commit.Tree()
	if err != nil {
		log.Fatal(err)
	}
	writetreetofilerecursive(repo, tree, treefile, "tree/")
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

	cleanreponame := strings.TrimSuffix(reponame, ".git")

	lastslash := strings.LastIndex(cleanreponame, "/")

	if lastslash != -1 {
		cleanreponame = cleanreponame[lastslash+1:]
	}

	writetofile(indexfile, "<h1>"+cleanreponame+"</h1>")

	for _, file := range mainfiles {
		writetofile(indexfile, "<a href=\"/"+file+"\">"+file+"</a><br>")
	}

	// TODO: make LICENSE file easily accessible
	// for _, file := range licensefiles {
	// 	fileObj, _, err := repo.RevparseExt(file)
	// 	if err == nil && fileObj.Type() == git.ObjectBlob {
	// 		realname := strings.TrimPrefix(file, "HEAD:")
	// 		writetofile(indexfile, "<a href=\"/"+"tree/"+realname+".html\">LICENSE</a><br>")
	// 		break
	// 	}
	// }

	for _, file := range readmefiles {
		fileobj, _, err := repo.RevparseExt(file)
		if err == nil && fileobj.Type() == git.ObjectBlob {
			writetofile(indexfile, "<hr>")

			blob, err := fileobj.AsBlob()

			if err != nil {
				log.Fatal(err)
			}

			size := blob.Size()
			contents := blob.Contents()

			writtensize, err := indexfile.Write(contents)
			if int64(writtensize) != size {
				log.Fatal()
			}
			if err != nil {
				log.Fatal(err)
			}
			break
		}
	}

	closefile(indexfile)

	// TODO: submodules are listed in .submodules

	if _, err := os.Stat("commit"); os.IsNotExist(err) {
		os.Mkdir("commit", 755)
	}

	if _, err := os.Stat("log"); os.IsNotExist(err) {
		os.Mkdir("log", 755)
	}
	logfile := openfile("log/index.html")
	writelogtofile(repo, head, logfile)
	closefile(logfile)

	if _, err := os.Stat("tree"); os.IsNotExist(err) {
		os.Mkdir("tree", 755)
	}

	treefile := openfile("tree/index.html")
	writetreetofile(repo, head, treefile)
	closefile(treefile)

	// TODO: add refs.html for branches and tags
	// refsfile := openfile("refs.html")
	// closefile(refsfile)
}

package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	git "github.com/libgit2/git2go/v34"
)

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

		commitfilename := filepath.Join(Config.DestDir, "commit", commit.TreeId().String()+".html")
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

		link := filepath.Join("/commit", commit.TreeId().String()+".html")
		msg := commit.Summary()
		if len(msg) > Config.MaxSummaryLen {
			msg = msg[:Config.MaxSummaryLen-3] + "..."
		}

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

		filemode := entry.Filemode

		mode := os.FileMode(filemode).String()

		if filemode == git.FilemodeTree {
			mode = "d" + mode[1:]
		}
		if filemode == git.FilemodeLink {
			mode = "l" + mode[1:]
		}
		if filemode == git.FilemodeCommit {
			mode = "m" + mode[1:]
		}

		size := 0

		blob, err := repo.LookupBlob(entry.Id)

		if err == nil {
			size = int(blob.Size())
		}

		if entry.Type == git.ObjectTree {
			// possibly very slow?
			nexttree, err := repo.LookupTree(entry.Id)
			if err != nil {
				log.Fatal()
			}

			newpath := filepath.Join(path, entry.Name)

			err = makedir(filepath.Join(Config.DestDir, newpath))
			if err != nil {
				log.Fatal(err)
			}

			filelist = append(filelist, FileListElem{entry.Name + "/", newpath, false, mode, size})
			indextreerecursive(repo, nexttree, newpath)
		}
		if entry.Type == git.ObjectBlob {
			blob, err := repo.LookupBlob(entry.Id)
			if err != nil {
				log.Fatal()
			}

			newpath := filepath.Join(path, entry.Name)
			file, err := os.Create(filepath.Join(Config.DestDir, newpath+".html"))

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

			filelist = append(filelist, FileListElem{entry.Name, newpath + ".html", true, mode, size})
		}
		if entry.Type == git.ObjectCommit {
			log.Print("FATAL: submodules not implemented")
			log.Fatal()
		}
	}
	treefile, err := os.Create(filepath.Join(Config.DestDir, path, "index.html"))
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
	indextreerecursive(repo, tree, "/tree")
}

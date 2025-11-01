package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	git "github.com/libgit2/git2go/v34"
)

func getCommitLog(repo *git.Repository, head *git.Oid) []CommitListElem {
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
		abbrevHash := commit.TreeId().String()[:8]

		commitlist = append(commitlist, CommitListElem{
			Link:       link,
			Msg:        msg,
			Name:       name,
			Date:       date,
			AbbrevHash: abbrevHash,
		})
	}

	return commitlist
}

// getLastModifiedDate returns the date of the last commit that modified the given path
func getLastModifiedDate(repo *git.Repository, repoPath string) time.Time {
	head, err := repo.Head()
	if err != nil {
		return time.Time{}
	}
	defer head.Free()

	walk, err := repo.Walk()
	if err != nil {
		return time.Time{}
	}
	defer walk.Free()

	if err = walk.Push(head.Target()); err != nil {
		return time.Time{}
	}

	// Remove leading /tree from path for pathspec
	cleanPath := strings.TrimPrefix(repoPath, "/tree/")
	if cleanPath == "/tree" {
		cleanPath = ""
	}

	walk.SimplifyFirstParent()

	id := git.Oid{}
	for {
		if err = walk.Next(&id); err != nil {
			break
		}

		commit, err := repo.LookupCommit(&id)
		if err != nil {
			continue
		}

		tree, err := commit.Tree()
		if err != nil {
			commit.Free()
			continue
		}

		// For the root tree, return the commit date
		if cleanPath == "" {
			date := commit.Author().When
			tree.Free()
			commit.Free()
			return date
		}

		// Check if this path exists in this commit
		_, err = tree.EntryByPath(cleanPath)
		tree.Free()

		if err == nil {
			// Path exists in this commit, check if it was modified
			if commit.ParentCount() == 0 {
				// Initial commit
				date := commit.Author().When
				commit.Free()
				return date
			}

			parent := commit.Parent(0)
			if parent == nil {
				date := commit.Author().When
				commit.Free()
				return date
			}

			parentTree, err := parent.Tree()
			if err != nil {
				parent.Free()
				date := commit.Author().When
				commit.Free()
				return date
			}

			// Check if path exists in parent
			_, err = parentTree.EntryByPath(cleanPath)
			parentTree.Free()
			parent.Free()

			if err != nil {
				// Path didn't exist in parent, so it was added in this commit
				date := commit.Author().When
				commit.Free()
				return date
			}

			// For directories and files, we need to check if content changed
			// For simplicity, we'll use the commit where the path first appears or changes
			opts, err := git.DefaultDiffOptions()
			if err != nil {
				commit.Free()
				continue
			}

			parentCommit := commit.Parent(0)
			if parentCommit == nil {
				date := commit.Author().When
				commit.Free()
				return date
			}

			currentTree, _ := commit.Tree()
			parentTree2, _ := parentCommit.Tree()

			if currentTree != nil && parentTree2 != nil {
				opts.Pathspec = []string{cleanPath}
				diff, err := repo.DiffTreeToTree(parentTree2, currentTree, &opts)

				if currentTree != nil {
					currentTree.Free()
				}
				if parentTree2 != nil {
					parentTree2.Free()
				}
				parentCommit.Free()

				if err == nil {
					numDeltas, _ := diff.NumDeltas()
					diff.Free()

					if numDeltas > 0 {
						// This commit modified the path
						date := commit.Author().When
						commit.Free()
						return date
					}
				} else {
					if diff != nil {
						diff.Free()
					}
				}
			} else {
				if currentTree != nil {
					currentTree.Free()
				}
				if parentTree2 != nil {
					parentTree2.Free()
				}
				parentCommit.Free()
			}
		}
		commit.Free()
	}

	return time.Time{}
}

// getLastCommitInfo returns the date, message, and link of the last commit that modified the given path
func getLastCommitInfo(repo *git.Repository, repoPath string) (time.Time, string, string) {
	head, err := repo.Head()
	if err != nil {
		return time.Time{}, "", ""
	}
	defer head.Free()

	walk, err := repo.Walk()
	if err != nil {
		return time.Time{}, "", ""
	}
	defer walk.Free()

	if err = walk.Push(head.Target()); err != nil {
		return time.Time{}, "", ""
	}

	// Remove leading /tree from path for pathspec
	cleanPath := strings.TrimPrefix(repoPath, "/tree/")
	if cleanPath == "/tree" {
		cleanPath = ""
	}

	walk.SimplifyFirstParent()

	id := git.Oid{}
	for {
		if err = walk.Next(&id); err != nil {
			break
		}

		commit, err := repo.LookupCommit(&id)
		if err != nil {
			continue
		}

		tree, err := commit.Tree()
		if err != nil {
			commit.Free()
			continue
		}

		// For the root tree, return the commit info
		if cleanPath == "" {
			date := commit.Author().When
			msg := commit.Summary()
			link := "/commit/" + commit.TreeId().String() + ".html"
			tree.Free()
			commit.Free()
			return date, msg, link
		}

		// Check if this path exists in this commit
		_, err = tree.EntryByPath(cleanPath)
		tree.Free()

		if err == nil {
			// Path exists in this commit, check if it was modified
			if commit.ParentCount() == 0 {
				// Initial commit
				date := commit.Author().When
				msg := commit.Summary()
				link := "/commit/" + commit.TreeId().String() + ".html"
				commit.Free()
				return date, msg, link
			}

			parent := commit.Parent(0)
			if parent == nil {
				date := commit.Author().When
				msg := commit.Summary()
				link := "/commit/" + commit.TreeId().String() + ".html"
				commit.Free()
				return date, msg, link
			}

			parentTree, err := parent.Tree()
			if err != nil {
				parent.Free()
				date := commit.Author().When
				msg := commit.Summary()
				link := "/commit/" + commit.TreeId().String() + ".html"
				commit.Free()
				return date, msg, link
			}

			// Check if path exists in parent
			_, err = parentTree.EntryByPath(cleanPath)
			parentTree.Free()
			parent.Free()

			if err != nil {
				// Path didn't exist in parent, so it was added in this commit
				date := commit.Author().When
				msg := commit.Summary()
				link := "/commit/" + commit.TreeId().String() + ".html"
				commit.Free()
				return date, msg, link
			}

			// For directories and files, we need to check if content changed
			opts, err := git.DefaultDiffOptions()
			if err != nil {
				commit.Free()
				continue
			}

			parentCommit := commit.Parent(0)
			if parentCommit == nil {
				date := commit.Author().When
				msg := commit.Summary()
				link := "/commit/" + commit.TreeId().String() + ".html"
				commit.Free()
				return date, msg, link
			}

			currentTree, _ := commit.Tree()
			parentTree2, _ := parentCommit.Tree()

			if currentTree != nil && parentTree2 != nil {
				opts.Pathspec = []string{cleanPath}
				diff, err := repo.DiffTreeToTree(parentTree2, currentTree, &opts)

				if currentTree != nil {
					currentTree.Free()
				}
				if parentTree2 != nil {
					parentTree2.Free()
				}
				parentCommit.Free()

				if err == nil {
					numDeltas, _ := diff.NumDeltas()
					diff.Free()

					if numDeltas > 0 {
						// This commit modified the path
						date := commit.Author().When
						msg := commit.Summary()
						link := "/commit/" + commit.TreeId().String() + ".html"
						commit.Free()
						return date, msg, link
					}
				} else {
					if diff != nil {
						diff.Free()
					}
				}
			} else {
				if currentTree != nil {
					currentTree.Free()
				}
				if parentTree2 != nil {
					parentTree2.Free()
				}
				parentCommit.Free()
			}
		}
		commit.Free()
	}

	return time.Time{}, "", ""
}

// getRootTreeFileList returns the file list for the root tree (non-recursive)
func getRootTreeFileList(repo *git.Repository, tree *git.Tree) []FileListElem {
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
			lastModified, commitMsg, commitLink := getLastCommitInfo(repo, filepath.Join("/tree", entry.Name))
			filelist = append(filelist, FileListElem{entry.Name + "/", filepath.Join("/tree", entry.Name), false, mode, size, lastModified, commitMsg, commitLink})
		}
		if entry.Type == git.ObjectBlob {
			lastModified, commitMsg, commitLink := getLastCommitInfo(repo, filepath.Join("/tree", entry.Name))
			filelist = append(filelist, FileListElem{entry.Name, filepath.Join("/tree", entry.Name) + ".html", true, mode, size, lastModified, commitMsg, commitLink})
		}
	}
	return filelist
}

func indexTreeRecursive(repo *git.Repository, tree *git.Tree, path string) {
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

			err = makeDir(filepath.Join(Config.DestDir, newpath))
			if err != nil {
				log.Fatal(err)
			}

			lastModified, commitMsg, commitLink := getLastCommitInfo(repo, filepath.Join(path, entry.Name))
			filelist = append(filelist, FileListElem{entry.Name + "/", newpath, false, mode, size, lastModified, commitMsg, commitLink})
			indexTreeRecursive(repo, nexttree, newpath)
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

			lines := contentsToLines(blob.Contents(), int(blob.Size()))

			err = t.ExecuteTemplate(file, "file.html", FileRenderData{&GlobalDataGlobal, FileViewRenderData{entry.Name, lines}})
			if err != nil {
				log.Print("execute:", err)
			}
			file.Sync()
			defer file.Close()

			lastModified, commitMsg, commitLink := getLastCommitInfo(repo, filepath.Join(path, entry.Name))
			filelist = append(filelist, FileListElem{entry.Name, newpath + ".html", true, mode, size, lastModified, commitMsg, commitLink})
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

	// Calculate parent path
	var parentPath string
	hasParent := false
	if path != "/tree" {
		// For paths like "/tree/subdir", parent is "/tree"
		// For "/tree/a/b", parent is "/tree/a"
		parentPath = filepath.Dir(path)
		if parentPath != "/tree" {
			parentPath = parentPath + "/"
		}
		hasParent = true
	}
	// For root "/tree", no parent link

	err = t.ExecuteTemplate(treefile, "tree.html", TreeRenderData{
		GlobalData:  &GlobalDataGlobal,
		Files:       filelist,
		CurrentPath: path,
		ParentPath:  parentPath,
		HasParent:   hasParent,
	})
	if err != nil {
		log.Print("execute:", err)
	}
	treefile.Sync()
	defer treefile.Close()
}

func indexTree(repo *git.Repository, head *git.Oid) {
	commit, err := repo.LookupCommit(head)
	if err != nil {
		log.Fatal(err)
	}
	tree, err := commit.Tree()
	if err != nil {
		log.Fatal(err)
	}
	indexTreeRecursive(repo, tree, "/tree")
}

// getContributors walks through the commit history and returns a list of unique contributors
// based on their email addresses
func getContributors(repo *git.Repository, head *git.Oid) []Contributor {
	emailToContributor := make(map[string]Contributor)

	walk, err := repo.Walk()
	if err != nil {
		log.Fatal(err)
	}
	if err = walk.Push(head); err != nil {
		log.Fatal(err)
	}

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

		author := commit.Author()
		email := author.Email

		// Only add if we haven't seen this email before
		if _, exists := emailToContributor[email]; !exists {
			emailToContributor[email] = Contributor{
				Name:  author.Name,
				Email: email,
			}
		}
		commit.Free()
	}

	// Convert map to slice
	contributors := make([]Contributor, 0, len(emailToContributor))
	for _, contributor := range emailToContributor {
		contributors = append(contributors, contributor)
	}

	return contributors
}

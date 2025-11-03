package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
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
			DiffStatLines: highlightDiffLines(diffstat)})

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

// getLastCommitInfo returns the date, message, link, and author name of the last commit that modified the given path
func getLastCommitInfo(repo *git.Repository, repoPath string) (time.Time, string, string, string) {
	head, err := repo.Head()
	if err != nil {
		return time.Time{}, "", "", ""
	}
	defer head.Free()

	walk, err := repo.Walk()
	if err != nil {
		return time.Time{}, "", "", ""
	}
	defer walk.Free()

	if err = walk.Push(head.Target()); err != nil {
		return time.Time{}, "", "", ""
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
			author := commit.Author().Name
			tree.Free()
			commit.Free()
			return date, msg, link, author
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
				author := commit.Author().Name
				commit.Free()
				return date, msg, link, author
			}

			parent := commit.Parent(0)
			if parent == nil {
				date := commit.Author().When
				msg := commit.Summary()
				link := "/commit/" + commit.TreeId().String() + ".html"
				author := commit.Author().Name
				commit.Free()
				return date, msg, link, author
			}

			parentTree, err := parent.Tree()
			if err != nil {
				parent.Free()
				date := commit.Author().When
				msg := commit.Summary()
				link := "/commit/" + commit.TreeId().String() + ".html"
				author := commit.Author().Name
				commit.Free()
				return date, msg, link, author
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
				author := commit.Author().Name
				commit.Free()
				return date, msg, link, author
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
				author := commit.Author().Name
				commit.Free()
				return date, msg, link, author
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
						author := commit.Author().Name
						commit.Free()
						return date, msg, link, author
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

	return time.Time{}, "", "", ""
}

// getRootTreeFileList returns the file list for the root tree (non-recursive)
func getRootTreeFileList(repo *git.Repository, tree *git.Tree) []FileListElem {
	var filelist []FileListElem
	count := int(tree.EntryCount())
	
	// Separate directories and files
	var dirs []*git.TreeEntry
	var files []*git.TreeEntry
	
	for i := 0; i < count; i++ {
		entry := tree.EntryByIndex(uint64(i))
		if entry.Type == git.ObjectTree {
			dirs = append(dirs, entry)
		} else if entry.Type == git.ObjectBlob {
			files = append(files, entry)
		}
	}
	
	// Sort directories and files alphabetically (case-insensitive)
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	
	// Process directories first
	for _, entry := range dirs {
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

		lastModified, commitMsg, commitLink, _ := getLastCommitInfo(repo, filepath.Join("/tree", entry.Name))
		filelist = append(filelist, FileListElem{entry.Name + "/", filepath.Join("/tree", entry.Name), false, mode, size, lastModified, commitMsg, commitLink})
	}
	
	// Process files
	for _, entry := range files {
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

		lastModified, commitMsg, commitLink, _ := getLastCommitInfo(repo, filepath.Join("/tree", entry.Name))
		filelist = append(filelist, FileListElem{entry.Name, filepath.Join("/tree", entry.Name) + ".html", true, mode, size, lastModified, commitMsg, commitLink})
	}
	
	return filelist
}

// getImageFileContents reads image file from the working directory if possible
// This handles Git LFS files which are stored as pointers in the git blob
func getImageFileContents(repo *git.Repository, treePath, filename string) ([]byte, error) {
	// Get the repository's working directory
	workdir := repo.Workdir()
	if workdir == "" {
		return nil, fmt.Errorf("repository has no working directory")
	}

	// Construct the file path relative to the working directory
	// treePath is like "/tree/subdir", we need to strip "/tree" prefix
	relPath := strings.TrimPrefix(treePath, "/tree")
	if relPath != "" && relPath[0] == '/' {
		relPath = relPath[1:]
	}

	filePath := filepath.Join(workdir, relPath, filename)

	// Read the actual file from the working directory
	contents, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return contents, nil
}

func indexTreeRecursive(repo *git.Repository, tree *git.Tree, path string) {
	var filelist []FileListElem
	count := int(tree.EntryCount())
	
	// Separate directories and files
	var dirs []*git.TreeEntry
	var files []*git.TreeEntry
	
	for i := 0; i < count; i++ {
		entry := tree.EntryByIndex(uint64(i))
		if entry.Type == git.ObjectTree {
			dirs = append(dirs, entry)
		} else if entry.Type == git.ObjectBlob {
			files = append(files, entry)
		} else if entry.Type == git.ObjectCommit {
			log.Print("FATAL: submodules not implemented")
			log.Fatal()
		}
	}
	
	// Sort directories and files alphabetically (case-insensitive)
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	
	// Process directories first
	for _, entry := range dirs {
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

		lastModified, commitMsg, commitLink, _ := getLastCommitInfo(repo, filepath.Join(path, entry.Name))
		filelist = append(filelist, FileListElem{entry.Name + "/", newpath, false, mode, size, lastModified, commitMsg, commitLink})
		indexTreeRecursive(repo, nexttree, newpath)
	}
	
	// Process files
	for _, entry := range files {
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

		blob, err = repo.LookupBlob(entry.Id)
		if err != nil {
			log.Fatal()
		}

		newpath := filepath.Join(path, entry.Name)
		file, err := os.Create(filepath.Join(Config.DestDir, newpath+".html"))

		if err != nil {
			log.Fatal(err)
		}

		lines := highlightFileContents(entry.Name, blob.Contents())

		lastModified, commitMsg, commitLink, commitAuthor := getLastCommitInfo(repo, filepath.Join(path, entry.Name))

		currentPath := newpath + ".html"

		err = t.ExecuteTemplate(file, "file.html", FileRenderData{
			GlobalData: &GlobalDataGlobal,
			FileViewData: FileViewRenderData{
				Name:             entry.Name,
				Lines:            lines,
				LastCommitMsg:    commitMsg,
				LastCommitLink:   commitLink,
				LastCommitDate:   lastModified,
				LastCommitAuthor: commitAuthor,
				RepoName:         Config.RepoName,
				CurrentPath:      currentPath,
			},
			FullTree:    GlobalFullTree,
			CurrentPath: currentPath,
		})
		if err != nil {
			log.Print("execute:", err)
		}
		file.Sync()
		defer file.Close()

		// If this is an image file, also write it to the assets directory
		// Read from working directory to handle Git LFS properly
		if isImageFile(entry.Name) {
			imagePath := filepath.Join(Config.DestDir, "assets", entry.Name)
			imageContents, err := getImageFileContents(repo, path, entry.Name)
			if err != nil {
				// Fallback to blob contents if reading from working directory fails
				log.Print("warning: failed to read image from working directory, using blob contents:", err)
				imageContents = blob.Contents()
			}
			err = os.WriteFile(imagePath, imageContents, 0644)
			if err != nil {
				log.Print("write image:", err)
			}
		}

		filelist = append(filelist, FileListElem{entry.Name, newpath + ".html", true, mode, size, lastModified, commitMsg, commitLink})
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

	// Get latest commit info for this folder
	var latestCommit CommitListElem
	commitFound := false
	lastModified, commitMsg, commitLink, commitAuthor := getLastCommitInfo(repo, path)
	if commitMsg != "" && commitLink != "" {
		// Extract treeId from commitLink (format: "/commit/{treeId}.html")
		treeId := strings.TrimPrefix(commitLink, "/commit/")
		treeId = strings.TrimSuffix(treeId, ".html")
		abbrevHash := treeId
		if len(abbrevHash) > 8 {
			abbrevHash = abbrevHash[:8]
		}

		latestCommit = CommitListElem{
			Link:       commitLink,
			Msg:        commitMsg,
			Name:       commitAuthor,
			Date:       lastModified,
			AbbrevHash: abbrevHash,
		}
		commitFound = true
	}

	err = t.ExecuteTemplate(treefile, "tree.html", TreeRenderData{
		GlobalData:   &GlobalDataGlobal,
		Files:        filelist,
		CurrentPath:  path,
		ParentPath:   parentPath,
		HasParent:    hasParent,
		LatestCommit: latestCommit,
		CommitFound:  commitFound,
		FullTree:     GlobalFullTree,
	})
	if err != nil {
		log.Print("execute:", err)
	}
	treefile.Sync()
	defer treefile.Close()
}

// flattenTree flattens a tree structure with depth information
func flattenTree(items []TreeItem, depth int) []FlatTreeItem {
	var flat []FlatTreeItem
	for _, item := range items {
		flat = append(flat, FlatTreeItem{
			Name:   item.Name,
			Link:   item.Link,
			IsFile: item.IsFile,
			Depth:  depth,
		})
		if !item.IsFile && len(item.Children) > 0 {
			flat = append(flat, flattenTree(item.Children, depth+1)...)
		}
	}
	return flat
}

// buildFullTreeRecursive builds the complete repository tree structure
func buildFullTreeRecursive(repo *git.Repository, tree *git.Tree, path string) []TreeItem {
	var items []TreeItem
	count := int(tree.EntryCount())

	// Sort entries: directories first, then files
	var dirs []*git.TreeEntry
	var files []*git.TreeEntry

	for i := 0; i < count; i++ {
		entry := tree.EntryByIndex(uint64(i))
		if entry.Type == git.ObjectTree {
			dirs = append(dirs, entry)
		} else if entry.Type == git.ObjectBlob {
			files = append(files, entry)
		}
	}

	// Sort directories and files alphabetically (case-insensitive)
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	// Process directories first
	for _, entry := range dirs {
		nexttree, err := repo.LookupTree(entry.Id)
		if err != nil {
			log.Print("warning: failed to lookup tree:", err)
			continue
		}

		newpath := filepath.Join(path, entry.Name)
		children := buildFullTreeRecursive(repo, nexttree, newpath)

		link := newpath

		items = append(items, TreeItem{
			Name:     entry.Name,
			Link:     link,
			IsFile:   false,
			Path:     newpath,
			Children: children,
		})
	}

	// Process files
	for _, entry := range files {
		newpath := filepath.Join(path, entry.Name)
		link := newpath + ".html"

		items = append(items, TreeItem{
			Name:     entry.Name,
			Link:     link,
			IsFile:   true,
			Path:     newpath,
			Children: nil,
		})
	}

	return items
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

	// Build full tree structure once and store globally (flattened)
	treeItems := buildFullTreeRecursive(repo, tree, "/tree")
	GlobalFullTree = flattenTree(treeItems, 0)

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

// getBranchName returns the name of the current branch that HEAD points to
// Returns the shorthand branch name (e.g., "main" instead of "refs/heads/main")
// If HEAD is detached or there's an error, returns "HEAD"
func getBranchName(repo *git.Repository) string {
	head, err := repo.Head()
	if err != nil {
		return "HEAD"
	}
	defer head.Free()

	if head.IsBranch() {
		branchName, err := head.Branch().Name()
		if err != nil {
			return "HEAD"
		}
		return branchName
	}

	return "HEAD"
}

// getBranches returns a list of all branches in the repository
func getBranches(repo *git.Repository) []RefListElem {
	var branches []RefListElem

	iter, err := repo.NewBranchIterator(git.BranchAll)
	if err != nil {
		log.Print("failed to create branch iterator:", err)
		return branches
	}
	defer iter.Free()

	err = iter.ForEach(func(branch *git.Branch, branchType git.BranchType) error {
		name, err := branch.Name()
		if err != nil {
			return nil
		}

		ref, err := branch.Resolve()
		if err != nil {
			return nil
		}
		defer ref.Free()

		commitHash := ref.Target().String()
		logLink := "/log/" + name

		branches = append(branches, RefListElem{
			Name:       name,
			Type:       "branch",
			CommitHash: commitHash[:8],
			LogLink:    logLink,
		})

		return nil
	})

	if err != nil {
		log.Print("error iterating branches:", err)
	}

	return branches
}

// getTags returns a list of all tags in the repository
func getTags(repo *git.Repository) []RefListElem {
	var tags []RefListElem

	err := repo.Tags.Foreach(func(name string, oid *git.Oid) error {
		// Remove refs/tags/ prefix
		tagName := name
		if len(name) > 10 && name[:10] == "refs/tags/" {
			tagName = name[10:]
		}

		// Try to peel the tag to get the commit it points to
		obj, err := repo.Lookup(oid)
		if err != nil {
			return nil
		}
		defer obj.Free()

		var commitHash string
		if obj.Type() == git.ObjectTag {
			tag, err := obj.AsTag()
			if err != nil {
				return nil
			}
			target := tag.Target()
			if target != nil {
				commitHash = target.Id().String()
				target.Free()
			}
		} else if obj.Type() == git.ObjectCommit {
			commitHash = oid.String()
		}

		if commitHash != "" {
			tags = append(tags, RefListElem{
				Name:       tagName,
				Type:       "tag",
				CommitHash: commitHash[:8],
				LogLink:    "", // Tags don't have log links
			})
		}

		return nil
	})

	if err != nil {
		log.Print("error iterating tags:", err)
	}

	return tags
}

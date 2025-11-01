package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	git "github.com/libgit2/git2go/v34"
)

// run is the core logic of gitgo, extracted for testability.
// It generates static HTML pages for a git repository.
func run(repoPath, destDir, installDir string, force bool) error {
	imageloc := filepath.Join(installDir, "logo.png")

	_, err := os.Stat(imageloc)
	if err == nil {
		// logo found
		GlobalDataGlobal.LogoFound = true
		err := os.Symlink(imageloc, filepath.Join(destDir, "logo.png"))
		if err != nil && !os.IsExist(err) {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	repo, err := git.OpenRepositoryExtended(repoPath, git.RepositoryOpenNoSearch, "")
	if err != nil {
		return err
	}

	obj, _, err := repo.RevparseExt("HEAD")
	if err != nil {
		return err
	}

	head := obj.Id()

	// Get the repo name using the helper function
	repoName, err := getRepoName(repoPath)
	if err != nil {
		return err
	}
	Config.RepoName = repoName

	// Get the current branch name and update log link
	branchName := getBranchName(repo)
	GlobalDataGlobal.BranchName = branchName
	// Update the log link to be branch-specific
	for i, link := range GlobalDataGlobal.Links {
		if link.Pretty == "log" {
			GlobalDataGlobal.Links[i].Link = "/log/" + branchName
			break
		}
	}

	// Update destination directory to include repo name
	destDir = filepath.Join(destDir, repoName)
	Config.DestDir = destDir

	// validate that destination directory doesn't exist or is empty
	err = validateDestDir(destDir, force)
	if err != nil {
		return err
	}

	err = makeDir(destDir)
	if err != nil {
		return err
	}

	templ = template.New("").Funcs(funcmap)

	t, err = templ.ParseGlob(filepath.Join(installDir, "templates/*.html"))

	if err != nil {
		return err
	}

	var (
		readmefiles = [...]string{"HEAD:README", "HEAD:README.md"}
		readmefile  FileViewRenderData
		readmefound = false

		licensefiles = [...]string{"HEAD:LICENSE", "HEAD:COPYING", "HEAD:LICENSE.md"}
		licensefile  FileViewRenderData
		licensefound = false

		latestCommit CommitListElem
		commitfound  = false
	)

	for _, file := range readmefiles {
		fileobj, _, err := repo.RevparseExt(file)
		if err == nil && fileobj.Type() == git.ObjectBlob {
			blob, err := fileobj.AsBlob()

			if err != nil {
				return err
			}

			lines := contentsToLines(blob.Contents(), int(blob.Size()))

			readmefile.Name = strings.TrimPrefix(file, "HEAD:")
			readmefile.Lines = lines
			readmefound = true
			break
		}
	}

	for _, file := range licensefiles {
		fileobj, _, err := repo.RevparseExt(file)
		if err == nil && fileobj.Type() == git.ObjectBlob {
			blob, err := fileobj.AsBlob()

			if err != nil {
				return err
			}

			lines := contentsToLines(blob.Contents(), int(blob.Size()))

			licensefile.Name = strings.TrimPrefix(file, "HEAD:")
			licensefile.Lines = lines
			licensefound = true
			break
		}
	}

	// Create directories first
	err = makeDir(filepath.Join(destDir, "commit"))
	if err != nil {
		return err
	}
	err = makeDir(filepath.Join(destDir, "tree"))
	if err != nil {
		return err
	}
	err = makeDir(filepath.Join(destDir, "log"))
	if err != nil {
		return err
	}
	// Create branch-specific log directory
	err = makeDir(filepath.Join(destDir, "log", branchName))
	if err != nil {
		return err
	}

	// Get commit list for commit count and latest commit
	commitlist := getCommitLog(repo, head)
	GlobalDataGlobal.CommitCount = len(commitlist)

	// Get latest commit
	headRef, headErr := repo.Head()
	if headErr == nil {
		commit, commitErr := repo.LookupCommit(headRef.Target())
		headRef.Free()
		if commitErr == nil {
			latestCommit.Link = "/commit/" + commit.TreeId().String() + ".html"
			latestCommit.Msg = commit.Summary()
			latestCommit.Name = commit.Author().Name
			latestCommit.Date = commit.Author().When
			latestCommit.AbbrevHash = commit.TreeId().String()[:8]
			commitfound = true
			commit.Free()
		}
	}

	// Get root tree file list for index page
	var rootTree []FileListElem
	treefound := false
	commit, err := repo.LookupCommit(head)
	if err == nil {
		tree, treeErr := commit.Tree()
		if treeErr == nil {
			rootTree = getRootTreeFileList(repo, tree)
			treefound = true
		}
	}

	// Get contributors
	contributors := getContributors(repo, head)

	indexfile, err := os.Create(filepath.Join(destDir, "index.html"))
	if err != nil {
		return err
	}
	err = t.ExecuteTemplate(indexfile, "index.html", IndexRenderData{
		GlobalData:     &GlobalDataGlobal,
		ReadmeFile:     readmefile,
		ReadmeFound:    readmefound,
		LicenseFile:    licensefile,
		LicenseFound:   licensefound,
		LatestCommit:   latestCommit,
		CommitFound:    commitfound,
		RootTree:       rootTree,
		TreeFound:      treefound,
		Contributors:   contributors,
		ContributorsCt: len(contributors),
	})
	if err != nil {
		return err
	}
	indexfile.Sync()
	defer indexfile.Close()

	logfile, err := os.Create(filepath.Join(destDir, "log", branchName, "index.html"))
	if err != nil {
		return err
	}
	err = t.ExecuteTemplate(logfile, "log.html", LogRenderData{GlobalData: &GlobalDataGlobal, Commits: commitlist})
	if err != nil {
		return err
	}
	logfile.Sync()
	defer logfile.Close()

	indexTree(repo, head)

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.StringVar(&Config.DestDir, "destdir", "build", "target directory")
	flag.StringVar(&Config.InstallDir, "installdir", ".", "install directory containing templates")
	flag.BoolVar(&Config.Force, "force", false, "force overwrite by clearing destination directory if not empty")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: gitgo [options] <git repo>\n")
		flag.VisitAll(func(f *flag.Flag) {
			// Determine the flag type
			flagType := "string"
			if _, ok := f.Value.(interface{ IsBoolFlag() bool }); ok {
				flagType = ""
			}

			if flagType != "" {
				fmt.Fprintf(flag.CommandLine.Output(), "  --%s %s\n", f.Name, flagType)
			} else {
				fmt.Fprintf(flag.CommandLine.Output(), "  --%s\n", f.Name)
			}
			fmt.Fprintf(flag.CommandLine.Output(), "    	%s\n", f.Usage)
			if f.DefValue != "" {
				fmt.Fprintf(flag.CommandLine.Output(), "    	(default %q)\n", f.DefValue)
			}
		})
	}

	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		flag.Usage()
		return
	}

	err := run(args[0], Config.DestDir, Config.InstallDir, Config.Force)
	if err != nil {
		log.Fatal(err)
	}
}

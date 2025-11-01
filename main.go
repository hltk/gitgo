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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.StringVar(&Config.DestDir, "destdir", "build", "target directory")
	flag.StringVar(&Config.InstallDir, "installdir", ".", "install directory containing templates")
	flag.BoolVar(&Config.Force, "force", false, "force overwrite by clearing destination directory if not empty")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: gitgo [options] <git repo>\n")
		flag.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(flag.CommandLine.Output(), "  --%s string\n", f.Name)
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

	imageloc := filepath.Join(Config.InstallDir, "logo.png")

	_, err := os.Stat(imageloc)
	if err == nil {
		// logo found
		GlobalDataGlobal.LogoFound = true
		err := os.Symlink(imageloc, filepath.Join(Config.DestDir, "logo.png"))
		if err != nil && !os.IsExist(err) {
			log.Fatal(err)
		}
	} else if !os.IsNotExist(err) {
		log.Fatal(err)
	}

	repo, err := git.OpenRepositoryExtended(args[0], git.RepositoryOpenNoSearch, "")
	if err != nil {
		log.Fatal(err)
	}

	obj, _, err := repo.RevparseExt("HEAD")
	if err != nil {
		log.Fatal(err)
	}

	head := obj.Id()

	// remove path and .git suffix from the repo's name
	Config.RepoName = strings.TrimSuffix(filepath.Base(args[0]), ".git")

	Config.DestDir = filepath.Join(Config.DestDir, Config.RepoName)

	// validate that destination directory doesn't exist or is empty
	err = validateDestDir(Config.DestDir, Config.Force)
	if err != nil {
		log.Fatal(err)
	}

	err = makedir(Config.DestDir)
	if err != nil {
		log.Fatal(err)
	}

	templ = template.New("").Funcs(funcmap)

	t, err = templ.ParseGlob(filepath.Join(Config.InstallDir, "templates/*.html"))

	if err != nil {
		log.Print("parse:", err)
	}

	var (
		readmefiles = [...]string{"HEAD:README", "HEAD:README.md"}
		readmefile  FileViewRenderData
		readmefound = false

		licensefiles = [...]string{"HEAD:LICENSE", "HEAD:COPYING", "HEAD:LICENSE.md"}
		licensefile  FileViewRenderData
		licensefound = false
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

	for _, file := range licensefiles {
		fileobj, _, err := repo.RevparseExt(file)
		if err == nil && fileobj.Type() == git.ObjectBlob {
			blob, err := fileobj.AsBlob()

			if err != nil {
				log.Fatal(err)
			}

			lines := contentstolines(blob.Contents(), int(blob.Size()))

			licensefile.Name = strings.TrimPrefix(file, "HEAD:")
			licensefile.Lines = lines
			licensefound = true
			break
		}
	}

	indexfile, err := os.Create(filepath.Join(Config.DestDir, "index.html"))
	if err != nil {
		log.Fatal(err)
	}
	err = t.ExecuteTemplate(indexfile, "index.html", IndexRenderData{&GlobalDataGlobal, readmefile, readmefound, licensefile, licensefound})
	if err != nil {
		log.Print("execute:", err)
	}
	indexfile.Sync()
	defer indexfile.Close()

	err = makedir(filepath.Join(Config.DestDir, "commit"))
	if err != nil {
		log.Fatal(err)
	}
	err = makedir(filepath.Join(Config.DestDir, "tree"))
	if err != nil {
		log.Fatal(err)
	}
	err = makedir(filepath.Join(Config.DestDir, "log"))
	if err != nil {
		log.Fatal(err)
	}

	commitlist := getcommitlog(repo, head)

	logfile, err := os.Create(filepath.Join(Config.DestDir, "log/index.html"))
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

}

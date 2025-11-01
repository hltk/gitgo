# Gitgo

Go program for creating static web pages of git repositories.

This project started as a fork of [stagit](https://git.codemadness.org/stagit/). The basic structure of the code and the interaction with the libgit2 api are from stagit.

The goal of the project is not to create the best possible git web portal, instead gitgo is intended to be a learning exercise of git and golang.

## Requirements

- **Go**: You must have Go installed locally on your system
- **libgit2**: You must have libgit2 installed locally on your system
- Go package: `github.com/libgit2/git2go`

## Usage

1. Obtain a copy of this repo: `git clone ..`
2. Build the program: `make`
3. Run `./gitgo`. When running the program you have to give it two flags:
   - `-destdir` indicates where the static pages are going to be stored
   - `-installdir` indicates where the git repo is stored in the file system

4. (NOT YET IMPLEMENTED) There is an optional extra step: install gitgo for all users with the command `make install`

If there is a `logo.png` file in the installation directory, the program will detect it, and add it to every page.

## License

Gitgo retains the MIT/X Consortium License of stagit.

## Authors

- Henrik Aalto <hltk@hltk.fi>

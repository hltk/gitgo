# Gitgo

Go program for creating static web pages of git repositories.

This project started as a fork of [stagit](https://git.codemadness.org/stagit/). The basic structure of the code and the interaction with the libgit2 api are from stagit.

The goal of the project is not to create the best possible git web portal, instead gitgo is intended to be a learning exercise of git and golang.

## Requirements

- **Go**: You must have Go installed locally on your system (version 1.23 or later)
- **libgit2**: You must have libgit2 v1.5.x installed locally on your system
  - Note: git2go v34 specifically requires libgit2 v1.5.x
  - If you have a different version, you may need to install libgit2 v1.5.x or downgrade:
    ```bash
    # On macOS with Homebrew
    brew uninstall libgit2
    brew install libgit2@1.5
    ```
- Go package: `github.com/libgit2/git2go/v34`

## Usage

1. Obtain a copy of this repo: `git clone https://github.com/hltk/gitgo.git`
2. Build the program:
   ```bash
   # If libgit2 v1.5.x is installed system-wide
   make
   
   # If you have a custom libgit2 build (e.g., in /path/to/libgit2)
   PKG_CONFIG_PATH=/path/to/libgit2/build LIBGIT2_PATH=/path/to/libgit2 make
   ```
3. Run `./gitgo`. When running the program you have to give it two flags:
   - `-destdir` indicates where the static pages are going to be stored
   - `-installdir` indicates where the git repo is stored in the file system

4. (NOT YET IMPLEMENTED) There is an optional extra step: install gitgo for all users with the command `make install`

If there is a `logo.png` file in the installation directory, the program will detect it, and add it to every page.

## License

Gitgo retains the MIT/X Consortium License of stagit.

## Authors

- Henrik Aalto

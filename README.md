# Gitgo

Go program for creating static web pages of git repositories.

This project is inspired by [stagit](https://git.codemadness.org/stagit/) (written in C). The basic structure and approach to interacting with the libgit2 API are based on stagit's design.

This project is primarily a learning exercise for exploring Git and Go, rather than aiming to be a production-ready git web portal.

## Requirements

- **Go**: You must have Go installed locally on your system (version 1.23 or later)
- **libgit2**: You must have libgit2 v1.5.x installed locally on your system
  - Note: git2go v34 specifically requires libgit2 v1.5.x
  - Note: [git2go](https://github.com/libgit2/git2go) is no longer actively maintained, so you may need to build libgit2 v1.5.0 from source
  - If you have a different version, you may need to install libgit2 v1.5.x or downgrade:
    ```bash
    # On macOS with Homebrew
    brew uninstall libgit2
    brew install libgit2@1.5
    ```
- Go package: `github.com/libgit2/git2go/v34`

### Building libgit2 v1.5.0 from Source

If the Homebrew version doesn't work (since git2go is no longer maintained), you can build libgit2 v1.5.0 manually:

```bash
# 1. Clone libgit2
git clone https://github.com/libgit2/libgit2.git
cd libgit2

# 2. Checkout v1.5.0
git checkout v1.5.0

# 3. Build libgit2
mkdir build
cd build
cmake ..
cmake --build .

# 4. Note the path to the build directory (e.g., /Users/username/personal/libgit2/build)
```

Then when building gitgo, use the `PKG_CONFIG_PATH` to point to your libgit2 build:

```bash
PKG_CONFIG_PATH=/path/to/libgit2/build make
```

For example:

```bash
PKG_CONFIG_PATH=/Users/henrik/personal/libgit2/build make
```

## Usage

1. Obtain a copy of this repo: `git clone https://github.com/hltk/gitgo.git`
2. Build the program:

   ```bash
   # If libgit2 v1.5.x is installed system-wide
   make

   # If you have a custom libgit2 build (e.g., in /path/to/libgit2)
   PKG_CONFIG_PATH=/path/to/libgit2/build LIBGIT2_PATH=/path/to/libgit2 make
   ```

3. Run `./gitgo` with a git repository path. The program accepts the following optional flags:
   - `--destdir`: Directory where static pages will be stored (default: `build`)
   - `--installdir`: Directory containing the `templates/` folder (default: current directory)

### Examples

Basic usage with defaults (outputs to `build/` directory):

```bash
./gitgo ../rustgrad
./gitgo .
```

Custom output directory:

```bash
./gitgo --destdir output ../rustgrad
./gitgo --destdir /path/to/output ../rustgrad
```

Custom installation directory (if templates are installed elsewhere):

```bash
./gitgo --installdir /usr/share/gitgo ../rustgrad
```

4. There is an optional extra step: install gitgo for all users with the command `make install`

If there is a `logo.png` file in the installation directory, the program will detect it, and add it to every page.

### Preview Generated Pages

To preview the generated static pages locally:

```bash
make serve
```

This will start a web server on http://localhost:8000 serving the `build/` directory. Open that URL in your browser to view the generated pages.

## Testing

Gitgo includes comprehensive test coverage for all major components. The test suite is organized into four main test files:

- `util_test.go` - Tests for utility functions
- `git_test.go` - Tests for Git operations
- `main_test.go` - Integration tests for the main application logic
- `cmd/serve/server_test.go` - Tests for the HTTP server functionality

### Running Tests

Since gitgo depends on libgit2, you need to set the appropriate environment variables when running tests:

```bash
PKG_CONFIG_PATH=/path/to/libgit2/build \
CGO_CFLAGS="-I/path/to/libgit2/include" \
CGO_LDFLAGS="-L/path/to/libgit2/build -Wl,-rpath,/path/to/libgit2/build" \
go test ./...
```

Run with verbose output:

```bash
PKG_CONFIG_PATH=/path/to/libgit2/build \
CGO_CFLAGS="-I/path/to/libgit2/include" \
CGO_LDFLAGS="-L/path/to/libgit2/build -Wl,-rpath,/path/to/libgit2/build" \
go test -v ./...
```

The server tests can be run independently without libgit2:

```bash
go test -v ./cmd/serve/
```

## License

Gitgo retains the MIT/X Consortium License of stagit.

## Authors

- Henrik Aalto

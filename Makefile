GO ?= $(shell which go || echo /usr/local/go/bin/go)

all: gitgo

gitgo: go.mod *.go
	$(GO) build -o gitgo

clean:
	rm -f gitgo

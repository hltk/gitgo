GO ?= $(shell which go || echo /usr/local/go/bin/go)
PKG_CONFIG_PATH ?=
LIBGIT2_PATH ?=

all: gitgo

gitgo: go.mod *.go
ifeq ($(LIBGIT2_PATH),)
	PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) $(GO) build -o gitgo
else
	PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) \
	CGO_CFLAGS="-I$(LIBGIT2_PATH)/include" \
	CGO_LDFLAGS="-L$(LIBGIT2_PATH)/build -lgit2" \
	$(GO) build -o gitgo
endif

clean:
	rm -f gitgo

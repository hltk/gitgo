GO ?= $(shell which go || echo /usr/local/go/bin/go)
PKG_CONFIG_PATH ?=
LIBGIT2_PATH ?=

# Infer LIBGIT2_PATH from PKG_CONFIG_PATH if PKG_CONFIG_PATH ends with /build
ifneq ($(PKG_CONFIG_PATH),)
ifeq ($(LIBGIT2_PATH),)
LIBGIT2_PATH_CANDIDATE := $(patsubst %/build,%,$(PKG_CONFIG_PATH))
ifneq ($(LIBGIT2_PATH_CANDIDATE),$(PKG_CONFIG_PATH))
LIBGIT2_PATH := $(LIBGIT2_PATH_CANDIDATE)
endif
endif
endif

all: gitgo

gitgo: go.mod *.go
ifneq ($(LIBGIT2_PATH),)
	PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) \
	CGO_CFLAGS="-I$(LIBGIT2_PATH)/include" \
	CGO_LDFLAGS="-L$(LIBGIT2_PATH)/build -Wl,-rpath,$(LIBGIT2_PATH)/build -lgit2" \
	$(GO) build -o gitgo
else
	PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) \
	$(GO) build -o gitgo
endif

clean:
	rm -f gitgo

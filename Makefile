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

.PHONY: all clean serve css format format-html

all: css gitgo

css: $(wildcard templates/css/*.css)
	cat $(sort $(wildcard templates/css/*.css)) > templates/styles.css

gitgo: go.mod main.go config.go types.go git.go util.go cmd/serve/server.go
ifneq ($(LIBGIT2_PATH),)
	PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) \
	CGO_CFLAGS="-I$(LIBGIT2_PATH)/include" \
	CGO_LDFLAGS="-L$(LIBGIT2_PATH)/build -Wl,-rpath,$(LIBGIT2_PATH)/build" \
	$(GO) build -o gitgo main.go config.go types.go git.go util.go
else
	PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) \
	$(GO) build -o gitgo main.go config.go types.go git.go util.go
endif

serve:
	$(GO) run cmd/serve/server.go

clean:
	rm -f gitgo templates/styles.css

format-html:
	uv run djlint templates/ --profile=golang --reformat

format: format-html

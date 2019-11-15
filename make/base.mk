SHELL := /usr/bin/env bash -o pipefail

ifndef PROJECT
$(error PROJECT is not set)
endif
ifndef GO_MODULE
$(error GO_MODULE is not set)
endif

UNAME_OS := $(shell uname -s)
UNAME_ARCH := $(shell uname -m)

ENV_DIR := .env
ENV_SH := $(ENV_DIR)/env.sh
ENV_BACKUP_DIR := $(HOME)/.config/$(PROJECT)/env

ifndef CACHE_BASE
CACHE_BASE := $(HOME)/.cache/$(PROJECT)
endif
CACHE := $(CACHE_BASE)/$(UNAME_OS)/$(UNAME_ARCH)
CACHE_BIN := $(CACHE)/bin
CACHE_INCLUDE := $(CACHE)/include
CACHE_VERSIONS := $(CACHE)/versions
CACHE_ENV := $(CACHE)/env
CACHE_GO := $(CACHE)/go

TMP := .tmp

export GO111MODULE := on
ifdef GOPRIVATE
export GOPRIVATE := $(GOPRIVATE),$(GO_MODULE)
else
export GOPRIVATE := $(GO_MODULE)
endif
export GOPATH := $(abspath $(CACHE_GO))
export GOBIN := $(abspath $(CACHE_BIN))
export PATH := $(GOBIN):$(PATH)

.PHONY: envbackup
envbackup:
	rm -rf "$(ENV_BACKUP_DIR)"
	mkdir -p "$(dir $(ENV_BACKUP_DIR))"
	cp -R "$(ENV_DIR)" "$(ENV_BACKUP_DIR)"

.PHONY: envrestore
envrestore:
	@ if [ ! -d "$(ENV_BACKUP_DIR)" ]; then echo "no backup stored in $(ENV_BACKUP_DIR)"; exit 1; fi
	rm -rf "$(ENV_DIR)"
	cp -R "$(ENV_BACKUP_DIR)" "$(ENV_DIR)"

.PHONY: direnv
direnv:
	@mkdir -p $(CACHE_ENV)
	@rm -f $(CACHE_ENV)/env.sh
	@echo 'export CACHE="$(abspath $(CACHE))"' >> $(CACHE_ENV)/env.sh
	@echo 'export GO111MODULE="$(GO111MODULE)"' >> $(CACHE_ENV)/env.sh
	@echo 'export GOPRIVATE="$(GOPRIVATE)"' >> $(CACHE_ENV)/env.sh
	@echo 'export GOPATH="$(GOPATH)"' >> $(CACHE_ENV)/env.sh
	@echo 'export GOBIN="$(GOBIN)"' >> $(CACHE_ENV)/env.sh
	@echo 'export PATH="$(GOBIN):$${PATH}"' >> $(CACHE_ENV)/env.sh
	@echo '[ -f "$(abspath $(ENV_SH))" ] && . "$(abspath $(ENV_SH))"' >> $(CACHE_ENV)/env.sh
	@echo $(CACHE_ENV)/env.sh

.PHONY: clean
clean:
	git clean -xdf -e /$(ENV_DIR)/

.PHONY: cleancache
cleancache:
	rm -rf $(CACHE_BASE)

.PHONY: nuke
nuke: clean cleancache
	sudo rm -rf $(CACHE_GO)/pkg/mod

.PHONY: dockerdeps
dockerdeps::

.PHONY: deps
deps:: dockerdeps

.PHONY: pregenerate
pregenerate::

.PHONY: postgenerate
postgenerate::

.PHONY: generate
generate:
	@$(MAKE) pregenerate
	@$(MAKE) postgenerate

.PHONY: checknodiffgenerated
checknodiffgenerated:
	@ if [ -d .git ]; then \
			$(MAKE) checknodiffgeneratedinternal; \
		else \
			echo "skipping make checknodiffgenerated due to no .git repository" >&2; \
		fi

.PHONY: checknodiffgeneratedinternal
checknodiffgeneratedinternal:
	bash make/scripts/checknodiffgenerated.bash $(MAKE) generate

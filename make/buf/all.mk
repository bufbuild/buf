GO_ALL_REPO_PKGS := ./cmd/... ./private/...
#GO_GET_PKGS := $(GO_GET_PKGS)
GO_BINS := $(GO_BINS) \
	cmd/buf \
	cmd/protoc-gen-buf-breaking \
	cmd/protoc-gen-buf-lint \
	private/buf/bufwkt/cmd/wkt-go-data \
	private/bufpkg/bufmodule/bufmoduleapi/cmd/buf-legacyfederation-go-data \
	private/bufpkg/bufmodule/bufmoduletesting/cmd/buf-digest \
	private/bufpkg/bufmodule/bufmoduletesting/cmd/buf-new-commit-id \
	private/bufpkg/bufstyle/cmd/bufstyle \
	private/pkg/bandeps/cmd/bandeps \
	private/pkg/git/cmd/git-ls-files-unstaged \
	private/pkg/storage/cmd/ddiff \
	private/pkg/storage/cmd/storage-go-data \
	private/pkg/licenseheader/cmd/license-header
GO_TEST_BINS := $(GO_TEST_BINS) \
	cmd/buf/internal/command/alpha/protoc/internal/protoc-gen-insertion-point-receiver \
	cmd/buf/internal/command/alpha/protoc/internal/protoc-gen-insertion-point-writer \
	cmd/buf/internal/command/generate/internal/protoc-gen-top-level-type-names-yaml \
	private/bufpkg/bufcheck/internal/cmd/buf-plugin-panic \
	private/bufpkg/bufcheck/internal/cmd/buf-plugin-suffix \
	private/bufpkg/bufcheck/internal/cmd/buf-plugin-protovalidate-ext \
	private/bufpkg/bufcheck/internal/cmd/buf-plugin-rpc-ext \
	private/bufpkg/bufcheck/internal/cmd/buf-plugin-duplicate-category \
	private/bufpkg/bufcheck/internal/cmd/buf-plugin-duplicate-rule
GO_TEST_WASM_BINS := $(GO_TEST_WASM_BINS) \
	private/bufpkg/bufcheck/internal/cmd/buf-plugin-suffix
GO_MOD_VERSION := 1.23
DOCKER_BINS := $(DOCKER_BINS) buf
FILE_IGNORES := $(FILE_IGNORES) \
	.build/ \
	.ctrlp \
	.idea/ \
	.vscode/ \
	cmd/buf/internal/command/alpha/protoc/test.txt \
	private/bufpkg/buftesting/cache/ \
	private/buf/buftesting/cache/ \
	private/pkg/storage/storageos/tmp/ \
	private/buf/cmd/buf/testdata/imports/cache/v3/modulelocks/ \
	private/buf/cmd/buf/testdata/imports/corrupted_cache_dep/v3/modulelocks/ \
	private/buf/cmd/buf/testdata/imports/corrupted_cache_file/v3/modulelocks/
LICENSE_HEADER_LICENSE_TYPE := apache
LICENSE_HEADER_COPYRIGHT_HOLDER := Buf Technologies, Inc.
LICENSE_HEADER_YEAR_RANGE := 2020-2025
LICENSE_HEADER_IGNORES := \/testdata enterprise
BANDEPS_CONFIG := etc/bandeps/bandeps.yaml
BUFPRIVATEUSAGE_PKGS := ./private/...
PROTOVALIDATE_VERSION := v1.0.0
# Comment out to use released buf
#BUF_GO_INSTALL_PATH := ./cmd/buf

BUF_LINT_INPUT := .
BUF_BREAKING_INPUT := .
BUF_BREAKING_AGAINST_INPUT ?= .git\#branch=main
BUF_FORMAT_INPUT := .

DISALLOW_NOLINT := true

LEGACY_FEDERATION_FILE_PATH ?=

include make/go/bootstrap.mk
include make/go/dep_buf.mk
include make/go/dep_minisign.mk
include make/go/dep_protoc.mk
include make/go/dep_protoc_gen_go.mk
include make/go/dep_protoc_gen_connect_go.mk
include make/go/go.mk
include make/go/docker.mk
include make/go/license_header.mk
include make/go/bandeps.mk
include make/go/bufprivateusage.mk
include make/go/buf.mk

installtest:: $(PROTOC) $(PROTOC_GEN_GO)

.PHONY: bufstyle
bufstyle: installbufstyle
	@echo bufstyle NON_GEN_GOPKGS
	@bufstyle $(shell go list $(GOPKGS) | grep -v \/gen\/)

postlint:: bufstyle

.PHONY: godata
godata: installwkt-go-data installbuf-legacyfederation-go-data $(PROTOC)
	rm -rf private/gen/data/datawkt
	mkdir -p private/gen/data/datawkt
	wkt-go-data "$(CACHE_INCLUDE)" --package datawkt --protobuf-version "$(PROTOC_VERSION)" > private/gen/data/datawkt/datawkt.gen.go
ifdef LEGACY_FEDERATION_FILE_PATH
	rm -rf private/gen/data/datalegacyfederation
	mkdir -p private/gen/data/datalegacyfederation
	cat "$(LEGACY_FEDERATION_FILE_PATH)" | buf-legacyfederation-go-data --package datalegacyfederation > private/gen/data/datalegacyfederation/datalegacyfederation.gen.go
endif

prepostgenerate:: godata

.PHONY: bufworkspacebuflocks
bufworkspacebuflocks: installbuf-digest installbuf-new-commit-id
	bash private/buf/bufworkspace/testdata/basic/scripts/fakebuflock.bash

prepostgenerate:: bufworkspacebuflocks

bufgeneratedeps:: $(PROTOC_GEN_GO) $(PROTOC_GEN_CONNECT_GO)

.PHONY: bufgeneratecleango
bufgeneratecleango:
	rm -rf private/gen/proto

.PHONY: bufgeneratecleantestdata
bufgeneratecleantestdata:
	rm -rf cmd/buf/testdata/check_plugins/current/vendor/protovalidate
	rm -rf cmd/buf/testdata/check_plugins/previous/vendor/protovalidate
	rm -rf private/bufpkg/bufcheck/testdata/lint/protovalidate/vendor/protovalidate
	rm -rf private/bufpkg/bufcheck/testdata/lint/protovalidate_predefines/vendor/protovalidate

bufgenerateclean:: \
	bufgeneratecleango \
	bufgeneratecleantestdata

.PHONY: bufgeneratego
bufgeneratego:
	$(BUF_BIN) generate --template etc/template/buf.go.gen.yaml
	$(BUF_BIN) generate --template etc/template/buf.go-client.gen.yaml

.PHONY: bufgeneratetestdata
bufgeneratetestdata:
	$(BUF_BIN) export \
		buf.build/bufbuild/protovalidate:$(PROTOVALIDATE_VERSION) \
		--output cmd/buf/testdata/check_plugins/current/vendor/protovalidate
	$(BUF_BIN) export \
		buf.build/bufbuild/protovalidate:$(PROTOVALIDATE_VERSION) \
		--output cmd/buf/testdata/check_plugins/previous/vendor/protovalidate
	$(BUF_BIN) export \
		buf.build/bufbuild/protovalidate:$(PROTOVALIDATE_VERSION) \
		--output private/bufpkg/bufcheck/testdata/lint/protovalidate/vendor/protovalidate
	$(BUF_BIN) export \
		buf.build/bufbuild/protovalidate:$(PROTOVALIDATE_VERSION) \
		--output private/bufpkg/bufcheck/testdata/lint/protovalidate_predefined/vendor/protovalidate

bufgeneratesteps:: \
	bufgeneratego \
	bufgeneratetestdata

.PHONY: bufrelease
bufrelease: $(MINISIGN)
	bash make/buf/scripts/release.bash

.PHONY: bufbinarysize
bufbinarysize:
	@bash make/buf/scripts/binarysize.bash ./cmd/buf

.PHONY: updateversion
updateversion:
ifndef VERSION
	$(error "VERSION must be set")
endif
	$(SED_I) "s/Version.*=.*\"[0-9]\.[0-9][0-9]*\.[0-9][0-9]*.*\"/Version = \"$(VERSION)\"/g" private/buf/bufcli/bufcli.go
	gofmt -s -w private/buf/bufcli/bufcli.go

.PHONY: releasechangelog
releasechangelog:
ifndef VERSION
	$(error "VERSION must be set")
endif
	$(SED_I) 's/## \[Unreleased\]/## \[v$(VERSION)\] - $(shell date '+%Y-%m-%d')/' CHANGELOG.md
	$(SED_I) -E '/^\[Unreleased\]: .*HEAD$$/s/(Unreleased|HEAD)/v$(VERSION)/g' CHANGELOG.md

.PHONY: updategoversion
updategoversion: installgit-ls-files-unstaged
ifndef GOVERSION
	$(error "GOVERSION must be set")
endif
	# make sure both of these docker images exist
	# the release of these images will lag the actual release
	docker pull golang:$(GOVERSION)-bookworm
	docker pull golang:$(GOVERSION)-alpine3.22
	$(SED_I) "s/golang:1\.[0-9][0-9]*/golang:$(GOVERSION)/g" $(shell git-ls-files-unstaged | grep Dockerfile)
	$(SED_I) "s/golang:1\.[0-9][0-9]*/golang:$(GOVERSION)/g" $(shell git-ls-files-unstaged | grep \.mk$)
	$(SED_I) "s/go-version: '1\.[0-9][0-9].x'/go-version: '$(GOVERSION).x'/g" $(shell git-ls-files-unstaged | grep \.github\/workflows | grep -v previous.yaml)
	$(MAKE) checkandupdateprecommithooks

.PHONY: bufimageutilupdateexpectations
bufimageutilupdateexpectations:
	# You may need to run this after updating protoc versions
	BUFBUILD_BUF_BUFIMAGEUTIL_SHOULD_UPDATE_EXPECTATIONS=1 go test -parallel 1 ./private/bufpkg/bufimage/bufimageutil

.PHONY: checkandupdateprecommithooks
checkandupdateprecommithooks:
	@bash make/buf/scripts/checkandupdateprecommithooks.bash

postupgrade:: checkandupdateprecommithooks

GO_ALL_REPO_PKGS := ./cmd/... ./private/...
GO_GET_PKGS := $(GO_GET_PKGS) \
	github.com/bufbuild/protocompile@146b831231f7f7c1a19b09065875b9778d3d5d25
GO_BINS := $(GO_BINS) \
	cmd/buf \
	cmd/protoc-gen-buf-breaking \
	cmd/protoc-gen-buf-lint \
	private/buf/cmd/buf-digest \
	private/bufpkg/bufstyle/cmd/bufstyle \
	private/bufpkg/bufwkt/cmd/wkt-go-data \
	private/pkg/bandeps/cmd/bandeps \
	private/pkg/git/cmd/git-ls-files-unstaged \
	private/pkg/storage/cmd/ddiff \
	private/pkg/storage/cmd/storage-go-data \
	private/pkg/licenseheader/cmd/license-header \
	private/pkg/spdx/cmd/spdx-go-data
GO_TEST_BINS := $(GO_TEST_BINS) \
	private/buf/cmd/buf/command/alpha/protoc/internal/protoc-gen-insertion-point-receiver \
	private/buf/cmd/buf/command/alpha/protoc/internal/protoc-gen-insertion-point-writer
DOCKER_BINS := $(DOCKER_BINS) buf
FILE_IGNORES := $(FILE_IGNORES) \
	.build/ \
	.ctrlp \
	.idea/ \
	.vscode/ \
	private/buf/cmd/buf/command/alpha/protoc/test.txt \
	private/buf/cmd/buf/workspacetests/other/proto/workspacetest/cache/ \
	private/bufpkg/buftesting/cache/ \
	private/pkg/storage/storageos/tmp/
LICENSE_HEADER_LICENSE_TYPE := apache
LICENSE_HEADER_COPYRIGHT_HOLDER := Buf Technologies, Inc.
LICENSE_HEADER_YEAR_RANGE := 2020-2023
LICENSE_HEADER_IGNORES := \/testdata enterprise
PROTOVALIDATE_VERSION := v0.5.1
# Comment out to use released buf
BUF_GO_INSTALL_PATH := ./cmd/buf

BUF_LINT_INPUT := .
BUF_BREAKING_INPUT := .
BUF_BREAKING_AGAINST_INPUT ?= .git\#branch=main
BUF_FORMAT_INPUT := .

include make/go/bootstrap.mk
include make/go/dep_buf.mk
include make/go/dep_minisign.mk
include make/go/dep_protoc.mk
include make/go/dep_protoc_gen_go.mk
include make/go/dep_protoc_gen_connect_go.mk
include make/go/go.mk
include make/go/docker.mk
include make/go/buf.mk

installtest:: $(PROTOC) $(PROTOC_GEN_GO)

.PHONY: bufstyle
bufstyle: installbufstyle
	@echo bufstyle NON_GEN_GOPKGS
	@bufstyle $(shell go list $(GOPKGS) | grep -v \/gen\/)

postlint:: bufstyle

.PHONY: bandeps
bandeps: installbandeps
	bandeps -f data/bandeps/bandeps.yaml

postlonglint:: bandeps

.PHONY: godata
godata: installspdx-go-data installwkt-go-data $(PROTOC)
	rm -rf private/gen/data
	mkdir -p private/gen/data/datawkt
	mkdir -p private/gen/data/dataspdx
	wkt-go-data $(CACHE_INCLUDE) --package datawkt > private/gen/data/datawkt/datawkt.gen.go
	spdx-go-data --package dataspdx > private/gen/data/dataspdx/dataspdx.gen.go

prepostgenerate:: godata

.PHONY: licenseheader
licenseheader: installlicense-header installgit-ls-files-unstaged
	@echo license-header \
		--license-type "$(LICENSE_HEADER_LICENSE_TYPE)" \
		--copyright-holder "$(LICENSE_HEADER_COPYRIGHT_HOLDER)" \
		--year-range "$(LICENSE_HEADER_YEAR_RANGE)" \
		ALL_FILES
	@license-header \
		--license-type "$(LICENSE_HEADER_LICENSE_TYPE)" \
		--copyright-holder "$(LICENSE_HEADER_COPYRIGHT_HOLDER)" \
		--year-range "$(LICENSE_HEADER_YEAR_RANGE)" \
		$(shell git-ls-files-unstaged | grep -v $(patsubst %,-e %,$(sort $(LICENSE_HEADER_IGNORES))))

licensegenerate:: licenseheader

.PHONY: privateusage
privateusage:
	bash make/buf/scripts/privateusage.bash

postprepostgenerate:: privateusage

bufgeneratedeps:: \
	$(PROTOC_GEN_GO) $(PROTOC_GEN_CONNECT_GO)

.PHONY: bufgeneratecleango
bufgeneratecleango:
	rm -rf private/gen/proto

.PHONY: bufgeneratecleanbuflinttestdata
bufgeneratecleanbuflinttestdata:
	rm -rf private/bufpkg/bufcheck/buflint/testdata/deps

bufgenerateclean:: \
	bufgeneratecleango \
	bufgeneratecleanbuflinttestdata

.PHONY: bufgenerateprotogo
bufgenerateprotogo:
	$(BUF_BIN) generate proto --template data/template/buf.go.gen.yaml
	$(BUF_BIN) generate buf.build/grpc/grpc --type grpc.reflection.v1.ServerReflection --template data/template/buf.go.gen.yaml

.PHONY: bufgenerateprotogoclient
bufgenerateprotogoclient:
	$(BUF_BIN) generate proto --template data/template/buf.go-client.gen.yaml

.PHONY: bufgeneratebuflinttestdata
bufgeneratebuflinttestdata:
	$(BUF_BIN) export buf.build/bufbuild/protovalidate:$(PROTOVALIDATE_VERSION) --output private/bufpkg/bufcheck/buflint/testdata/deps/protovalidate

bufgeneratesteps:: \
	bufgenerateprotogo \
	bufgenerateprotogoclient \
	bufgeneratebuflinttestdata

.PHONY: bufrelease
bufrelease: $(MINISIGN)
	DOCKER_IMAGE=golang:1.21-bullseye bash make/buf/scripts/release.bash

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
	docker pull golang:$(GOVERSION)-bullseye
	docker pull golang:$(GOVERSION)-alpine3.18
	$(SED_I) "s/golang:1\.[0-9][0-9]*\.[0-9][0-9]*/golang:$(GOVERSION)/g" $(shell git-ls-files-unstaged | grep Dockerfile)
	$(SED_I) "s/golang:1\.[0-9][0-9]*\.[0-9][0-9]*/golang:$(GOVERSION)/g" $(shell git-ls-files-unstaged | grep \.mk$)
	$(SED_I) "s/go-version: 1\.[0-9][0-9]*\.[0-9][0-9]*/go-version: $(GOVERSION)/g" $(shell git-ls-files-unstaged | grep \.github\/workflows | grep -v previous.yaml)

.PHONY: gofuzz
gofuzz: $(GO_FUZZ)
	@rm -rf $(TMP)/gofuzz
	@mkdir -p $(TMP)/gofuzz $(TMP)/gofuzz/corpus
	# go-fuzz-build requires github.com/dvyukov/go-fuzz be in go.mod, but we don't need that dependency otherwise.
	# This adds go-fuzz-dep to go.mod, runs go-fuzz-build, then restores go.mod.
	cp go.mod $(TMP)/go.mod.bak; cp go.sum $(TMP)/go.sum.bak
	go get github.com/dvyukov/go-fuzz/go-fuzz-dep@$(GO_FUZZ_VERSION)
	cd private/bufpkg/bufimage/bufimagebuild/bufimagebuildtesting; go-fuzz-build -o $(abspath $(TMP))/gofuzz/gofuzz.zip
	rm go.mod go.sum; mv $(TMP)/go.mod.bak go.mod; mv $(TMP)/go.sum.bak go.sum
	cp private/bufpkg/bufimage/bufimagebuild/bufimagebuildtesting/corpus/* $(TMP)/gofuzz/corpus
	go-fuzz -bin $(TMP)/gofuzz/gofuzz.zip -workdir $(TMP)/gofuzz $(GO_FUZZ_EXTRA_ARGS)

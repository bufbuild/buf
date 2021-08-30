# https://github.com/jhump/protoreflect/commits/master 20210817 checked 20210817
PROTOREFLECT_VERSION := db1a327a393ec12e4a565f4813c154abd843376a
GO_GET_PKGS := $(GO_GET_PKGS) \
	github.com/jhump/protoreflect@$(PROTOREFLECT_VERSION)
GO_BINS := $(GO_BINS) \
	cmd/buf \
	cmd/protoc-gen-buf-breaking \
	cmd/protoc-gen-buf-lint \
	cmd/protoc-gen-buf-check-breaking \
	cmd/protoc-gen-buf-check-lint \
	private/bufpkg/bufprotoplugin/cmd/protoc-gen-go-api \
	private/bufpkg/bufprotoplugin/cmd/protoc-gen-go-apiclient \
	private/bufpkg/bufprotoplugin/cmd/protoc-gen-go-apiclientgrpc \
	private/bufpkg/bufprotoplugin/cmd/protoc-gen-go-apiclienttwirp \
	private/pkg/bandeps/cmd/bandeps \
	private/pkg/git/cmd/git-ls-files-unstaged \
	private/pkg/storage/cmd/ddiff \
	private/pkg/storage/cmd/storage-go-data \
	private/pkg/licenseheader/cmd/license-header \
	private/pkg/spdx/cmd/spdx-go-data
GO_TEST_BINS := $(GO_TEST_BINS) \
	private/buf/cmd/buf/command/protoc/internal/protoc-gen-insertion-point-receiver \
	private/buf/cmd/buf/command/protoc/internal/protoc-gen-insertion-point-writer
DOCKER_BINS := $(DOCKER_BINS) buf
FILE_IGNORES := $(FILE_IGNORES) \
	.build/ \
	.ctrlp \
	.vscode/ \
	private/buf/cmd/buf/cache/ \
	private/buf/cmd/buf/workspacetests/other/proto/workspacetest/cache/ \
	private/bufpkg/buftesting/cache/ \
	private/pkg/storage/storageos/tmp/
LICENSE_HEADER_LICENSE_TYPE := apache
LICENSE_HEADER_COPYRIGHT_HOLDER := Buf Technologies, Inc.
LICENSE_HEADER_YEAR_RANGE := 2020-2021
LICENSE_HEADER_IGNORES := \/testdata enterprise

BUF_LINT_INPUT := .
BUF_BREAKING_INPUT := .
BUF_BREAKING_AGAINST_INPUT ?= .git\#branch=main

include make/go/bootstrap.mk
include make/go/dep_buf.mk
include make/go/dep_minisign.mk
include make/go/dep_protoc.mk
include make/go/dep_protoc_gen_go.mk
include make/go/dep_protoc_gen_go_grpc.mk
include make/go/dep_protoc_gen_twirp.mk
include make/go/dep_go_fuzz.mk
include make/go/go.mk
include make/go/docker.mk
include make/go/buf.mk

installtest:: $(PROTOC) $(PROTOC_GEN_GO)

.PHONY: bandeps
bandeps: installbandeps
	bandeps -f data/bandeps/bandeps.yaml

postlonglint:: bandeps

.PHONY: godata
godata: installspdx-go-data installstorage-go-data $(PROTOC)
	rm -rf private/gen/data
	mkdir -p private/gen/data/datawkt
	mkdir -p private/gen/data/dataspdx
	storage-go-data $(CACHE_INCLUDE) --package datawkt > private/gen/data/datawkt/datawkt.gen.go
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
	installprotoc-gen-go-api \
	installprotoc-gen-go-apiclient \
	installprotoc-gen-go-apiclientgrpc \
	installprotoc-gen-go-apiclienttwirp \
	$(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) $(PROTOC_GEN_TWIRP)

.PHONY: bufgeneratecleango
bufgeneratecleango:
	rm -rf private/gen/proto

bufgenerateclean:: bufgeneratecleango

.PHONY: bufgenerateprotogo
bufgenerateprotogo:
	buf generate proto --template data/template/buf.go.gen.yaml

.PHONY: bufgenerateprotogoclient
bufgenerateprotogoclient:
	buf generate proto --template data/template/buf.go-client.gen.yaml

bufgeneratesteps:: \
	bufgenerateprotogo \
	bufgenerateprotogoclient

.PHONY: bufrelease
bufrelease: $(MINISIGN)
	DOCKER_IMAGE=golang:1.17.0-buster bash make/buf/scripts/release.bash

# We have to manually set the Homebrew version on the Homebrew badge as there
# is no badge on shields.io for Homebrew packages outside of homebrew-core

.PHONY: updatehomebrewbadge
updatehomebrewbadge:
	$(SED_I) "s/HOMEBREW_VERSION/v$(shell bash make/buf/scripts/homebrewversion.bash)/g" README.md

.PHONY: updateversion
updateversion:
ifndef VERSION
	$(error "VERSION must be set")
endif
	$(SED_I) "s/Version.*=.*\"0\.[0-9][0-9]*\.[0-9][0-9]*.*\"/Version = \"$(VERSION)\"/g" private/buf/bufcli/bufcli.go
	gofmt -s -w private/buf/bufcli/bufcli.go

.PHONY: updategoversion
updategoversion: installgit-ls-files-unstaged
ifndef GOVERSION
	$(error "GOVERSION must be set")
endif
	# make sure both of these docker images exist
	# the release of these images will lag the actual release
	docker pull golang:$(GOVERSION)-buster
	docker pull golang:$(GOVERSION)-alpine3.14
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

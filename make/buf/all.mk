GO_ALL_REPO_PKGS := ./cmd/... ./private/...
# TODO: Remove go-winio v0.6.1 and otel v1.24.0 when we no longer need to support Golang <1.21
# TODO: Remove antlr when we no longer need Golang <1.22
GO_GET_PKGS := $(GO_GET_PKGS) \
	github.com/Microsoft/go-winio@v0.6.1 \
	go.opentelemetry.io/otel@v1.24.0 \
	go.opentelemetry.io/otel/sdk@v1.24.0 \
	go.opentelemetry.io/otel/trace@v1.24.0 \
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp@v1.24.0 \
	github.com/antlr4-go/antlr/v4@v4.13.0
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
	private/pkg/licenseheader/cmd/license-header \
	private/pkg/spdx/cmd/spdx-go-data
GO_TEST_BINS := $(GO_TEST_BINS) \
	private/buf/cmd/buf/command/alpha/protoc/internal/protoc-gen-insertion-point-receiver \
	private/buf/cmd/buf/command/alpha/protoc/internal/protoc-gen-insertion-point-writer \
	private/buf/cmd/buf/command/generate/internal/protoc-gen-top-level-type-names-yaml
GO_MOD_VERSION := 1.20
DOCKER_BINS := $(DOCKER_BINS) buf
FILE_IGNORES := $(FILE_IGNORES) \
	.build/ \
	.ctrlp \
	.idea/ \
	.vscode/ \
	private/buf/cmd/buf/command/alpha/protoc/test.txt \
	private/bufpkg/buftesting/cache/ \
	private/buf/buftesting/cache/ \
	private/pkg/storage/storageos/tmp/
LICENSE_HEADER_LICENSE_TYPE := apache
LICENSE_HEADER_COPYRIGHT_HOLDER := Buf Technologies, Inc.
LICENSE_HEADER_YEAR_RANGE := 2020-2024
LICENSE_HEADER_IGNORES := \/testdata enterprise
PROTOVALIDATE_VERSION := v0.6.0
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
godata: installspdx-go-data installwkt-go-data installbuf-legacyfederation-go-data $(PROTOC)
	rm -rf private/gen/data/datawkt
	mkdir -p private/gen/data/datawkt
	wkt-go-data "$(CACHE_INCLUDE)" --package datawkt --protobuf-version "$(PROTOC_VERSION)" > private/gen/data/datawkt/datawkt.gen.go
	rm -rf private/gen/data/dataspdx
	mkdir -p private/gen/data/dataspdx
	spdx-go-data --package dataspdx > private/gen/data/dataspdx/dataspdx.gen.go
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

bufgeneratedeps:: $(PROTOC_GEN_GO) $(PROTOC_GEN_CONNECT_GO)

.PHONY: bufgeneratecleango
bufgeneratecleango:
	rm -rf private/gen/proto

.PHONY: bufgeneratecleanbuflinttestdata
bufgeneratecleanbuflinttestdata:
	rm -rf private/bufpkg/bufcheck/buflint/testdata/protovalidate/vendor/protovalidate

bufgenerateclean:: \
	bufgeneratecleango \
	bufgeneratecleanbuflinttestdata

.PHONY: bufgeneratego
bufgeneratego:
	$(BUF_BIN) generate --template data/template/buf.go.gen.yaml
	$(BUF_BIN) generate --template data/template/buf.go-client.gen.yaml

.PHONY: bufgeneratebuflinttestdata
bufgeneratebuflinttestdata:
	$(BUF_BIN) export \
		buf.build/bufbuild/protovalidate:$(PROTOVALIDATE_VERSION) \
		--output private/bufpkg/bufcheck/buflint/testdata/protovalidate/vendor/protovalidate

bufgeneratesteps:: \
	bufgeneratego \
	bufgeneratebuflinttestdata

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
	docker pull golang:$(GOVERSION)-alpine3.20
	$(SED_I) "s/golang:1\.[0-9][0-9]*/golang:$(GOVERSION)/g" $(shell git-ls-files-unstaged | grep Dockerfile)
	$(SED_I) "s/golang:1\.[0-9][0-9]*/golang:$(GOVERSION)/g" $(shell git-ls-files-unstaged | grep \.mk$)
	$(SED_I) "s/go-version: '1\.[0-9][0-9].x'/go-version: '$(GOVERSION).x'/g" $(shell git-ls-files-unstaged | grep \.github\/workflows | grep -v previous.yaml)

.PHONY: bufimageutilupdateexpectations
bufimageutilupdateexpectations:
	# You may need to run this after updating protoc versions
	BUFBUILD_BUF_BUFIMAGEUTIL_SHOULD_UPDATE_EXPECTATIONS=1 go test -parallel 1 ./private/bufpkg/bufimage/bufimageutil

.PHONY: newtodos
newtodos:
	@bash make/buf/scripts/newtodos.bash | grep -v FUTURE

.PHONY: newtodofiles
newtodofiles:
	@bash make/buf/scripts/newtodos.bash | grep -v FUTURE | cut -f 1 -d : | sort | uniq

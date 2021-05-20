BUF_BIN ?= cmd/buf

PROTOREFLECT_VERSION := 8255811fc3c054aab548f7208e1471b668f4c5b3
GO_GET_PKGS := $(GO_GET_PKGS) \
	github.com/jhump/protoreflect@$(PROTOREFLECT_VERSION)
GO_BINS := $(GO_BINS) \
	$(BUF_BIN) \
	cmd/protoc-gen-buf-breaking \
	cmd/protoc-gen-buf-lint \
	cmd/protoc-gen-buf-check-breaking \
	cmd/protoc-gen-buf-check-lint \
	internal/pkg/git/cmd/git-ls-files-unstaged \
	internal/pkg/storage/cmd/ddiff \
	internal/pkg/storage/cmd/storage-go-data \
	internal/pkg/licenseheader/cmd/license-header \
	internal/pkg/spdx/cmd/spdx-go-data \
	internal/protoplugin/cmd/protoc-gen-go-api \
	internal/protoplugin/cmd/protoc-gen-go-apiclient \
	internal/protoplugin/cmd/protoc-gen-go-apiclientgrpc \
	internal/protoplugin/cmd/protoc-gen-go-apiclienttwirp \
	internal/protoplugin/cmd/protoc-gen-proxy
GO_TEST_BINS := $(GO_TEST_BINS) \
	internal/buf/cmd/buf/command/protoc/internal/protoc-gen-insertion-point-receiver \
	internal/buf/cmd/buf/command/protoc/internal/protoc-gen-insertion-point-writer
DOCKER_BINS := $(DOCKER_BINS) buf
FILE_IGNORES := $(FILE_IGNORES) \
	.build/ \
	.vscode/ \
	internal/buf/cmd/buf/cache/ \
	internal/buf/cmd/buf/workspacetests/other/proto/workspacetest/cache/ \
	internal/buf/internal/buftesting/cache/ \
	internal/pkg/storage/storageos/tmp/
LICENSE_HEADER_LICENSE_TYPE := apache
LICENSE_HEADER_COPYRIGHT_HOLDER := Buf Technologies, Inc.
LICENSE_HEADER_YEAR_RANGE := 2020-2021
LICENSE_HEADER_IGNORES := \/testdata enterprise

include make/go/bootstrap.mk
include make/go/dep_buf.mk
include make/go/dep_protoc.mk
include make/go/dep_protoc_gen_go.mk
include make/go/dep_protoc_gen_go_grpc.mk
include make/go/dep_protoc_gen_twirp.mk
include make/go/dep_go_fuzz.mk
include make/go/go.mk
include make/go/docker.mk

# Settable
BUF_BREAKING_PROTO_INPUT ?= .git\#branch=master,subdir=proto

installtest:: $(PROTOC) $(PROTOC_GEN_GO)

.PHONY: godata
godata: installspdx-go-data installstorage-go-data $(PROTOC)
	rm -rf internal/gen/data
	mkdir -p internal/gen/data/datawkt
	mkdir -p internal/gen/data/dataspdx
	storage-go-data $(CACHE_INCLUDE) --package datawkt > internal/gen/data/datawkt/datawkt.gen.go
	spdx-go-data --package dataspdx > internal/gen/data/dataspdx/dataspdx.gen.go

prepostgenerate:: godata

.PHONY: licenseheader
licenseheader: installlicense-header installgit-ls-files-unstaged
	license-header \
		--license-type "$(LICENSE_HEADER_LICENSE_TYPE)" \
		--copyright-holder "$(LICENSE_HEADER_COPYRIGHT_HOLDER)" \
		--year-range "$(LICENSE_HEADER_YEAR_RANGE)" \
		$(shell git-ls-files-unstaged | grep -v $(patsubst %,-e %,$(sort $(LICENSE_HEADER_IGNORES))))

licensegenerate:: licenseheader

# Runtime variable BUFBINARYHEAD uses the buf binary installed
# from this repository to do buf generate instead of using a
# downloaded release binary. We typically use the downloaded
# release binary as when generating, the codebase can break,
# and if we have generate depend on installing buf, we have
# a chicken or egg problem.

# TODO: Remove this after this PR is merged.
BUFBINARYHEAD=1

.PHONY: bufbinaryinstall
ifeq ($(BUFBINARYHEAD),)
BUF_BINARY := $(CACHE_BIN)/buf
bufbinaryinstall: $(BUF)
else
BUF_BINARY := $(CACHE_GOBIN)/buf
bufbinaryinstall: installbuf
endif

.PHONY: bufgeneratedeps
bufgeneratedeps:: \
	bufbinaryinstall \
	installprotoc-gen-go-api \
	installprotoc-gen-go-apiclient \
	installprotoc-gen-go-apiclientgrpc \
	installprotoc-gen-go-apiclienttwirp \
	$(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) $(PROTOC_GEN_TWIRP)

.PHONY: bufgenerateclean
bufgenerateclean::

.PHONY: bufgeneratecleango
bufgeneratecleango:
	rm -rf internal/gen/proto

bufgenerateclean:: bufgeneratecleango

.PHONY: bufgeneratesteps
bufgeneratesteps::

.PHONY: bufgenerateprotogo
bufgenerateprotogo:
	$(BUF_BINARY) generate proto --template data/buf/template/buf.go.gen.yaml

.PHONY: bufgenerateprotogoclient
bufgenerateprotogoclient:
	$(BUF_BINARY) generate proto --template data/buf/template/buf.go-client.gen.yaml

bufgeneratesteps:: \
	bufgenerateprotogo \
	bufgenerateprotogoclient

.PHONY: bufgenerate
bufgenerate:
	$(MAKE) bufgeneratedeps
	$(MAKE) bufgenerateclean
	$(MAKE) bufgeneratesteps

pregenerate:: bufgenerate

.PHONY: buflintproto
buflintproto: bufbinaryinstall
	$(BUF_BINARY) lint proto

.PHONY: bufbreakingproto
bufbreakingproto: bufbinaryinstall
	-$(BUF_BINARY) breaking proto --against $(BUF_BREAKING_PROTO_INPUT)

postlint:: buflintproto bufbreakingproto

.PHONY: bufrelease
bufrelease:
	DOCKER_IMAGE=golang:1.16.4-buster bash make/buf/scripts/release.bash

.PHONY: gofuzz
gofuzz: $(GO_FUZZ)
	@rm -rf $(TMP)/gofuzz
	@mkdir -p $(TMP)/gofuzz
	cd internal/buf/bufbuild/bufbuildtesting; go-fuzz-build -o $(abspath $(TMP))/gofuzz/gofuzz.zip
	go-fuzz -bin $(TMP)/gofuzz/gofuzz.zip -workdir $(TMP)/gofuzz

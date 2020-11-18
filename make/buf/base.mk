BUF_BIN ?= cmd/buf

PROTOREFLECT_VERSION := 0050302ca4f8e433a0de84423d9de3dce02299dc
# Remove when https://github.com/spf13/cobra/pull/1070 is released
COBRA_VERSION := b97b5ead31f7d34f764ac8666e40c214bb8e06dc
GO_GET_PKGS := $(GO_GET_PKGS) \
	github.com/jhump/protoreflect@$(PROTOREFLECT_VERSION) \
	github.com/spf13/cobra@$(COBRA_VERSION)
GO_BINS := $(GO_BINS) \
	$(BUF_BIN) \
	cmd/protoc-gen-buf-check-breaking \
	cmd/protoc-gen-buf-check-lint \
	internal/pkg/storage/cmd/ddiff \
	internal/pkg/storage/cmd/storage-go-binary-data \
	internal/pkg/app/appproto/appprotoexec/cmd/protoc-gen-proxy
GO_TEST_BINS := $(GO_TEST_BINS) \
	internal/buf/cmd/buf/command/protoc/internal/protoc-gen-insertion-point-receiver \
	internal/buf/cmd/buf/command/protoc/internal/protoc-gen-insertion-point-writer
DOCKER_BINS := $(DOCKER_BINS) buf
FILE_IGNORES := $(FILE_IGNORES) \
	.build/ \
	.vscode/ \
	internal/buf/cmd/buf/cache/ \
	internal/buf/internal/buftesting/cache/ \
	internal/pkg/storage/storageos/tmp/

USE_BUF_GENERATE := true

# Set to an alternative location for the buf binary when doing
# code-breaking changes that will result in installbuf failing
BUF_GENERATE_BINARY_PATH ?= buf

include make/go/bootstrap.mk
include make/go/dep_protoc.mk
include make/go/dep_protoc_gen_go.mk
include make/go/dep_go_fuzz.mk
include make/go/go.mk
include make/go/codecov.mk
include make/go/docker.mk

# Settable
BUF_BREAKING_INPUT ?= .git\#branch=master

installtest:: $(PROTOC) $(PROTOC_GEN_GO)

.PHONY: wkt
wkt: installstorage-go-binary-data $(PROTOC)
	rm -rf internal/gen/data
	mkdir -p internal/gen/data/wkt
	storage-go-binary-data $(CACHE_INCLUDE) --package wkt > internal/gen/data/wkt/wkt.gen.go

prepostgenerate:: wkt

.PHONY: bufgeneratedeps
bufgeneratedeps:: $(PROTOC_GEN_GO)
ifeq ($(BUF_GENERATE_BINARY_PATH),buf)
bufgeneratedeps:: installbuf
endif

.PHONY: bufgenerateclean
bufgenerateclean::

.PHONY: bufgeneratecleango
bufgeneratecleango:
	rm -rf internal/gen/proto/go

bufgenerateclean:: bufgeneratecleango

.PHONY: bufgenerate
bufgenerate:
	$(MAKE) bufgeneratedeps
	$(MAKE) bufgenerateclean
	$(BUF_GENERATE_BINARY_PATH) generate

pregenerate:: bufgenerate

.PHONY: buflint
buflint: installbuf
	buf check lint

.PHONY: bufbreaking
bufbreaking: installbuf
ifneq ($(BUF_BREAKING_INPUT),)
	-buf check breaking --against $(BUF_BREAKING_INPUT)
else
	@echo "skipping make bufbreaking" >&2
endif

postlint:: buflint bufbreaking

.PHONY: bufrelease
bufrelease:
	DOCKER_IMAGE=golang:1.15.5-buster bash make/buf/scripts/release.bash

.PHONY: gofuzz
gofuzz: $(GO_FUZZ)
	@rm -rf $(TMP)/gofuzz
	@mkdir -p $(TMP)/gofuzz
	cd internal/buf/bufbuild/bufbuildtesting; go-fuzz-build -o $(abspath $(TMP))/gofuzz/gofuzz.zip
	go-fuzz -bin $(TMP)/gofuzz/gofuzz.zip -workdir $(TMP)/gofuzz

BUF_BIN ?= cmd/buf

PROTOREFLECT_VERSION := v1.8.1
# Remove when https://github.com/spf13/cobra/pull/1070 is released
COBRA_VERSION := b97b5ead31f7d34f764ac8666e40c214bb8e06dc
GO_GET_PKGS := $(GO_GET_PKGS) \
	github.com/jhump/protoreflect@$(PROTOREFLECT_VERSION) \
	github.com/spf13/cobra@$(COBRA_VERSION)
GO_BINS := $(GO_BINS) \
	$(BUF_BIN) \
	cmd/protoc-gen-buf-breaking \
	cmd/protoc-gen-buf-lint \
	cmd/protoc-gen-buf-check-breaking \
	cmd/protoc-gen-buf-check-lint \
	internal/pkg/storage/cmd/ddiff \
	internal/pkg/storage/cmd/storage-go-binary-data
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
include make/go/dep_protoc_gen_go_grpc.mk
include make/go/dep_protoc_gen_twirp.mk
include make/go/dep_go_fuzz.mk
include make/go/go.mk
include make/go/docker.mk

# Settable
BUF_BREAKING_PROTO_INPUT ?= .git\#branch=master,subdir=proto

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
bufgenerate::

.PHONY: bufgenerateprotogo
bufgenerateprotogo:
	$(BUF_GENERATE_BINARY_PATH) generate proto \
		--output internal/gen/proto \
		--template data/buf/template/buf.go.gen.yaml

bufgenerate:: bufgenerateprotogo

.PHONY: bufgeneratesteps
bufgeneratesteps:
	$(MAKE) bufgeneratedeps
	$(MAKE) bufgenerateclean
	$(MAKE) bufgenerate

pregenerate:: bufgeneratesteps

.PHONY: buflintproto
buflintproto: installbuf
	buf lint proto

.PHONY: bufbreakingproto
bufbreakingproto: installbuf
	-buf breaking proto --against $(BUF_BREAKING_PROTO_INPUT)

postlint:: buflintproto bufbreakingproto

.PHONY: bufrelease
bufrelease:
	DOCKER_IMAGE=golang:1.15.6-buster bash make/buf/scripts/release.bash

.PHONY: gofuzz
gofuzz: $(GO_FUZZ)
	@rm -rf $(TMP)/gofuzz
	@mkdir -p $(TMP)/gofuzz
	cd internal/buf/bufbuild/bufbuildtesting; go-fuzz-build -o $(abspath $(TMP))/gofuzz/gofuzz.zip
	go-fuzz -bin $(TMP)/gofuzz/gofuzz.zip -workdir $(TMP)/gofuzz

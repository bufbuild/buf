BUF_BIN ?= cmd/buf

PROTOREFLECT_VERSION := 8d2d934ce5650f450ffe1f441313e26c75e1fdab
GO_GET_PKGS := $(GO_GET_PKGS) github.com/jhump/protoreflect@$(PROTOREFLECT_VERSION)
GO_BINS := $(GO_BINS) $(BUF_BIN) \
	cmd/protoc-gen-buf-check-breaking \
	cmd/protoc-gen-buf-check-lint \
	internal/pkg/storage/cmd/storage-go-binary-data
GO_LINT_IGNORES := $(GO_LINT_IGNORES) /internal/buf/cmd/buf/internal/protoc
DOCKER_BINS := $(DOCKER_BINS) buf
PROTO_PATH := proto
PROTOC_GEN_GO_OUT := internal/gen/proto/go/v1
FILE_IGNORES := $(FILE_IGNORES) \
	.build/ \
	.vscode/ \
	internal/buf/internal/buftesting/cache/ \
	internal/pkg/storage/storageos/tmp/


include make/go/bootstrap.mk
include make/go/go.mk
include make/go/codecov.mk
include make/go/dep_protoc.mk
include make/go/docker.mk
include make/go/protoc_gen_go.mk
include make/go/dep_go_fuzz.mk

.PHONY: wkt
wkt: installstorage-go-binary-data $(PROTOC)
	rm -rf internal/gen/data
	mkdir -p internal/gen/data/wkt
	storage-go-binary-data $(CACHE_INCLUDE) --package wkt > internal/gen/data/wkt/wkt.gen.go

pregenerate:: wkt

.PHONY: buflint
buflint: installbuf
	buf check lint

.PHONY: bufbreaking
bufbreaking: installbuf
	@ if [ -d .git ]; then \
			$(MAKE) bufbreakinginternal; \
		else \
			echo "skipping make bufbreaking due to no .git repository" >&2; \
		fi

.PHONY: bufbreakinginternal
bufbreakinginternal:
	-buf check breaking --against-input '.git#branch=master'

postlint:: buflint bufbreaking

.PHONY: bufrelease
bufrelease:
	DOCKER_IMAGE=golang:1.14.4-buster bash make/buf/scripts/release.bash

.PHONY: gofuzz
gofuzz: $(GO_FUZZ)
	@rm -rf $(TMP)/gofuzz
	@mkdir -p $(TMP)/gofuzz
	cd internal/buf/bufbuild/bufbuildtesting; go-fuzz-build -o $(abspath $(TMP))/gofuzz/gofuzz.zip
	go-fuzz -bin $(TMP)/gofuzz/gofuzz.zip -workdir $(TMP)/gofuzz

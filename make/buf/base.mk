BUF_BIN ?= cmd/buf

PROTOREFLECT_VERSION := 11eaaf73e0ec2d04934c5d86d46577568463cc86
# Remove when https://github.com/spf13/cobra/pull/1070 is released
COBRA_VERSION := 884edc58ad08083e6c9a505041695aa2c3ca2d7a
# Remove when https://github.com/ory/dockertest/issues/208 is fixed
GOLANG_X_SYS_VERSION := f9321e4c35a6ee62ff0348f57ad5a3b5e73e299e
GO_GET_PKGS := $(GO_GET_PKGS) \
	github.com/jhump/protoreflect@$(PROTOREFLECT_VERSION) \
	github.com/spf13/cobra@$(COBRA_VERSION) \
	golang.org/x/sys@$(GOLANG_X_SYS_VERSION)
GO_BINS := $(GO_BINS) $(BUF_BIN) \
	cmd/protoc-gen-buf-check-breaking \
	cmd/protoc-gen-buf-check-lint \
	internal/pkg/storage/cmd/storage-go-binary-data \
	internal/pkg/app/appproto/appprotoexec/cmd/protoc-gen-proxy
GO_LINT_IGNORES := $(GO_LINT_IGNORES) /internal/buf/cmd/buf/command/protoc
DOCKER_BINS := $(DOCKER_BINS) buf
PROTO_PATH := proto
PROTOC_GEN_GO_OUT := internal/gen/proto/go
PROTOC_GEN_VALIDATE_OUT := internal/gen/proto/go
FILE_IGNORES := $(FILE_IGNORES) \
	.build/ \
	.vscode/ \
	internal/buf/internal/buftesting/cache/ \
	internal/pkg/storage/storageos/tmp/

PROTOC_USE_BUF_BY_DIR := true

include make/go/bootstrap.mk
include make/go/go.mk
include make/go/codecov.mk
include make/go/dep_protoc.mk
include make/go/docker.mk
include make/go/protoc_gen_go.mk
include make/go/protoc_gen_validate.mk
include make/go/dep_go_fuzz.mk

.PHONY: wkt
wkt: installstorage-go-binary-data $(PROTOC)
	rm -rf internal/gen/data
	mkdir -p internal/gen/data/wkt
	storage-go-binary-data $(CACHE_INCLUDE) --package wkt > internal/gen/data/wkt/wkt.gen.go

prepostgenerate:: wkt

.PHONY: prebuflint
prebuflint::

.PHONY: buflint
buflint: installbuf prebuflint
	buf check lint

.PHONY: prebufbreaking
prebufbreaking::

.PHONY: bufbreaking
bufbreaking: installbuf prebufbreaking
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
	DOCKER_IMAGE=golang:1.15.1-buster bash make/buf/scripts/release.bash

.PHONY: gofuzz
gofuzz: $(GO_FUZZ)
	@rm -rf $(TMP)/gofuzz
	@mkdir -p $(TMP)/gofuzz
	cd internal/buf/bufbuild/bufbuildtesting; go-fuzz-build -o $(abspath $(TMP))/gofuzz/gofuzz.zip
	go-fuzz -bin $(TMP)/gofuzz/gofuzz.zip -workdir $(TMP)/gofuzz

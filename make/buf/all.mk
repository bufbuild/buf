PROTO_PATH := proto
PROTOC_GEN_GO_OUT := internal/gen/proto
GO_BINS := $(GO_BINS) \
	buf \
	protoc-gen-buf-check-breaking \
	protoc-gen-buf-check-lint
GO_GET_PKGS := github.com/jhump/protoreflect@master

DOCKER_BINS := $(DOCKER_BINS) buf

FILE_IGNORES := $(FILE_IGNORES) .build/ internal/buf/bufbuild/cache/

include make/versions.mk
include make/base.mk
include make/dep_errcheck.mk
include make/dep_golint.mk
include make/dep_ineffassign.mk
include make/dep_protoc.mk
include make/dep_protoc_gen_go.mk
include make/dep_staticcheck.mk
include make/golang.mk
include make/protoc_gen_go.mk
include make/docker.mk


.PHONY: buflint
buflint: bufinstall
	buf check lint

.PHONY: bufbreaking
bufbreaking: bufinstall
	@ if [ -d .git ]; then \
			$(MAKE) bufbreakinginternal; \
		else \
			echo "skipping make bufbreaking due to no .git repository" >&2; \
		fi

.PHONY: bufbreakinginternal
bufbreakinginternal:
	-buf check breaking --against-input '.git#branch=master'

postlint:: buflint bufbreaking

.PHONY: bufdevinstall
bufdevinstall:
	go install -ldflags "-X github.com/bufbuild/buf/internal/buf/cmd/buf.develMode=1" ./cmd/buf

.PHONY: bufrelease
bufrelease: all
	DOCKER_IMAGE=golang:1.13.3-buster bash make/scripts/buf/release.bash

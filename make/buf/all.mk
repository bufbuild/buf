GO_BINS := $(GO_BINS) cmd/buf cmd/protoc-gen-buf-check-breaking cmd/protoc-gen-buf-check-lint
GO_GET_PKGS := $(GO_GET_PKGS) github.com/jhump/protoreflect@master
DOCKER_BINS := $(DOCKER_BINS) buf
PROTO_PATH := proto
PROTOC_GEN_GO_OUT := internal/gen/proto
FILE_IGNORES := $(FILE_IGNORES) .build/ internal/buf/bufbuild/cache/

include make/go/bootstrap.mk
include make/go/go.mk
include make/go/codecov.mk
include make/go/docker.mk
include make/go/protoc_gen_go.mk

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

.PHONY: installbufdev
installbufdev:
	go install -ldflags "-X github.com/bufbuild/buf/internal/buf/cmd/buf.develMode=1" ./cmd/buf

.PHONY: bufrelease
bufrelease: all
	DOCKER_IMAGE=golang:1.13.4-buster bash make/buf/scripts/release.bash

BUF_BIN ?= cmd/buf

GO_BINS := $(GO_BINS) $(BUF_BIN) cmd/protoc-gen-buf-check-breaking cmd/protoc-gen-buf-check-lint
GO_GET_PKGS := $(GO_GET_PKGS) github.com/jhump/protoreflect@master
DOCKER_BINS := $(DOCKER_BINS) buf
PROTO_PATH := proto
PROTOC_GEN_GO_OUT := internal/gen/proto/go/v1
FILE_IGNORES := $(FILE_IGNORES) .build/ internal/buf/bufbuild/cache/

include make/go/bootstrap.mk
include make/go/go.mk
include make/go/codecov.mk
include make/go/dep_protoc.mk
include make/go/docker.mk
include make/go/protoc_gen_go.mk

.PHONY: embed
embed: $(PROTOC)
	rm -rf internal/gen/embed
	mkdir -p internal/gen/embed/wkt
	go run internal/cmd/embed/main.go $(CACHE_INCLUDE) wkt .proto > internal/gen/embed/wkt/wkt.gen.go

pregenerate:: embed

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
bufrelease: all
	DOCKER_IMAGE=golang:1.14.1-buster bash make/buf/scripts/release.bash

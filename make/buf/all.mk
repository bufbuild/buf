PROTO_PATH := proto
PROTOC_GEN_GO_OUT := internal/gen/proto
GO_BINS := \
	buf \
	protoc-gen-buf-check-breaking \
	protoc-gen-buf-check-lint
GO_GET_PKGS := github.com/jhump/protoreflect@master

include make/buf/versions.mk

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

RELEASE_DOCKER_IMAGE := golang:1.13.1-buster

.PHONY: buflint
buflint: bufinstall
	buf check lint

.PHONY: bufbreaking
bufbreaking: bufinstall
	-buf check breaking --against-input '.git#branch=master'

postlint:: buflint bufbreaking

.PHONY: bufdevinstall
bufdevinstall:
	go install -ldflags "-X github.com/bufbuild/buf/internal/buf/cmd/buf.develMode=1" ./cmd/buf

.PHONY: bufrelease
bufrelease: all
	docker run --volume "$(CURDIR):/app" --workdir "/app" $(RELEASE_DOCKER_IMAGE) bash -x scripts/buf/release.bash

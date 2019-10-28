PROJECT := buf
GO_MODULE := github.com/bufbuild/buf
DOCKER_ORG := bufbuild
DOCKER_PROJECT := buf

GOPKGS ?= ./...

include make/buf/all.mk

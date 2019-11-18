MAKEGO := make/go
MAKEGO_REMOTE := https://github.com/bufbuild/makego.git
PROJECT := buf
GO_MODULE := github.com/bufbuild/buf
DOCKER_ORG := bufbuild
DOCKER_PROJECT := buf

include make/buf/all.mk

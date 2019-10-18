PROJECT := buf
GO_MODULE := github.com/bufbuild/buf
DOCKER_IMAGE := bufbuild/buf
DOCKER_FILE := Dockerfile.buf
DOCKER_DIR := /workspace

GOPKGS ?= ./...

include make/buf/all.mk

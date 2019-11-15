ifndef GO_MODULE
$(error GO_MODULE is not set)
endif
ifndef GOLINT
$(error GOLINT is not set)
endif
ifndef ERRCHECK
$(error ERRCHECK is not set)
endif
ifndef INEFFASSIGN
$(error INEFFASSIGN is not set)
endif
ifndef STATICCHECK
$(error STATICCHECK is not set)
endif
ifndef TMP
$(error TMP is not set)
endif

GO_BINS ?=
GO_GET_PKGS ?=

GOPKGS ?= ./...

.DEFAULT_GOAL := all

.PHONY: all
all:
	@$(MAKE) lint
	@$(MAKE) test

.PHONY: ci
ci:
	@$(MAKE) deps
	@$(MAKE) lint
	@$(MAKE) cover

.PHONY: updategodeps
updategodeps:
	rm -f go.mod go.sum
	go mod init $(GO_MODULE)
ifneq ($(GO_GET_PKGS),)
	go get $(GO_GET_PKGS)
endif
	go get -u -t ./...
	$(MAKE) generate
	$(MAKE)

.PHONY: godeps
godeps: deps
	go mod download

.PHONY: gofmtmodtidy
gofmtmodtidy:
	go fmt ./...
	go mod tidy -v

postgenerate:: gofmtmodtidy

.PHONY: golint
golint: $(GOLINT)
	golint -set_exit_status $(GOPKGS)

.PHONY: vet
vet:
	go vet $(GOPKGS)

.PHONY:
errcheck: $(ERRCHECK)
	errcheck $(GOPKGS)

.PHONY:
ineffassign: $(INEFFASSIGN)
	ineffassign .

.PHONY: staticcheck
staticcheck: $(STATICCHECK)
	staticcheck $(GOPKGS)

.PHONY: postlint
postlint::

.PHONY: lint
lint:
	@$(MAKE) checknodiffgenerated
	@$(MAKE) golint vet errcheck ineffassign staticcheck
	@$(MAKE) postlint

.PHONY: prebuild
prebuild::

.PHONY: build
build: prebuild
	go build ./...

.PHONY: pretest
pretest::

.PHONY: test
test: pretest
	go test $(GOPKGS)

.PHONY: deppkgs
deppkgs:
	@go list -f '{{join .Deps "\n"}}' $(GOPKGS) | xargs go list -f '{{if not .Standard}}{{.ImportPath}}{{end}}'

.PHONY: coverpkgs
coverpkgs:
	@go list $(GOPKGS) | grep -v \/gen\/ | tr '\n' ',' | sed "s/,$$//"

.PHONY: cover
cover: pretest
	@mkdir -p $(TMP)
	@rm -f $(TMP)/cover.txt $(TMP)/cover.html
	go test -race -coverprofile=$(TMP)/cover.txt -coverpkg=$(shell GOPKGS=$(GOPKGS) $(MAKE) -s coverpkgs) $(GOPKGS)
	@go tool cover -html=$(TMP)/cover.txt -o $(TMP)/cover.html
	@echo
	@go tool cover -func=$(TMP)/cover.txt | grep total
	@echo
	@if [ -z "$$OPEN" ]; then echo open $(TMP)/cover.html; else open $(TMP)/cover.html; fi

.PHONY: codecov
codecov:
	bash <(curl -s https://codecov.io/bash) -f $(TMP)/cover.txt

.PHONY: codecovcopyfile
codecovcopyfile:
	cp make/assets/.codecov.yml $(CURDIR)/.codecov.yml

pregenerate:: codecovcopyfile

.PHONY: install
install::

define gobinfunc
.PHONY: $(1)install
$(1)install:
	go install ./cmd/$(1)

install:: $(1)install
endef

$(foreach gobin,$(GO_BINS),$(eval $(call gobinfunc,$(gobin))))

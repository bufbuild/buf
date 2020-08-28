# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/dep_errcheck.mk)
$(call _conditional_include,$(MAKEGO)/dep_golint.mk)
$(call _conditional_include,$(MAKEGO)/dep_staticcheck.mk)
# Must be set
$(call _assert_var,GO_MODULE)
$(call _assert_var,GOLINT)
$(call _assert_var,ERRCHECK)
$(call _assert_var,STATICCHECK)
$(call _assert_var,TMP)
$(call _assert_var,OPEN_CMD)

# Settable
GO_BINS ?=
# Settable
GO_GET_PKGS ?=
# Settable
GO_LINT_IGNORES := $(GO_LINT_IGNORES) \/gen\/
# Settable
GO_MOD_VERSION ?= 1.14

# Runtime
GOPKGS ?= ./...
# Runtime GONOTESTCACHE
# Runtime COVEROPEN

COVER_HTML := $(TMP)/cover.html
COVER_TXT := $(TMP)/cover.txt

ifdef GONOTESTCACHE
GO_TEST_FLAGS := -count=1
else
GO_TEST_FLAGS :=
endif

.DEFAULT_GOAL := all

.PHONY: all
all:
	@$(MAKE) lint
	@$(MAKE) test

.PHONY: shortall
shortall:
	@$(MAKE) lint
	@$(MAKE) shorttest

.PHONY: ci
ci:
	@$(MAKE) deps
	@$(MAKE) lint
	@$(MAKE) cover

.PHONY: updategodeps
updategodeps:
	rm -f go.mod go.sum
	go mod init $(GO_MODULE)
	go mod edit -go=$(GO_MOD_VERSION)
	go get -u -t ./... $(GO_GET_PKGS)
ifneq ($(GO_GET_PKGS),)
	go get $(sort $(GO_GET_PKGS))
endif
	$(MAKE) generate
	$(MAKE)

initmakego:: updategodeps

.PHONY: godeps
godeps: deps
	go mod download

.PHONY: gofmtmodtidy
gofmtmodtidy:
	gofmt -s -w $(shell find . -name '*.go')
	go mod tidy -v

postgenerate:: gofmtmodtidy

.PHONY: golint
golint: __go_lint_pkgs $(GOLINT)
	golint -set_exit_status $(GO_LINT_PKGS)

.PHONY: vet
vet: __go_lint_pkgs
	go vet $(GO_LINT_PKGS)

.PHONY:
errcheck: __go_lint_pkgs $(ERRCHECK)
	errcheck $(GO_LINT_PKGS)

.PHONY: staticcheck
staticcheck: __go_lint_pkgs $(STATICCHECK)
	staticcheck $(GO_LINT_PKGS)

.PHONY: postlint
postlint::

.PHONY: lint
lint:
	@$(MAKE) checknodiffgenerated
	@$(MAKE) golint vet errcheck staticcheck
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
	go test $(GO_TEST_FLAGS) $(GOPKGS)

.PHONY: shorttest
shorttest: pretest
	go test -test.short $(GO_TEST_FLAGS) $(GOPKGS)

.PHONY: deppkgs
deppkgs:
	@go list -f '{{join .Deps "\n"}}' $(GOPKGS) | xargs go list -f '{{if not .Standard}}{{.ImportPath}}{{end}}'

.PHONY: coverpkgs
coverpkgs:
	@go list $(GOPKGS) | grep -v \/gen\/ | tr '\n' ',' | sed "s/,$$//"

.PHONY: cover
cover: pretest
	@mkdir -p $(dir $(COVER_HTML)) $(dir $(COVER_TXT))
	@rm -f $(COVER_HTML) $(COVER_TXT)
	go test -race -coverprofile=$(COVER_TXT) -coverpkg=$(shell GOPKGS=$(GOPKGS) $(MAKE) -s coverpkgs) $(GOPKGS)
	@go tool cover -html=$(COVER_TXT) -o $(COVER_HTML)
	@echo
	@go tool cover -func=$(COVER_TXT) | grep total
	@echo
ifndef COVEROPEN
	@echo $(OPEN_CMD) $(COVER_HTML)
else
	$(OPEN_CMD) $(COVER_HTML)
endif

.PHONY: install
install::

define gobinfunc
.PHONY: install$(notdir $(1))
install$(notdir $(1)):
	go install ./$(1)

install:: install$(notdir $(1))
endef

$(foreach gobin,$(sort $(GO_BINS)),$(eval $(call gobinfunc,$(gobin))))
$(foreach gobin,$(sort $(GO_BINS)),$(eval FILE_IGNORES := $(FILE_IGNORES) $(gobin)/$(notdir $(gobin))))

.PHONY: __go_lint_pkgs
__go_lint_pkgs:
ifdef GO_LINT_IGNORES
	$(eval GO_LINT_PKGS := $(shell go list $(GOPKGS) | grep -v $(patsubst %,-e %,$(sort $(GO_LINT_IGNORES)))))
else
	$(eval GO_LINT_PKGS := $(GOPKGS))
endif

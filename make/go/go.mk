# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/dep_golangci_lint.mk)
# Must be set
$(call _assert_var,GO_MODULE)
$(call _assert_var,GOLANGCI_LINT)
$(call _assert_var,TMP)
$(call _assert_var,OPEN_CMD)

# Settable
GO_BINS ?=
# Settable
GO_TEST_BINS ?=
# Settable
GO_GET_PKGS ?=
# Settable
GO_MOD_VERSION ?= 1.14

# Runtime
GOPKGS ?= ./...
# Runtime
GOLANGCILINTTIMEOUT ?= 2m0s
# Runtime GONOTESTCACHE
# Runtime COVEROPEN

COVER_HTML := $(TMP)/cover.html
COVER_TXT := $(TMP)/cover.txt

ifdef GONOTESTCACHE
GO_TEST_FLAGS := -count=1
else
GO_TEST_FLAGS :=
endif

.DEFAULT_GOAL := shortall

.PHONY: all
all:
	@$(MAKE) lint
	@$(MAKE) test

postupgrade:: all

.PHONY: shortall
shortall:
	@$(MAKE) lint
	@$(MAKE) shorttest

.PHONY: ci
ci:
	@$(MAKE) lint
	@$(MAKE) cover

.PHONY: upgradegodeps
upgradegodeps:
	rm -f go.mod go.sum
	go mod init $(GO_MODULE)
	go mod edit -go=$(GO_MOD_VERSION)
	go get -u -t ./... $(GO_GET_PKGS)
ifneq ($(GO_GET_PKGS),)
	go get $(sort $(GO_GET_PKGS))
endif

preupgrade:: upgradegodeps

initmakego:: upgradegodeps

.PHONY: godeps
godeps: deps
	go mod download

.PHONY: gofmtmodtidy
gofmtmodtidy:
	gofmt -s -w $(shell find . -name '*.go')
	go mod tidy -v

postgenerate:: gofmtmodtidy

.PHONY: checknonolint
checknonolint:
	@if grep '//nolint' $(shell find . -name '*.go'); then \
		echo '//nolint directives found, surface ignores in .golangci.yml instead' >&2; \
		exit 1; \
	fi

.PHONY: golangcilint
golangcilint: $(GOLANGCI_LINT)
	golangci-lint run --timeout $(GOLANGCILINTTIMEOUT)

.PHONY: postlint
postlint::

.PHONY: lint
lint:
	@$(MAKE) checknodiffgenerated
	@$(MAKE) checknonolint golangcilint postlint

.PHONY: prebuild
prebuild::

.PHONY: build
build: prebuild
	go build ./...

.PHONY: pretest
pretest::

.PHONY: test
test: pretest installtest
	go test $(GO_TEST_FLAGS) $(GOPKGS)

.PHONY: shorttest
shorttest: pretest installtest
	go test -test.short $(GO_TEST_FLAGS) $(GOPKGS)

.PHONY: deppkgs
deppkgs:
	@go list -f '{{join .Deps "\n"}}' $(GOPKGS) | xargs go list -f '{{if not .Standard}}{{.ImportPath}}{{end}}'

.PHONY: coverpkgs
coverpkgs:
	@go list $(GOPKGS) | grep -v \/gen\/ | tr '\n' ',' | sed "s/,$$//"

.PHONY: cover
cover: pretest installtest
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

.PHONY: installtest
installtest::

define gotestbinfunc
.PHONY: installtest$(notdir $(1))
installtest$(notdir $(1)):
	go install ./$(1)

installtest:: installtest$(notdir $(1))
endef

$(foreach gobin,$(sort $(GO_TEST_BINS)),$(eval $(call gotestbinfunc,$(gobin))))
$(foreach gobin,$(sort $(GO_TEST_BINS)),$(eval FILE_IGNORES := $(FILE_IGNORES) $(gobin)/$(notdir $(gobin))))

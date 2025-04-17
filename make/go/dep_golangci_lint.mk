# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

ifeq ($(UNAME_OS),Darwin)
GOLANGCI_LINT_OS := darwin
else ifeq ($(UNAME_OS),Linux)
GOLANGCI_LINT_OS := linux
endif

ifeq ($(UNAME_ARCH),x86_64)
GOLANGCI_LINT_ARCH := amd64
else ifeq ($(UNAME_ARCH),arm64)
GOLANGCI_LINT_ARCH := arm64
else ifeq ($(UNAME_ARCH),aarch64)
GOLANGCI_LINT_ARCH := arm64
else
GOLANGCI_LINT_ARCH := $(UNAME_ARCH)
endif

# Settable
# https://github.com/golangci/golangci-lint/releases 20250325 checked 20250401
# Contrast golangci-lint configuration with the one in https://github.com/connectrpc/connect-go/blob/main/.golangci.yml when upgrading
GOLANGCI_LINT_VERSION ?= v2.0.2

GOLANGCI_LINT := $(CACHE_VERSIONS)/golangci-lint/$(GOLANGCI_LINT_VERSION)
$(GOLANGCI_LINT):
	@rm -f $(CACHE_BIN)/golangci-lint
	@mkdir -p $(CACHE_BIN)
	$(eval GOLANGCI_LINT_TMP := $(shell mktemp -d))
	curl -fsSL -o $(GOLANGCI_LINT_TMP)/golangci-lint.tar.gz \
		https://github.com/golangci/golangci-lint/releases/download/$(GOLANGCI_LINT_VERSION)/golangci-lint-$(subst v,,$(GOLANGCI_LINT_VERSION))-$(GOLANGCI_LINT_OS)-$(GOLANGCI_LINT_ARCH).tar.gz
	cd $(GOLANGCI_LINT_TMP); tar zxf $(GOLANGCI_LINT_TMP)/golangci-lint.tar.gz --strip-components 1 && mv golangci-lint $(CACHE_BIN)/golangci-lint
	@rm -rf $(GOLANGCI_LINT_TMP)
	@rm -rf $(dir $(GOLANGCI_LINT))
	@mkdir -p $(dir $(GOLANGCI_LINT))
	@touch $(GOLANGCI_LINT)

dockerdeps:: $(GOLANGCI_LINT)

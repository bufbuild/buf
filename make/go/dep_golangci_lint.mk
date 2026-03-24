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
# https://github.com/golangci/golangci-lint/releases 20260322 checked 20260323
GOLANGCI_LINT_VERSION ?= v2.11.4

GOLANGCI_LINT := $(CACHE_BIN)/golangci-lint

$(CACHE_VERSIONS)/golangci-lint/golangci-lint-$(GOLANGCI_LINT_VERSION):
	@rm -f $(GOLANGCI_LINT)
	@rm -rf $(dir $@)
	@mkdir -p $(dir $@)
	$(eval GOLANGCI_LINT_TMP := $(shell mktemp -d))
	curl -fsSL -o $(GOLANGCI_LINT_TMP)/golangci-lint.tar.gz \
		https://github.com/golangci/golangci-lint/releases/download/$(GOLANGCI_LINT_VERSION)/golangci-lint-$(subst v,,$(GOLANGCI_LINT_VERSION))-$(GOLANGCI_LINT_OS)-$(GOLANGCI_LINT_ARCH).tar.gz
	cd $(GOLANGCI_LINT_TMP); tar zxf golangci-lint.tar.gz --strip-components 1 && mv golangci-lint $@
	@rm -rf $(GOLANGCI_LINT_TMP)
	@chmod +x $@
	@test -x $@
	@touch $@

$(GOLANGCI_LINT): $(CACHE_VERSIONS)/golangci-lint/golangci-lint-$(GOLANGCI_LINT_VERSION)
	@mkdir -p $(dir $@)
	@ln -sf $< $@

dockerdeps:: $(GOLANGCI_LINT)

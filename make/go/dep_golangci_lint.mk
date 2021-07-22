# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/golangci/golangci-lint/releases 20210619 checked 20210716
# Check for new linters and add to .golangci.yml (even if commented out) when upgrading
GOLANGCI_LINT_VERSION ?= v1.41.1

GOLANGCI_LINT := $(CACHE_VERSIONS)/golangci-lint/$(GOLANGCI_LINT_VERSION)
$(GOLANGCI_LINT):
	@rm -f $(CACHE_BIN)/golangci-lint
	$(eval GOLANGCI_LINT_TMP := $(shell mktemp -d))
	cd $(GOLANGCI_LINT_TMP); GOBIN=$(CACHE_BIN) go get github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@rm -rf $(GOLANGCI_LINT_TMP)
	@rm -rf $(dir $(GOLANGCI_LINT))
	@mkdir -p $(dir $(GOLANGCI_LINT))
	@touch $(GOLANGCI_LINT)

dockerdeps:: $(GOLANGCI_LINT)

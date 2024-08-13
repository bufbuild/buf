# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/golangci/golangci-lint/releases 20240813 checked 20240813
# TODO: Update to a release version instead of a commit on master when a version >1.59.1 is released
# Contrast golangci-lint configuration with the one in https://github.com/connectrpc/connect-go/blob/main/.golangci.yml when upgrading
GOLANGCI_LINT_VERSION ?= 1147824c61441fb1a928927ca095aa3d0f208459

GOLANGCI_LINT := $(CACHE_VERSIONS)/golangci-lint/$(GOLANGCI_LINT_VERSION)
$(GOLANGCI_LINT):
	@rm -f $(CACHE_BIN)/golangci-lint
	GOBIN=$(CACHE_BIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@rm -rf $(dir $(GOLANGCI_LINT))
	@mkdir -p $(dir $(GOLANGCI_LINT))
	@touch $(GOLANGCI_LINT)

dockerdeps:: $(GOLANGCI_LINT)

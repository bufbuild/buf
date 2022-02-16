# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/bufbuild/buf/releases
BUFSTYLE_VERSION ?= v1.0.0-rc12

BUFSTYLE := $(CACHE_VERSIONS)/bufstyle/$(BUFSTYLE_VERSION)
$(BUFSTYLE):
	@rm -f $(CACHE_BIN)/bufstyle
	GOBIN=$(CACHE_BIN) go install github.com/bufbuild/buf/private/bufpkg/bufstyle/cmd/bufstyle@$(BUFSTYLE_VERSION)
	@rm -rf $(dir $(BUFSTYLE))
	@mkdir -p $(dir $(BUFSTYLE))
	@touch $(BUFSTYLE)

dockerdeps:: $(BUFSTYLE)

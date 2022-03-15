# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/bufbuild/buf/releases
LICENSE_HEADER_VERSION ?= v1.0.0

LICENSE_HEADER := $(CACHE_VERSIONS)/license-header/$(LICENSE_HEADER_VERSION)
$(LICENSE_HEADER):
	@rm -f $(CACHE_BIN)/license-header
	GOBIN=$(CACHE_BIN) go install github.com/bufbuild/buf/private/pkg/licenseheader/cmd/license-header@$(LICENSE_HEADER_VERSION)
	@rm -rf $(dir $(LICENSE_HEADER))
	@mkdir -p $(dir $(LICENSE_HEADER))
	@touch $(LICENSE_HEADER)

dockerdeps:: $(LICENSE_HEADER)

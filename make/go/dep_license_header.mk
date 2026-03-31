# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/dep_buf.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)
$(call _assert_var,BUF_VERSION)

# Settable
# https://github.com/bufbuild/buf/releases
LICENSE_HEADER_VERSION ?= $(BUF_VERSION)

LICENSE_HEADER := $(CACHE_BIN)/license-header

$(CACHE_VERSIONS)/license-header/license-header-$(LICENSE_HEADER_VERSION):
	@rm -f $(LICENSE_HEADER)
	@rm -rf $(dir $@)
	@mkdir -p $(dir $@)
	GOBIN=$(dir $@) go install github.com/bufbuild/buf/private/pkg/licenseheader/cmd/license-header@$(LICENSE_HEADER_VERSION)
	@mv $(dir $@)/license-header $@
	@test -x $@
	@touch $@

$(LICENSE_HEADER): $(CACHE_VERSIONS)/license-header/license-header-$(LICENSE_HEADER_VERSION)
	@mkdir -p $(dir $@)
	@ln -sf $< $@

dockerdeps:: $(LICENSE_HEADER)

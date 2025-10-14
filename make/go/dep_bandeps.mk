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
BANDEPS_VERSION ?= $(BUF_VERSION)

BANDEPS := $(CACHE_VERSIONS)/bandeps/$(BANDEPS_VERSION)
$(BANDEPS):
	@rm -f $(CACHE_BIN)/bandeps
	GOBIN=$(CACHE_BIN) go install github.com/bufbuild/buf/private/pkg/bandeps/cmd/bandeps@$(BANDEPS_VERSION)
	@rm -rf $(dir $(BANDEPS))
	@mkdir -p $(dir $(BANDEPS))
	@touch $(BANDEPS)

dockerdeps:: $(BANDEPS)

# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/dep_buf.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)
$(call _assert_var,BUF_VERSION)

# Settable
# https://github.com/bufbuild/bufstyle-go/releases
BUFSTYLE_VERSION ?= v0.1.0

BUFSTYLE := $(CACHE_VERSIONS)/bufstyle/$(BUFSTYLE_VERSION)
$(BUFSTYLE):
	@rm -f $(CACHE_BIN)/bufstyle
	GOBIN=$(CACHE_BIN) go install buf.build/go/bufstyle@$(BUFSTYLE_VERSION)
	@rm -rf $(dir $(BUFSTYLE))
	@mkdir -p $(dir $(BUFSTYLE))
	@touch $(BUFSTYLE)

dockerdeps:: $(BUFSTYLE)

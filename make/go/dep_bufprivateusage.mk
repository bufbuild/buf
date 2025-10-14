# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/dep_buf.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)
$(call _assert_var,BUF_VERSION)

# Settable
# https://github.com/bufbuild/bufprivateusage-go/releases
BUFPRIVATEUSAGE_VERSION ?= v0.1.0

BUFPRIVATEUSAGE := $(CACHE_VERSIONS)/bufprivateusage/$(BUFPRIVATEUSAGE_VERSION)
$(BUFPRIVATEUSAGE):
	@rm -f $(CACHE_BIN)/bufprivateusage
	GOBIN=$(CACHE_BIN) go install buf.build/go/bufprivateusage/cmd/bufprivateusage@$(BUFPRIVATEUSAGE_VERSION)
	@rm -rf $(dir $(BUFPRIVATEUSAGE))
	@mkdir -p $(dir $(BUFPRIVATEUSAGE))
	@touch $(BUFPRIVATEUSAGE)

dockerdeps:: $(BUFPRIVATEUSAGE)

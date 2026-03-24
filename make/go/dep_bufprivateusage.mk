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

BUFPRIVATEUSAGE := $(CACHE_BIN)/bufprivateusage

$(CACHE_VERSIONS)/bufprivateusage/bufprivateusage-$(BUFPRIVATEUSAGE_VERSION):
	@rm -f $(BUFPRIVATEUSAGE)
	@rm -rf $(dir $@)
	@mkdir -p $(dir $@)
	GOBIN=$(dir $@) go install buf.build/go/bufprivateusage/cmd/bufprivateusage@$(BUFPRIVATEUSAGE_VERSION)
	@mv $(dir $@)/bufprivateusage $@
	@test -x $@
	@touch $@

$(BUFPRIVATEUSAGE): $(CACHE_VERSIONS)/bufprivateusage/bufprivateusage-$(BUFPRIVATEUSAGE_VERSION)
	@mkdir -p $(dir $@)
	@ln -sf $< $@

dockerdeps:: $(BUFPRIVATEUSAGE)

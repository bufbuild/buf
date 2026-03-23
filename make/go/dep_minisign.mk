# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/aead/minisign 20240519 checked 20240524
MINISIGN_VERSION ?= v0.3.0

MINISIGN := $(CACHE_BIN)/minisign

$(CACHE_VERSIONS)/minisign/minisign-$(MINISIGN_VERSION):
	@rm -f $(MINISIGN)
	@rm -rf $(dir $@)
	@mkdir -p $(dir $@)
	GOBIN=$(dir $@) go install aead.dev/minisign/cmd/minisign@$(MINISIGN_VERSION)
	@mv $(dir $@)/minisign $@
	@test -x $@
	@touch $@

$(MINISIGN): $(CACHE_VERSIONS)/minisign/minisign-$(MINISIGN_VERSION)
	@mkdir -p $(dir $@)
	@ln -sf $< $@

deps:: $(MINISIGN)

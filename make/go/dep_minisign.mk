# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/aead/minisign 20240519 checked 20240524
MINISIGN_VERSION ?= v0.3.0

MINISIGN := $(CACHE_VERSIONS)/MINISIGN/$(MINISIGN_VERSION)
$(MINISIGN):
	@rm -f $(CACHE_BIN)/minisign
	GOBIN=$(CACHE_BIN) go install aead.dev/minisign/cmd/minisign@$(MINISIGN_VERSION)
	@rm -rf $(dir $(MINISIGN))
	@mkdir -p $(dir $(MINISIGN))
	@touch $(MINISIGN)

deps:: $(MINISIGN)

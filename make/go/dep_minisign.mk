# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/aead/minisign/commit/4b0a1d5cec55046cced3e84526660c7820b8f58c 20211108 checked 20211112
# Contains fix for https://github.com/aead/minisign/issues/11
MINISIGN_VERSION ?= 4b0a1d5cec55046cced3e84526660c7820b8f58c

MINISIGN := $(CACHE_VERSIONS)/MINISIGN/$(MINISIGN_VERSION)
$(MINISIGN):
	@rm -f $(CACHE_BIN)/minisign
	GOBIN=$(CACHE_BIN) go install aead.dev/minisign/cmd/minisign@$(MINISIGN_VERSION)
	@rm -rf $(dir $(MINISIGN))
	@mkdir -p $(dir $(MINISIGN))
	@touch $(MINISIGN)

deps:: $(MINISIGN)

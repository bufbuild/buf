# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/bufbuild/buf/releases
BUF_VERSION ?= v0.46.0

BUF := $(CACHE_VERSIONS)/buf/$(BUF_VERSION)
$(BUF):
	@rm -f $(CACHE_BIN)/buf
	$(eval BUF_TMP := $(shell mktemp -d))
	cd $(BUF_TMP); GOBIN=$(CACHE_BIN) go get github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
	@rm -rf $(BUF_TMP)
	@rm -rf $(dir $(BUF))
	@mkdir -p $(dir $(BUF))
	@touch $(BUF)

dockerdeps:: $(BUF)

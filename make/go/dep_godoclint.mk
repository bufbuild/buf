# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/dep_buf.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)
$(call _assert_var,BUF_VERSION)

# Settable
#
# Based off of dev branch.
# https://github.com/bufbuild/godoc-lint/commits/dev
GODOCLINT_VERSION ?= 5405ef06cd81f5b3115c6e52744f6cd9a8fa84f1a

GODOCLINT := $(CACHE_VERSIONS)/godoclint/$(GODOCLINT_VERSION)
$(GODOCLINT):
	@rm -f $(CACHE_BIN)/godoclint
	$(eval GODOCLINT_TMP := $(shell mktemp -d))
	cd $(GODOCLINT_TMP); \
		git clone https://github.com/bufbuild/godoc-lint && \
		cd ./godoc-lint && \
		git checkout $(GODOCLINT_VERSION) && \
		GOBIN=$(CACHE_BIN) go install ./cmd/godoclint
	@rm -rf $(GODOCLINT_TMP)
	@rm -rf $(dir $(GODOCLINT))
	@mkdir -p $(dir $(GODOCLINT))
	@touch $(GODOCLINT)

dockerdeps:: $(GODOCLINT)

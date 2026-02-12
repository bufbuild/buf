# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/dep_buf.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)
$(call _assert_var,BUF_VERSION)

# We want to ensure we rebuild godoclint every time we require a new Go minor version.
# Otherwise, the cached version may not support the latest language features.
GODOCLINT_GO_VERSION := $(shell go list -m -f '{{.GoVersion}}' | cut -d'.' -f1-2)

# Settable
#
# Based off of dev branch.
# https://github.com/bufbuild/godoc-lint/commits/dev
GODOCLINT_VERSION ?= 26c7b506fc2bf37a67fc2b42a3d9825c7ade2068

GODOCLINT := $(CACHE_VERSIONS)/godoclint/$(GODOCLINT_VERSION)-go$(GODOCLINT_GO_VERSION)
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

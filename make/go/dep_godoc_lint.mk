# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/dep_buf.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)
$(call _assert_var,BUF_VERSION)

# Settable
# https://github.com/bufbuild/godoc-lint/commits
GODOC_LINT_VERSION ?= 130759d40c7f0bc863b199cbfa8de19d884cebd1

GODOC_LINT := $(CACHE_VERSIONS)/godoc_lint/$(GODOC_LINT_VERSION)
$(GODOC_LINT):
	@rm -f $(CACHE_BIN)/godoclint
	$(eval GODOC_LINT_TMP := $(shell mktemp -d))
	cd $(GODOC_LINT_TMP); \
		git clone https://github.com/bufbuild/godoc-lint && \
		cd ./godoc-lint && \
		git checkout $(GODOC_LINT_VERSION) && \
		GOBIN=$(CACHE_BIN) go install ./cmd/godoclint
	@rm -rf $(GODOC_LINT_TMP)
	@rm -rf $(dir $(GODOC_LINT))
	@mkdir -p $(dir $(GODOC_LINT))
	@touch $(GODOC_LINT)

dockerdeps:: $(GODOC_LINT)

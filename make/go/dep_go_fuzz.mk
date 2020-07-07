# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,GOBIN)

# Settable
# https://github.com/dvyukov/go-fuzz/commits/master 20200318 checked 20200707
GO_FUZZ_VERSION ?= be3528f3a81351d8a438aed216130e1e7da39f7c

GO_FUZZ := $(CACHE_VERSIONS)/go-fuzz/$(GO_FUZZ_VERSION)
$(GO_FUZZ):
	@rm -f $(GOBIN)/go-fuzz $(GOBIN)/go-fuzz-build
	$(eval GO_FUZZ_TMP := $(shell mktemp -d))
	cd $(GO_FUZZ_TMP); go get \
		github.com/dvyukov/go-fuzz/go-fuzz@$(GO_FUZZ_VERSION) \
		github.com/dvyukov/go-fuzz/go-fuzz-build@$(GO_FUZZ_VERSION)
	@rm -rf $(GO_FUZZ_TMP)
	@rm -rf $(dir $(GO_FUZZ))
	@mkdir -p $(dir $(GO_FUZZ))
	@touch $(GO_FUZZ)

dockerdeps:: $(GO_FUZZ)

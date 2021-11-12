# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/dvyukov/go-fuzz/commits/master 20210914 checked 20211112
GO_FUZZ_VERSION ?= 4980593459a186bd2a389fe4557a260cce742594

GO_FUZZ := $(CACHE_VERSIONS)/go-fuzz/$(GO_FUZZ_VERSION)
$(GO_FUZZ):
	@rm -f $(CACHE_BIN)/go-fuzz $(CACHE_BIN)/go-fuzz-build
	GOBIN=$(CACHE_BIN) go install \
		github.com/dvyukov/go-fuzz/go-fuzz@$(GO_FUZZ_VERSION) \
		github.com/dvyukov/go-fuzz/go-fuzz-build@$(GO_FUZZ_VERSION)
	@rm -rf $(dir $(GO_FUZZ))
	@mkdir -p $(dir $(GO_FUZZ))
	@touch $(GO_FUZZ)

dockerdeps:: $(GO_FUZZ)

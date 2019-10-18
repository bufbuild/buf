ifndef CACHE_VERSIONS
$(error CACHE_VERSIONS is not set)
endif
ifndef GOBIN
$(error GOBIN is not set)
endif
ifndef STATICCHECK_VERSION
$(error STATICCHECK_VERSION is not set)
endif

STATICCHECK := $(CACHE_VERSIONS)/staticcheck/$(STATICCHECK_VERSION)
$(STATICCHECK):
	@rm -f $(GOBIN)/staticcheck
	$(eval STATICCHECK_TMP := $(shell mktemp -d))
	cd $(STATICCHECK_TMP); go get honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	@rm -rf $(STATICCHECK_TMP)
	@rm -rf $(dir $(STATICCHECK))
	@mkdir -p $(dir $(STATICCHECK))
	@touch $(STATICCHECK)

deps:: $(STATICCHECK)

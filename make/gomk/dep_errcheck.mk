ifndef CACHE_VERSIONS
$(error CACHE_VERSIONS is not set)
endif
ifndef GOBIN
$(error GOBIN is not set)
endif
ifndef ERRCHECK_VERSION
$(error ERRCHECK_VERSION is not set)
endif

ERRCHECK := $(CACHE_VERSIONS)/errcheck/$(ERRCHECK_VERSION)
$(ERRCHECK):
	@rm -f $(GOBIN)/errcheck
	$(eval ERRCHECK_TMP := $(shell mktemp -d))
	cd $(ERRCHECK_TMP); go get github.com/kisielk/errcheck@$(ERRCHECK_VERSION)
	@rm -rf $(ERRCHECK_TMP)
	@rm -rf $(dir $(ERRCHECK))
	@mkdir -p $(dir $(ERRCHECK))
	@touch $(ERRCHECK)

dockerdeps:: $(ERRCHECK)

ifndef CACHE_VERSIONS
$(error CACHE_VERSIONS is not set)
endif
ifndef GOBIN
$(error GOBIN is not set)
endif
ifndef GOLINT_VERSION
$(error GOLINT_VERSION is not set)
endif

GOLINT := $(CACHE_VERSIONS)/golint/$(GOLINT_VERSION)
$(GOLINT):
	@rm -f $(GOBIN)/golint
	$(eval GOLINT_TMP := $(shell mktemp -d))
	cd $(GOLINT_TMP); go get golang.org/x/lint/golint@$(GOLINT_VERSION)
	@rm -rf $(GOLINT_TMP)
	@rm -rf $(dir $(GOLINT))
	@mkdir -p $(dir $(GOLINT))
	@touch $(GOLINT)

dockerdeps:: $(GOLINT)

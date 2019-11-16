ifndef CACHE_VERSIONS
$(error CACHE_VERSIONS is not set)
endif
ifndef GOBIN
$(error GOBIN is not set)
endif
ifndef INEFFASSIGN_VERSION
$(error INEFFASSIGN_VERSION is not set)
endif

INEFFASSIGN := $(CACHE_VERSIONS)/ineffassign/$(INEFFASSIGN_VERSION)
$(INEFFASSIGN):
	@rm -f $(GOBIN)/ineffassign
	$(eval INEFFASSIGN_TMP := $(shell mktemp -d))
	cd $(INEFFASSIGN_TMP); go get github.com/gordonklaus/ineffassign@$(INEFFASSIGN_VERSION)
	@rm -rf $(INEFFASSIGN_TMP)
	@rm -rf $(dir $(INEFFASSIGN))
	@mkdir -p $(dir $(INEFFASSIGN))
	@touch $(INEFFASSIGN)

dockerdeps:: $(INEFFASSIGN)

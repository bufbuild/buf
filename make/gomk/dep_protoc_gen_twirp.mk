ifndef CACHE_VERSIONS
$(error CACHE_VERSIONS is not set)
endif
ifndef GOBIN
$(error GOBIN is not set)
endif
ifndef PROTOC_GEN_TWIRP_VERSION
$(error PROTOC_GEN_TWIRP_VERSION is not set)
endif

PROTOC_GEN_TWIRP := $(CACHE_VERSIONS)/protoc-gen-twirp/$(PROTOC_GEN_TWIRP_VERSION)
$(PROTOC_GEN_TWIRP):
	@rm -f $(GOBIN)/protoc-gen-twirp
	$(eval PROTOC_GEN_TWIRP_TMP := $(shell mktemp -d))
	cd $(PROTOC_GEN_TWIRP_TMP); go get github.com/twitchtv/twirp/protoc-gen-twirp@$(PROTOC_GEN_TWIRP_VERSION)
	@rm -rf $(PROTOC_GEN_TWIRP_TMP)
	@rm -rf $(dir $(PROTOC_GEN_TWIRP))
	@mkdir -p $(dir $(PROTOC_GEN_TWIRP))
	@touch $(PROTOC_GEN_TWIRP)

dockerdeps:: $(PROTOC_GEN_TWIRP)

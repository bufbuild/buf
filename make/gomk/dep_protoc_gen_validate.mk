ifndef CACHE_VERSIONS
$(error CACHE_VERSIONS is not set)
endif
ifndef CACHE_INCLUDE
$(error CACHE_INCLUDE is not set)
endif
ifndef GOBIN
$(error GOBIN is not set)
endif
ifndef PROTOC_GEN_VALIDATE_VERSION
$(error PROTOC_GEN_VALIDATE_VERSION is not set)
endif

PROTOC_GEN_VALIDATE := $(CACHE_VERSIONS)/protoc-gen-validate/$(PROTOC_GEN_VALIDATE_VERSION)
$(PROTOC_GEN_VALIDATE):
	@rm -f $(GOBIN)/protoc-gen-validate
	@rm -rf $(CACHE_INCLUDE)/validate
	@mkdir -p $(CACHE_INCLUDE)/validate
	$(eval PROTOC_GEN_VALIDATE_TMP := $(shell mktemp -d))
	cd $(PROTOC_GEN_VALIDATE_TMP); go get github.com/envoyproxy/protoc-gen-validate@$(PROTOC_GEN_VALIDATE_VERSION)
	curl -sSL \
		https://raw.githubusercontent.com/envoyproxy/protoc-gen-validate/$(PROTOC_GEN_VALIDATE_VERSION)/validate/validate.proto \
		-o $(CACHE_INCLUDE)/validate/validate.proto
	@rm -rf $(PROTOC_GEN_VALIDATE_TMP)
	@rm -rf $(dir $(PROTOC_GEN_VALIDATE))
	@mkdir -p $(dir $(PROTOC_GEN_VALIDATE))
	@touch $(PROTOC_GEN_VALIDATE)

dockerdeps:: $(PROTOC_GEN_VALIDATE)

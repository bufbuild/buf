# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/twitchtv/twirp/releases 20210616 checked 20210712
PROTOC_GEN_TWIRP_VERSION ?= v8.1.0

GO_GET_PKGS := $(GO_GET_PKGS) github.com/twitchtv/twirp@$(PROTOC_GEN_TWIRP_VERSION)

PROTOC_GEN_TWIRP := $(CACHE_VERSIONS)/protoc-gen-twirp/$(PROTOC_GEN_TWIRP_VERSION)
$(PROTOC_GEN_TWIRP):
	@rm -f $(CACHE_BIN)/protoc-gen-twirp
	GOBIN=$(CACHE_BIN) go install github.com/twitchtv/twirp/protoc-gen-twirp@$(PROTOC_GEN_TWIRP_VERSION)
	@rm -rf $(dir $(PROTOC_GEN_TWIRP))
	@mkdir -p $(dir $(PROTOC_GEN_TWIRP))
	@touch $(PROTOC_GEN_TWIRP)

dockerdeps:: $(PROTOC_GEN_TWIRP)

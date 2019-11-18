# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,GOBIN)

# Settable
# https://github.com/twitchtv/twirp/releases 20190726
PROTOC_GEN_TWIRP_VERSION ?= v5.8.0

GO_GET_PKGS := $(GO_GET_PKGS) github.com/twitchtv/twirp@$(PROTOC_GEN_TWIRP_VERSION)

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

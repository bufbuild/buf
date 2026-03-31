# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/connectrpc/connect-go 20251006 checked 20251013
CONNECT_VERSION ?= v1.19.1

GO_GET_PKGS := $(GO_GET_PKGS) \
	connectrpc.com/connect@$(CONNECT_VERSION)

PROTOC_GEN_CONNECT_GO := $(CACHE_BIN)/protoc-gen-connect-go

$(CACHE_VERSIONS)/connect-go/protoc-gen-connect-go-$(CONNECT_VERSION):
	@rm -f $(PROTOC_GEN_CONNECT_GO)
	@rm -rf $(dir $@)
	@mkdir -p $(dir $@)
	GOBIN=$(dir $@) go install connectrpc.com/connect/cmd/protoc-gen-connect-go@$(CONNECT_VERSION)
	@mv $(dir $@)/protoc-gen-connect-go $@
	@test -x $@
	@touch $@

$(PROTOC_GEN_CONNECT_GO): $(CACHE_VERSIONS)/connect-go/protoc-gen-connect-go-$(CONNECT_VERSION)
	@mkdir -p $(dir $@)
	@ln -sf $< $@

dockerdeps:: $(PROTOC_GEN_CONNECT_GO)

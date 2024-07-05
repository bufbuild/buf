# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

PLUGINRPC_GO_VERSION ?= c762588190c665dc853e0ab190888f251d717e84

GO_GET_PKGS := $(GO_GET_PKGS) \
	github.com/bufbuild/pluginrpc-go@$(PLUGINRPC_GO_VERSION)

PROTOC_GEN_PLUGINRPC_GO := $(CACHE_VERSIONS)/connect-go/$(PLUGINRPC_GO_VERSION)
$(PROTOC_GEN_PLUGINRPC_GO):
	@rm -f $(CACHE_BIN)/protoc-gen-pluginrpc-go
	GOBIN=$(CACHE_BIN) go install github.com/bufbuild/pluginrpc-go/cmd/protoc-gen-pluginrpc-go@$(PLUGINRPC_GO_VERSION)
	@rm -rf $(dir $(PROTOC_GEN_PLUGINRPC_GO))
	@mkdir -p $(dir $(PROTOC_GEN_PLUGINRPC_GO))
	@touch $(PROTOC_GEN_PLUGINRPC_GO)

dockerdeps:: $(PROTOC_GEN_PLUGINRPC_GO)

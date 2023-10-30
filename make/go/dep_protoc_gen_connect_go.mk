# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/connectrpc/connect-go 20231024 checked 20231027
CONNECT_VERSION ?= v1.12.0

GO_GET_PKGS := $(GO_GET_PKGS) \
	connectrpc.com/connect@$(CONNECT_VERSION)

PROTOC_GEN_CONNECT_GO := $(CACHE_VERSIONS)/connect-go/$(CONNECT_VERSION)
$(PROTOC_GEN_CONNECT_GO):
	@rm -f $(CACHE_BIN)/protoc-gen-connect-go
	GOBIN=$(CACHE_BIN) go install connectrpc.com/connect/cmd/protoc-gen-connect-go@$(CONNECT_VERSION)
	@rm -rf $(dir $(PROTOC_GEN_CONNECT_GO))
	@mkdir -p $(dir $(PROTOC_GEN_CONNECT_GO))
	@touch $(PROTOC_GEN_CONNECT_GO)

dockerdeps:: $(PROTOC_GEN_CONNECT_GO)

# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/bufbuild/connect-go 20221018 checked 20221024
CONNECT_VERSION ?= v1.1.0

GO_GET_PKGS := $(GO_GET_PKGS) \
	github.com/bufbuild/connect-go@$(CONNECT_VERSION)

PROTOC_GEN_CONNECT_GO := $(CACHE_VERSIONS)/connect-go/$(CONNECT_VERSION)
$(PROTOC_GEN_CONNECT_GO):
	@rm -f $(CACHE_BIN)/connect-go
	GOBIN=$(CACHE_BIN) go install github.com/bufbuild/connect-go/cmd/protoc-gen-connect-go@$(CONNECT_VERSION)
	@rm -rf $(dir $(PROTOC_GEN_CONNECT_GO))
	@mkdir -p $(dir $(PROTOC_GEN_CONNECT_GO))
	@touch $(PROTOC_GEN_CONNECT_GO)

dockerdeps:: $(PROTOC_GEN_CONNECT_GO)

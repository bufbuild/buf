# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/bufbuild/connect-go 20220514 checked 20220517
CONNECT_VERSION ?= 316a3a0a4a4258b17ae9d5de86dc4327c329ca08

PROTOC_GEN_CONNECT_GO := $(CACHE_VERSIONS)/connect-go/$(CONNECT_VERSION)
$(PROTOC_GEN_CONNECT_GO):
	@rm -f $(CACHE_BIN)/connect-go
	GOBIN=$(CACHE_BIN) go install github.com/bufbuild/connect-go/cmd/protoc-gen-connect-go@$(CONNECT_VERSION)
	@rm -rf $(dir $(PROTOC_GEN_CONNECT_GO))
	@mkdir -p $(dir $(PROTOC_GEN_CONNECT_GO))
	@touch $(PROTOC_GEN_CONNECT_GO)

dockerdeps:: $(PROTOC_GEN_CONNECT_GO)

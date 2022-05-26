# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/bufbuild/connect-go 20220525 checked 20220525
CONNECT_VERSION ?= b79148bf7e449283e5b506e945ddf2b3d56e536c

PROTOC_GEN_CONNECT_GO := $(CACHE_VERSIONS)/connect-go/$(CONNECT_VERSION)
$(PROTOC_GEN_CONNECT_GO):
	@rm -f $(CACHE_BIN)/connect-go
	GOBIN=$(CACHE_BIN) go install github.com/bufbuild/connect-go/cmd/protoc-gen-connect-go@$(CONNECT_VERSION)
	@rm -rf $(dir $(PROTOC_GEN_CONNECT_GO))
	@mkdir -p $(dir $(PROTOC_GEN_CONNECT_GO))
	@touch $(PROTOC_GEN_CONNECT_GO)

dockerdeps:: $(PROTOC_GEN_CONNECT_GO)

# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/grpc/grpc-go/commits/master 20201218 checked 20201222
PROTOC_GEN_GO_GRPC_VERSION ?= 666aea1fb34c9ceeab997f7a9c37c2cb51deb6e5

GO_GET_PKGS := $(GO_GET_PKGS) google.golang.org/grpc@$(PROTOC_GEN_GO_GRPC_VERSION)

PROTOC_GEN_GO_GRPC := $(CACHE_VERSIONS)/protoc-gen-go-grpc/$(PROTOC_GEN_GO_GRPC_VERSION)
$(PROTOC_GEN_GO_GRPC):
	@rm -f $(CACHE_BIN)/protoc-gen-go-grpc
	$(eval PROTOC_GEN_GO_GRPC_TMP := $(shell mktemp -d))
	cd $(PROTOC_GEN_GO_GRPC_TMP); GOBIN=$(CACHE_BIN) go get \
		google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)
	@rm -rf $(PROTOC_GEN_GO_GRPC_TMP)
	@rm -rf $(dir $(PROTOC_GEN_GO_GRPC))
	@mkdir -p $(dir $(PROTOC_GEN_GO_GRPC))
	@touch $(PROTOC_GEN_GO_GRPC)

dockerdeps:: $(PROTOC_GEN_GO_GRPC)

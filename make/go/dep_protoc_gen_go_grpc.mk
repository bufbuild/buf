# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/grpc/grpc-go/releases 20220513 checked 20220517
# You MUST use a commit instead of the release number due to the multi-module
# setup of grpc-go and protoc-gen-go-grpc.
PROTOC_GEN_GO_GRPC_VERSION ?= 46da11bc8bf12ea2b86d0783cd92e098ba0ccc99

GO_GET_PKGS := $(GO_GET_PKGS) google.golang.org/grpc@$(PROTOC_GEN_GO_GRPC_VERSION)

PROTOC_GEN_GO_GRPC := $(CACHE_VERSIONS)/protoc-gen-go-grpc/$(PROTOC_GEN_GO_GRPC_VERSION)
$(PROTOC_GEN_GO_GRPC):
	@rm -f $(CACHE_BIN)/protoc-gen-go-grpc
	GOBIN=$(CACHE_BIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)
	@rm -rf $(dir $(PROTOC_GEN_GO_GRPC))
	@mkdir -p $(dir $(PROTOC_GEN_GO_GRPC))
	@touch $(PROTOC_GEN_GO_GRPC)

dockerdeps:: $(PROTOC_GEN_GO_GRPC)

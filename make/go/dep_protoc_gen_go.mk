# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/protocolbuffers/protobuf-go/commits/master 20240201 checked 20240202
PROTOC_GEN_GO_VERSION ?= 82c6b3ad29a1a6116348452560e4fb822a943193

GO_GET_PKGS := $(GO_GET_PKGS) \
	google.golang.org/protobuf/proto@$(PROTOC_GEN_GO_VERSION)

PROTOC_GEN_GO := $(CACHE_VERSIONS)/protoc-gen-go/$(PROTOC_GEN_GO_VERSION)
$(PROTOC_GEN_GO):
	@rm -f $(CACHE_BIN)/protoc-gen-go
	GOBIN=$(CACHE_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	@rm -rf $(dir $(PROTOC_GEN_GO))
	@mkdir -p $(dir $(PROTOC_GEN_GO))
	@touch $(PROTOC_GEN_GO)

dockerdeps:: $(PROTOC_GEN_GO)

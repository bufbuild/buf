# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/protocolbuffers/protobuf-go/commits/master 20210302 checked 20210309
PROTOC_GEN_GO_VERSION ?= 839ce436895b59ff32f47ca10a6ae33f3f0603e2
# TODO: remove when https://github.com/protocolbuffers/protobuf-go/commit/839ce436895b59ff32f47ca10a6ae33f3f0603e2 is fixed
GOLANG_PROTOBUF_V1_VERSION ?= acacf8158c9a307051c92dc233966e8324facd45

GO_GET_PKGS := $(GO_GET_PKGS) \
	google.golang.org/protobuf/proto@$(PROTOC_GEN_GO_VERSION) \
	github.com/golang/protobuf/proto@$(GOLANG_PROTOBUF_V1_VERSION)

PROTOC_GEN_GO := $(CACHE_VERSIONS)/protoc-gen-go/$(PROTOC_GEN_GO_VERSION)
$(PROTOC_GEN_GO):
	@rm -f $(CACHE_BIN)/protoc-gen-go
	$(eval PROTOC_GEN_GO_TMP := $(shell mktemp -d))
	cd $(PROTOC_GEN_GO_TMP); GOBIN=$(CACHE_BIN) go get google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	@rm -rf $(PROTOC_GEN_GO_TMP)
	@rm -rf $(dir $(PROTOC_GEN_GO))
	@mkdir -p $(dir $(PROTOC_GEN_GO))
	@touch $(PROTOC_GEN_GO)

dockerdeps:: $(PROTOC_GEN_GO)

ifndef PROTO_PATH
$(error PROTO_PATH is not set)
endif
ifndef PROTOC_GEN_GO_OUT
$(error PROTOC_GEN_GO_OUT is not set)
endif

PROTOC_GEN_GO_PARAMETER ?=

.PHONY: protocgengoclean
protocgengoclean:
	rm -rf "$(PROTOC_GEN_GO_OUT)"

.PHONY: protocgengo
protocgengo: protocgengoclean $(PROTOC) $(PROTOC_GEN_GO)
	bash scripts/protoc_gen_go.bash "$(PROTO_PATH)" "$(PROTOC_GEN_GO_OUT)" "$(PROTOC_GEN_GO_PARAMETER)"

pregenerate:: protocgengo

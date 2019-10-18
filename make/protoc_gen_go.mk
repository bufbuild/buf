ifndef PROTO_PATH
$(error PROTO_PATH is not set)
endif
ifndef PROTOC_GEN_GO_OUT
$(error PROTOC_GEN_GO_OUT is not set)
endif

.PHONY: protocgengo
protocgengo: $(PROTOC) $(PROTOC_GEN_GO)
	bash scripts/protoc_gen_go.bash "$(PROTO_PATH)" "$(PROTOC_GEN_GO_OUT)"

pregenerate:: protocgengo

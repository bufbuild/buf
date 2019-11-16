ifndef GOMK_DIR
$(error GOMK_DIR is not set)
endif
ifndef PROTOC
$(error PROTOC is not set)
endif
ifndef PROTOC_GEN_GO
$(error PROTOC_GEN_GO is not set)
endif
ifndef PROTO_PATH
$(error PROTO_PATH is not set)
endif
ifndef PROTOC_GEN_GO_OUT
$(error PROTOC_GEN_GO_OUT is not set)
endif

PROTOC_GEN_GO_OPT ?=

.PHONY: protocgengoclean
protocgengoclean:
	rm -rf "$(PROTOC_GEN_GO_OUT)"

.PHONY: protocgengo
protocgengo: protocgengoclean $(PROTOC) $(PROTOC_GEN_GO)
	bash $(GOMK_DIR)/protoc_gen_plugin.bash \
		"--proto_path=$(PROTO_PATH)" \
		"--plugin_name=go" \
		"--plugin_out=$(PROTOC_GEN_GO_OUT)" \
		"--plugin_opt=$(PROTOC_GEN_GO_OPT)"

pregenerate:: protocgengo

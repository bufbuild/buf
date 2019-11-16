ifndef GOMK_DIR
$(error GOMK_DIR is not set)
endif
ifndef PROTOC
$(error PROTOC is not set)
endif
ifndef PROTOC_GEN_TWIRP
$(error PROTOC_GEN_TWIRP is not set)
endif
ifndef PROTO_PATH
$(error PROTO_PATH is not set)
endif
ifndef PROTOC_GEN_TWIRP_OUT
$(error PROTOC_GEN_TWIRP_OUT is not set)
endif

PROTOC_GEN_TWIRP_OPT ?=

.PHONY: protocgentwirp
protocgentwirp: protocgengoclean $(PROTOC) $(PROTOC_GEN_TWIRP)
	bash $(GOMK_DIR)/protoc_gen_plugin.bash \
		"--proto_path=$(PROTO_PATH)" \
		"--plugin_name=twirp" \
		"--plugin_out=$(PROTOC_GEN_TWIRP_OUT)" \
		"--plugin_opt=$(PROTOC_GEN_TWIRP_OPT)"

pregenerate:: protocgentwirp

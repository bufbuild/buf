ifndef GOMK_DIR
$(error GOMK_DIR is not set)
endif
ifndef CACHE_INCLUDE
$(error CACHE_INCLUDE is not set)
endif
ifndef PROTOC
$(error PROTOC is not set)
endif
ifndef PROTOC_GEN_VALIDATE
$(error PROTOC_GEN_VALIDATE is not set)
endif
ifndef PROTO_PATH
$(error PROTO_PATH is not set)
endif
ifndef PROTOC_GEN_VALIDATE_OUT
$(error PROTOC_GEN_VALIDATE_OUT is not set)
endif

# Not modifiable for now
PROTOC_GEN_VALIDATE_OPT := lang=go

.PHONY: protocgenvalidate
protocgenvalidate: protocgengoclean $(PROTOC) $(PROTOC_GEN_VALIDATE)
	bash $(GOMK_DIR)/protoc_gen_plugin.bash \
		"--include_path=$(CACHE_INCLUDE)" \
		"--proto_path=$(PROTO_PATH)" \
		"--plugin_name=validate" \
		"--plugin_out=$(PROTOC_GEN_VALIDATE_OUT)" \
		"--plugin_opt=$(PROTOC_GEN_VALIDATE_OPT)"

pregenerate:: protocgenvalidate

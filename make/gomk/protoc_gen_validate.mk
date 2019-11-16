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
PROTO_INCLUDE_PATHS := $(PROTO_INCLUDE_PATHS) third_party/proto

.PHONY: protocgenvalidate
protocgenvalidate: protocgengoclean $(PROTOC) $(PROTOC_GEN_VALIDATE)
	bash $(GOMK_DIR)/protoc_gen_plugin.bash \
		"--proto_path=$(PROTO_PATH)" \
		"--proto_include_path=$(CACHE_INCLUDE)" \
		$(patsubst %,--proto_include_path=%,$(PROTO_INCLUDE_PATHS)) \
		"--plugin_name=validate" \
		"--plugin_out=$(PROTOC_GEN_VALIDATE_OUT)" \
		"--plugin_opt=$(PROTOC_GEN_VALIDATE_OPT)"

pregenerate:: protocgenvalidate

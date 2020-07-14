# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/dep_protoc.mk)
$(call _conditional_include,$(MAKEGO)/dep_protoc_gen_go.mk)
# Must be set
$(call _assert_var,PROTO_PATH)
# Must be set
$(call _assert_var,PROTOC_GEN_GO_OUT)
$(call _assert_var,CACHE_INCLUDE)
$(call _assert_var,PROTOC)
$(call _assert_var,PROTOC_GEN_GO)

# Not modifiable for now
PROTOC_GEN_GO_OPT := paths=source_relative

EXTRA_MAKEGO_FILES := $(EXTRA_MAKEGO_FILES) scripts/protoc_gen_plugin.bash

PROTOC_GEN_GO_EXTRA_FLAGS :=
ifdef PROTOC_USE_BUF
PROTOC_GEN_GO_EXTRA_FLAGS := --use-buf
endif
ifdef PROTOC_USE_BUF_BY_DIR
PROTOC_GEN_GO_EXTRA_FLAGS := --use-buf --by-dir
endif

.PHONY: protocgengoclean
protocgengoclean:
	rm -rf "$(PROTOC_GEN_GO_OUT)"

.PHONY: protocgengo
protocgengo: protocgengoclean $(PROTOC) $(PROTOC_GEN_GO)
	bash $(MAKEGO)/scripts/protoc_gen_plugin.bash $(PROTOC_GEN_GO_EXTRA_FLAGS) \
		"--proto_path=$(PROTO_PATH)" \
		"--proto_include_path=$(CACHE_INCLUDE)" \
		$(patsubst %,--proto_include_path=%,$(PROTO_INCLUDE_PATHS)) \
		"--plugin_name=go" \
		"--plugin_out=$(PROTOC_GEN_GO_OUT)" \
		"--plugin_opt=$(PROTOC_GEN_GO_OPT)"

protocgenerate:: protocgengo

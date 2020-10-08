# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/dep_buf.mk)
$(call _conditional_include,$(MAKEGO)/dep_protoc_gen_go.mk)
# Must be set
$(call _assert_var,PROTOC_GEN_GO_OUT)
$(call _assert_var,CACHE_BIN)
$(call _assert_var,BUF)
$(call _assert_var,PROTOC_GEN_GO)

# Not modifiable for now
PROTOC_GEN_GO_OPT := paths=source_relative

.PHONY: protocgengoclean
protocgengoclean:
	rm -rf "$(PROTOC_GEN_GO_OUT)"
	mkdir -p "$(PROTOC_GEN_GO_OUT)"

.PHONY: protocgengo
protocgengo: protocgengoclean $(BUF) $(PROTOC_GEN_GO)
	$(CACHE_BIN)/buf beta generate \
		--plugin go \
		--plugin-out $(PROTOC_GEN_GO_OUT) \
		--plugin-opt $(PROTOC_GEN_GO_OPT)

bufgenerate:: protocgengo

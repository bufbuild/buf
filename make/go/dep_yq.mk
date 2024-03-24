# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,UNAME_OS)
$(call _assert_var,UNAME_ARCH)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/mikefarah/yq/releases 20240225 checked 20240320
YQ_VERSION ?= v4.42.1

ifeq ($(UNAME_OS),Darwin)
YQ_OS := darwin
ifeq ($(UNAME_ARCH),x86_64)
YQ_ARCH := amd64
endif
ifeq ($(UNAME_ARCH),arm64)
YQ_ARCH := arm64
endif
endif

ifeq ($(UNAME_ARCH),x86_64)
ifeq ($(UNAME_OS),Linux)
YQ_OS := linux
YQ_ARCH := amd64
endif
endif

YQ := $(CACHE_VERSIONS)/yq/$(YQ_VERSION)
$(YQ):
	@rm -f $(CACHE_BIN)/yq
	@mkdir -p $(CACHE_BIN)
	curl -sSL \
		https://github.com/mikefarah/yq/releases/download/$(YQ_VERSION)/yq_$(YQ_OS)_$(YQ_ARCH) \
		-o $(CACHE_BIN)/yq
	chmod +x $(CACHE_BIN)/yq
	@rm -rf $(dir $(YQ))
	@mkdir -p $(dir $(YQ))
	@touch $(YQ)

# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,UNAME_OS)
$(call _assert_var,UNAME_ARCH)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://jqlang.github.io/jq/download 20250701 checked 20250808
JQ_VERSION ?= 1.8.1

ifeq ($(UNAME_OS),Darwin)
JQ_OS := macos
else ifeq ($(UNAME_OS),Linux)
JQ_OS := linux
endif

ifeq ($(UNAME_ARCH),x86_64)
JQ_ARCH := amd64
else
JQ_ARCH := $(UNAME_ARCH)
endif

JQ := $(CACHE_BIN)/jq

$(CACHE_VERSIONS)/jq/jq-$(JQ_VERSION):
	@rm -f $(JQ)
	@rm -rf $(dir $@)
	@mkdir -p $(dir $@)
	curl -sSL \
		https://github.com/jqlang/jq/releases/download/jq-$(JQ_VERSION)/jq-$(JQ_OS)-$(JQ_ARCH) \
		-o $@
	@chmod +x $@
	@test -x $@
	@touch $@

$(JQ): $(CACHE_VERSIONS)/jq/jq-$(JQ_VERSION)
	@mkdir -p $(dir $@)
	@ln -sf $< $@

dockerdeps:: $(JQ)

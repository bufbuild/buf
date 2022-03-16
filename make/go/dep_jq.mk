# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,UNAME_OS)
$(call _assert_var,UNAME_ARCH)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://stedolan.github.io/jq/download checked 20220224
JQ_VERSION ?= 1.6

# jq does not have an ARM release on Github so we'll use
# the amd64 version via Rosetta
ifeq ($(UNAME_OS),Darwin)
JQ_OS := osx
JQ_ARCH := -amd64
endif

ifeq ($(UNAME_ARCH),x86_64)
ifeq ($(UNAME_OS),Linux)
JQ_OS := linux
JQ_ARCH := 64
endif
endif
JQ := $(CACHE_VERSIONS)/jq/$(JQ_VERSION)
$(JQ):
	@rm -f $(CACHE_BIN)/jq
	@mkdir -p $(CACHE_BIN)
	curl -sSL \
		https://github.com/stedolan/jq/releases/download/jq-$(JQ_VERSION)/jq-$(JQ_OS)$(JQ_ARCH) \
		-o $(CACHE_BIN)/jq
	chmod +x $(CACHE_BIN)/jq
	@rm -rf $(dir $(JQ))
	@mkdir -p $(dir $(JQ))
	@touch $(JQ)

dockerdeps:: $(JQ)

ifndef UNAME_OS
$(error UNAME_OS is not set)
endif
ifndef UNAME_ARCH
$(error UNAME_ARCH is not set)
endif
ifndef CACHE_VERSIONS
$(error CACHE_VERSIONS is not set)
endif
ifndef CACHE_BIN
$(error CACHE_BIN is not set)
endif
ifndef JQ_VERSION
$(error JQ_VERSION is not set)
endif

ifeq ($(UNAME_ARCH),x86_64)
ifeq ($(UNAME_OS),Darwin)
JQ_OS := osx
JQ_ARCH := -amd64
endif
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

deps:: $(JQ)

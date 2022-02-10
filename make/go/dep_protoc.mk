# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,UNAME_OS)
$(call _assert_var,UNAME_ARCH)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_INCLUDE)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/protocolbuffers/protobuf/releases 20220128 checked 20220210
PROTOC_VERSION ?= 3.19.4

# There are no protobuf releases for Darwin ARM so for
# now we always use the x86_64 release through Rosetta.
ifeq ($(UNAME_OS),Darwin)
PROTOC_OS := osx
PROTOC_ARCH := x86_64
endif
ifeq ($(UNAME_OS),Linux)
PROTOC_OS = linux
PROTOC_ARCH := $(UNAME_ARCH)
endif

PROTOC := $(CACHE_VERSIONS)/protoc/$(PROTOC_VERSION)
$(PROTOC):
	@if ! command -v curl >/dev/null 2>/dev/null; then echo "error: curl must be installed"  >&2; exit 1; fi
	@if ! command -v unzip >/dev/null 2>/dev/null; then echo "error: unzip must be installed"  >&2; exit 1; fi
	@rm -f $(CACHE_BIN)/protoc
	@rm -rf $(CACHE_INCLUDE)/google
	@mkdir -p $(CACHE_BIN) $(CACHE_INCLUDE)
	$(eval PROTOC_TMP := $(shell mktemp -d))
	cd $(PROTOC_TMP); curl -sSL https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-$(PROTOC_OS)-$(PROTOC_ARCH).zip -o protoc.zip
	cd $(PROTOC_TMP); unzip protoc.zip && mv bin/protoc $(CACHE_BIN)/protoc && mv include/google $(CACHE_INCLUDE)/google
	@rm -rf $(PROTOC_TMP)
	@rm -rf $(dir $(PROTOC))
	@mkdir -p $(dir $(PROTOC))
	@touch $(PROTOC)

dockerdeps:: $(PROTOC)

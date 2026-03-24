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
# https://github.com/protocolbuffers/protobuf/releases 20250225 checked 20250310
# NOTE: Set to version compatible with genproto source code (only used in tests).
PROTOC_VERSION ?= 33.3

# Google adds a dash to release candidate versions in the name of the
# release artifact, i.e. v27.0-rc1 -> v27.0-rc-1
ifeq (,$(findstring $(PROTOC_VERSION),-rc))
PROTOC_RELEASE_VERSION := $(subst -rc,-rc-,$(PROTOC_VERSION))
else
PROTOC_RELEASE_VERSION := $(PROTOC_VERSION)
endif

ifeq ($(UNAME_OS),Darwin)
PROTOC_OS := osx
ifeq ($(UNAME_ARCH),arm64)
PROTOC_ARCH := aarch_64
else
PROTOC_ARCH := x86_64
endif
endif
ifeq ($(UNAME_OS),Linux)
PROTOC_OS = linux
PROTOC_ARCH := $(UNAME_ARCH)
endif

PROTOC := $(CACHE_BIN)/protoc
PROTOC_INCLUDE := $(CACHE_INCLUDE)/google

$(CACHE_VERSIONS)/protoc/protoc-$(PROTOC_VERSION):
	@if ! command -v curl >/dev/null 2>/dev/null; then echo "error: curl must be installed"  >&2; exit 1; fi
	@if ! command -v unzip >/dev/null 2>/dev/null; then echo "error: unzip must be installed"  >&2; exit 1; fi
	@rm -f $(PROTOC)
	@rm -rf $(PROTOC_INCLUDE)
	@rm -rf $(dir $@)
	@mkdir -p $(dir $@)
	@mkdir -p $(CACHE_VERSIONS)/protoc/protoc-$(PROTOC_VERSION)-include
	$(eval PROTOC_TMP := $(shell mktemp -d))
	cd $(PROTOC_TMP); curl -sSL https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_RELEASE_VERSION)-$(PROTOC_OS)-$(PROTOC_ARCH).zip -o protoc.zip
	cd $(PROTOC_TMP); unzip protoc.zip && mv bin/protoc $@ && mv include/google $(CACHE_VERSIONS)/protoc/protoc-$(PROTOC_VERSION)-include/google
	@rm -rf $(PROTOC_TMP)
	@chmod +x $@
	@test -x $@
	@touch $@

$(PROTOC): $(CACHE_VERSIONS)/protoc/protoc-$(PROTOC_VERSION)
	@mkdir -p $(dir $@)
	@ln -sf $< $@

$(PROTOC_INCLUDE): $(CACHE_VERSIONS)/protoc/protoc-$(PROTOC_VERSION)
	@mkdir -p $(dir $@)
	@ln -sf $(CACHE_VERSIONS)/protoc/protoc-$(PROTOC_VERSION)-include/google $@

dockerdeps:: $(PROTOC) $(PROTOC_INCLUDE)

# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/bufbuild/buf/releases
BUF_VERSION ?= v1.4.0
# Settable
#
# If set, this path will be installed every time someone depends on $(BUF)
# as opposed to installing from github with @$(BUF_VERSION).
#
# This can be used to always do "go install ./cmd/buf" or
# "go install github.com/bufbuild/buf/cmd/buf".
BUF_GO_INSTALL_PATH ?=
ifneq ($(BUF_GO_INSTALL_PATH),)
.PHONY: __goinstallbuf
__goinstallbuf:
	go install $(BUF_GO_INSTALL_PATH)

BUF := __goinstallbuf

# Use this instead of "buf" when using buf.
BUF_BIN := $(CACHE_GOBIN)/buf
else
BUF := $(CACHE_VERSIONS)/buf/$(BUF_VERSION)
$(BUF):
	@rm -f $(CACHE_BIN)/buf
	GOBIN=$(CACHE_BIN) go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
	@rm -rf $(dir $(BUF))
	@mkdir -p $(dir $(BUF))
	@touch $(BUF)

# Use this instead of "buf" when using buf.
BUF_BIN := $(CACHE_BIN)/buf

dockerdeps:: $(BUF)
endif

# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_BIN)

# Unlike 'buf', we always install bufstyle from source since
# we don't release it as an independent binary.
.PHONY: $(BUFSTYLE)
$(BUFSTYLE):
	GOBIN=$(CACHE_BIN) go install ./cmd/bufstyle

# Use this instead of "buf" when using buf.
BUFSTYLE_BIN := $(CACHE_BIN)/bufstyle

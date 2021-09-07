# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,make/go/dep_buf.mk)

# Settable
BUF_LINT_INPUT ?=
# Settable
BUF_BREAKING_INPUT ?=
# Settable
BUF_BREAKING_AGAINST_INPUT ?=
# Settable

.PHONY: bufgeneratedeps
bufgeneratedeps:: $(BUF)

.PHONY: bufgenerateclean
bufgenerateclean::

.PHONY: bufgeneratesteps
bufgeneratesteps::

.PHONY: bufgenerate
bufgenerate:
	$(MAKE) bufgeneratedeps
	$(MAKE) bufgenerateclean
	$(MAKE) bufgeneratesteps

pregenerate:: bufgenerate

ifneq ($(BUF_LINT_INPUT),)
.PHONY: buflint
buflint: $(BUF)
	$(BUF_BIN) lint $(BUF_LINT_INPUT)

postlint:: buflint
endif

ifneq ($(BUF_BREAKING_INPUT),)
ifneq ($(BUF_BREAKING_AGAINST_INPUT),)
.PHONY: bufbreaking
bufbreaking: $(BUF)
	$(BUF_BIN) breaking $(BUF_BREAKING_INPUT) --against $(BUF_BREAKING_AGAINST_INPUT)

postlint:: bufbreaking
endif
endif

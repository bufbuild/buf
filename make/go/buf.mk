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
BUF_FORMAT_INPUT ?=

.PHONY: bufgeneratedeps
bufgeneratedeps:: $(BUF)

.PHONY: bufgenerateclean
bufgenerateclean::

.PHONY: bufgeneratesteps
bufgeneratesteps::

.PHONY: bufgenerate
bufgenerate: ## Run all generation steps for Protobuf.
	@$(MAKE) __bufgenerate
	@$(MAKE) licensegenerate

# __bufgenerate calls just the buf generate steps. It should
# only be referenced wtihin this file
#
# bufgenerate can be called independently of make generate and
# *should* do everything required for a proto-only change. It's
# not super-safe, but should work 99.9999% of the time, and CI
# will catch when it does not via calling the full make generate.
#
# It will not work if other generate steps are required for
# Protobuf files in the future that are not captured by
# __bufgenerate and licensegenerate, however this is very unlikely.

.PHONY: __bufgenerate
__bufgenerate:
	@echo make bufgeneratedeps
	@$(MAKE) bufgeneratedeps
ifneq ($(BUF_FORMAT_INPUT),)
	@echo buf format -w $(BUF_FORMAT_INPUT)
	@$(BUF_BIN) format -w $(BUF_FORMAT_INPUT)
endif
	@echo make bufgenerateclean
	@$(MAKE) bufgenerateclean
	@echo make bufgeneratesteps
	@$(MAKE) bufgeneratesteps

pregenerate:: __bufgenerate

.PHONY: buflintdeps
buflintdeps:: $(BUF)

ifneq ($(BUF_LINT_INPUT),)
.PHONY: buflint
buflint:
	@echo make buflintdeps
	@$(MAKE) buflintdeps
	@echo buf lint $(BUF_LINT_INPUT)
	@$(BUF_BIN) lint $(BUF_LINT_INPUT)

postlint:: buflint
endif

ifneq ($(BUF_BREAKING_INPUT),)
ifneq ($(BUF_BREAKING_AGAINST_INPUT),)
.PHONY: bufbreaking
bufbreaking: $(BUF)
	@echo buf breaking $(BUF_BREAKING_INPUT) --against $(BUF_BREAKING_AGAINST_INPUT)
	@$(BUF_BIN) breaking $(BUF_BREAKING_INPUT) --against $(BUF_BREAKING_AGAINST_INPUT)

postlint:: bufbreaking
endif
endif

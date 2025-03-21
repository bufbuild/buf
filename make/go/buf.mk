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
bufgenerate:
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

pregenerate:: bufgenerate

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

.PHONY: updatebufversion
updatebufversion:
	$(SED_I) -E "s/BUF_VERSION \?=.*/BUF_VERSION ?= v${RELEASE_BUF_VERSION}/" "make/go/dep_buf.mk"
	$(SED_I) -E "s/\# https\:\/\/github.com\/bufbuild\/buf\/releases.*/\# https\:\/\/github.com\/bufbuild\/buf\/releases $(shell date "+%Y%m%d") checked $(shell date "+%Y%m%d")/" "make/go/dep_buf.mk"

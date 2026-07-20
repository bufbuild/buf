# Managed by makego. DO NOT EDIT.

SHELL := /usr/bin/env bash -o pipefail

MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --no-print-directory

define _assert
	$(if $(1),,$(error Assertion failed: $(2)))
endef
define _assert_var
	$(call _assert,$($(1)),$(1) is not set)
endef
define _conditional_include
	$(if $(filter $(1),$(MAKEFILE_LIST)),,$(eval include $(1)))
endef
# Extracts the major.minor (e.g. 1.26) from a dotted version string (e.g. 1.26.3).
define _major_minor
$(word 1,$(subst ., ,$(1))).$(word 2,$(subst ., ,$(1)))
endef

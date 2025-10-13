# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,make/go/dep_bandeps.mk)

BANDEPS_CONFIG ?=

ifneq ($(BANDEPS_CONFIG),)
.PHONY: bandeps
bandeps: $(BANDEPS)
	bandeps -f $(BANDEPS_CONFIG)

postlonglint:: bandeps
endif

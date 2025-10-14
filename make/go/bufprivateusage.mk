# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,make/go/dep_bufprivateusage.mk)

BUFPRIVATEUSAGE_PKGS ?=

ifneq ($(BUFPRIVATEUSAGE_PKGS),)
.PHONY: bufprivateusage
bufprivateusage: $(BUFPRIVATEUSAGE)
	bufprivateusage $(BUFPRIVATEUSAGE_PKGS)

postprepostgenerate:: bufprivateusage
endif

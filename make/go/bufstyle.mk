# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/go.mk)
$(call _conditional_include,make/go/dep_bufstyle.mk)
# Must be set
$(call _assert_var,GOPKGS)

.PHONY: bufstyle
bufstyle: $(BUFSTYLE)
	@echo bufstyle GOPKGS
	@bufstyle $(GOPKGS)

postlint:: bufstyle

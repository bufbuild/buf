# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/go.mk)
$(call _conditional_include,make/go/dep_godoclint.mk)
# Must be set
$(call _assert_var,GOPKGS)

.PHONY: godoclint
godoclint: $(GODOCLINT)
	@echo godoclint GOPKGS
	@godoclint $(GOPKGS)

postlint:: godoclint

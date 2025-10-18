# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/go.mk)
$(call _conditional_include,make/go/dep_godoc_lint.mk)
# Must be set
$(call _assert_var,GOPKGS)

.PHONY: godoclint
godoclint: $(GODOC_LINT)
	@echo godoc-lint NON_GEN_GOPKGS
	@godoclint \
		-enable pkg-doc,single-pkg-doc,start-with-name,deprecated,no-unused-link \
		$(shell go list $(GOPKGS) | grep -v private\/gen)

postlint:: godoclint

#-enable pkg-doc,single-pkg-doc,require-pkg-doc,start-with-name,require-doc,deprecated,no-unused-link \

# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/dep_addlicense.mk)
# Must be set
$(call _assert_var,COPYRIGHT_OWNER)
# Must be set
$(call _assert_var,COPYRIGHT_YEAR)
# Must be set
$(call _assert_var,LICENSE_TYPE)
$(call _assert_var,ADDLICENSE)

# Settable ADDLICENSE_IGNORES

.PHONY: addlicense
addlicense: __addlicense_files $(ADDLICENSE)
	@$(foreach addlicense_file,$(sort $(ADDLICENSE_FILES)),addlicense -c "$(COPYRIGHT_OWNER)" -l "$(LICENSE_TYPE)" -y "$(COPYRIGHT_YEAR)" $(addlicense_file) || exit 1;)

licensegenerate:: addlicense

.PHONY: __addlicense_files
__addlicense_files:
ifdef ADDLICENSE_IGNORES
	$(eval ADDLICENSE_FILES := $(shell git ls-files | grep -v $(patsubst %,-e %,$(sort $(ADDLICENSE_IGNORES)))))
else
	$(eval ADDLICENSE_FILES := .)
endif

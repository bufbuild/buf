# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,make/go/dep_license_header.mk)
$(call _conditional_include,make/go/dep_git_ls_files_unstaged.mk)

# Must be set
$(call _assert_var,LICENSE_HEADER_LICENSE_TYPE)
# Must be set
$(call _assert_var,LICENSE_HEADER_COPYRIGHT_HOLDER)
# Must be set
$(call _assert_var,LICENSE_HEADER_YEAR_RANGE)
# Must be set
$(call _assert_var,LICENSE_HEADER_IGNORES)

.PHONY: licenseheader
licenseheader: $(LICENSE_HEADER) $(GIT_LS_FILES_UNSTAGED)
	git-ls-files-unstaged | \
		grep -v $(patsubst %,-e %,$(sort $(LICENSE_HEADER_IGNORES))) | \
		xargs license-header \
			--license-type "$(LICENSE_HEADER_LICENSE_TYPE)" \
			--copyright-holder "$(LICENSE_HEADER_COPYRIGHT_HOLDER)" \
			--year-range "$(LICENSE_HEADER_YEAR_RANGE)"

licensegenerate:: licenseheader

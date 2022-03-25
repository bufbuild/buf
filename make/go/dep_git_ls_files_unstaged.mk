# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _conditional_include,$(MAKEGO)/dep_buf.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)
$(call _assert_var,BUF_VERSION)

# Settable
# https://github.com/bufbuild/buf/releases
GIT_LS_FILES_UNSTAGED_VERSION ?= $(BUF_VERSION)

GIT_LS_FILES_UNSTAGED := $(CACHE_VERSIONS)/git-ls-files-unstaged/$(GIT_LS_FILES_UNSTAGED_VERSION)
$(GIT_LS_FILES_UNSTAGED):
	@rm -f $(CACHE_BIN)/licnese-header
	GOBIN=$(CACHE_BIN) go install github.com/bufbuild/buf/private/pkg/git/cmd/git-ls-files-unstaged@$(GIT_LS_FILES_UNSTAGED_VERSION)
	@rm -rf $(dir $(GIT_LS_FILES_UNSTAGED))
	@mkdir -p $(dir $(GIT_LS_FILES_UNSTAGED))
	@touch $(GIT_LS_FILES_UNSTAGED)

dockerdeps:: $(GIT_LS_FILES_UNSTAGED)

# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,GOBIN)

# Settable
# https://github.com/gordonklaus/ineffassign/commits/master 20190601
INEFFASSIGN_VERSION ?= ed7b1b5ee0f816bbc0ff35bf7c6fdb4f53b6c59a

INEFFASSIGN := $(CACHE_VERSIONS)/ineffassign/$(INEFFASSIGN_VERSION)
$(INEFFASSIGN):
	@rm -f $(GOBIN)/ineffassign
	$(eval INEFFASSIGN_TMP := $(shell mktemp -d))
	cd $(INEFFASSIGN_TMP); go get github.com/gordonklaus/ineffassign@$(INEFFASSIGN_VERSION)
	@rm -rf $(INEFFASSIGN_TMP)
	@rm -rf $(dir $(INEFFASSIGN))
	@mkdir -p $(dir $(INEFFASSIGN))
	@touch $(INEFFASSIGN)

dockerdeps:: $(INEFFASSIGN)

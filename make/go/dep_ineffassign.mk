# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,GOBIN)

# Settable
# https://github.com/gordonklaus/ineffassign/commits/master 20200309 checked 20200311
INEFFASSIGN_VERSION ?= 7953dde2c7bf4ce700d9f14c2e41c0966763760c

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

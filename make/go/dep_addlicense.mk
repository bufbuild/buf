# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,GOBIN)

# Settable
# https://github.com/google/addlicense/commits/master 2020422 checked 2020503
ADDLICENSE_VERSION ?= 68a83edd47bcc23c80f28cd8c6e62b2db7275d3b

ADDLICENSE := $(CACHE_VERSIONS)/addlicense/$(ADDLICENSE_VERSION)
$(ADDLICENSE):
	@rm -f $(GOBIN)/addlicense
	$(eval ADDLICENSE_TMP := $(shell mktemp -d))
	cd $(ADDLICENSE_TMP); go get github.com/google/addlicense@$(ADDLICENSE_VERSION)
	@rm -rf $(ADDLICENSE_TMP)
	@rm -rf $(dir $(ADDLICENSE))
	@mkdir -p $(dir $(ADDLICENSE))
	@touch $(ADDLICENSE)

dockerdeps:: $(ADDLICENSE)

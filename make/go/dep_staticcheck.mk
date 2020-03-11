# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,GOBIN)

# Settable
# https://github.com/dominikh/go-tools/commits/master 20200222 checked 20200311
STATICCHECK_VERSION ?= c702d0a1c8d58bb55e039d028e92ae714040936c

STATICCHECK := $(CACHE_VERSIONS)/staticcheck/$(STATICCHECK_VERSION)
$(STATICCHECK):
	@rm -f $(GOBIN)/staticcheck
	$(eval STATICCHECK_TMP := $(shell mktemp -d))
	cd $(STATICCHECK_TMP); go get honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	@rm -rf $(STATICCHECK_TMP)
	@rm -rf $(dir $(STATICCHECK))
	@mkdir -p $(dir $(STATICCHECK))
	@touch $(STATICCHECK)

dockerdeps:: $(STATICCHECK)

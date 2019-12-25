# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,GOBIN)

# Settable
# https://github.com/dominikh/go-tools/commits/master 20191201
STATICCHECK_VERSION ?= 32de78dc1c6c931f2bf017870750b0ca80006601

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

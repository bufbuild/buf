# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,GOBIN)

# Settable
# https://github.com/golang/lint/commits/master 20200301 checked 20200531
GOLINT_VERSION ?= 738671d3881b9731cc63024d5d88cf28db875626

GOLINT := $(CACHE_VERSIONS)/golint/$(GOLINT_VERSION)
$(GOLINT):
	@rm -f $(GOBIN)/golint
	$(eval GOLINT_TMP := $(shell mktemp -d))
	cd $(GOLINT_TMP); go get golang.org/x/lint/golint@$(GOLINT_VERSION)
	@rm -rf $(GOLINT_TMP)
	@rm -rf $(dir $(GOLINT))
	@mkdir -p $(dir $(GOLINT))
	@touch $(GOLINT)

dockerdeps:: $(GOLINT)

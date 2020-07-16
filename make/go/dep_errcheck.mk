# Managed by makego. DO NOT EDIT.

# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/kisielk/errcheck/commits/master 20200103 checked 20200531
ERRCHECK_VERSION ?= e14f8d59a22d460d56c5ee92507cd94c78fbf274

ERRCHECK := $(CACHE_VERSIONS)/errcheck/$(ERRCHECK_VERSION)
$(ERRCHECK):
	@rm -f $(CACHE_BIN)/errcheck
	$(eval ERRCHECK_TMP := $(shell mktemp -d))
	cd $(ERRCHECK_TMP); GOBIN=$(CACHE_BIN) go get github.com/kisielk/errcheck@$(ERRCHECK_VERSION)
	@rm -rf $(ERRCHECK_TMP)
	@rm -rf $(dir $(ERRCHECK))
	@mkdir -p $(dir $(ERRCHECK))
	@touch $(ERRCHECK)

dockerdeps:: $(ERRCHECK)

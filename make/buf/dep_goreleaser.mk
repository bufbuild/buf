# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Settable
# https://github.com/goreleaser/goreleaser/releases 20211013 checked 20211013
GORELEASER_VERSION ?= v0.182.1

GORELEASER := $(CACHE_VERSIONS)/GORELEASER/$(GORELEASER_VERSION)
$(GORELEASER):
	@rm -f $(CACHE_BIN)/goreleaser
	GOBIN=$(CACHE_BIN) go install github.com/goreleaser/goreleaser@$(GORELEASER_VERSION)
	@rm -rf $(dir $(GORELEASER))
	@mkdir -p $(dir $(GORELEASER))
	@touch $(GORELEASER)

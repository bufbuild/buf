# Managed by makego. DO NOT EDIT.
#
# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# We want to ensure we rebuild govulncheck every time we require a new Go minor version.
# Otherwise, the cached version may not support the latest language features.
GOVULNCHECK_GO_VERSION := $(shell go env GOVERSION | sed 's/^go//' | cut -d'.' -f1-2)

# Settable
# https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck 20250106 checked 20250212
GOVULNCHECK_VERSION ?= v1.1.4

GOVULNCHECK := $(CACHE_VERSIONS)/govulncheck/$(GOVULNCHECK_VERSION)-go$(GOVULNCHECK_GO_VERSION)
$(GOVULNCHECK):
	@rm -f $(CACHE_BIN)/govulncheck
	GOBIN=$(CACHE_BIN) go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	@rm -rf $(dir $(GOVULNCHECK))
	@mkdir -p $(dir $(GOVULNCHECK))
	@touch $(GOVULNCHECK)

dockerdeps:: $(GOVULNCHECK)

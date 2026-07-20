# Managed by makego. DO NOT EDIT.
#
# Must be set
$(call _assert_var,MAKEGO)
$(call _conditional_include,$(MAKEGO)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# We want to ensure we rebuild govulncheck every time we require a new Go minor version.
# Otherwise, the cached version may not support the latest language features.
# This version is the go toolchain version (which may be more specific than the module
# version) to ensure the build handles specific language features in newer toolchains.
GOVULNCHECK_GOTOOLCHAIN_VERSION := $(patsubst go%,%,$(shell go env GOVERSION))
GOVULNCHECK_GO_VERSION := $(call _major_minor,$(GOVULNCHECK_GOTOOLCHAIN_VERSION))

# Settable
# https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck 20260625 checked 20260625
GOVULNCHECK_VERSION ?= v1.5.0

GOVULNCHECK := $(CACHE_BIN)/govulncheck

$(CACHE_VERSIONS)/govulncheck/govulncheck-$(GOVULNCHECK_VERSION)-go$(GOVULNCHECK_GO_VERSION):
	@rm -f $(GOVULNCHECK)
	@rm -rf $(dir $@)
	@mkdir -p $(dir $@)
	GOBIN=$(dir $@) GOTOOLCHAIN=go$(GOVULNCHECK_GOTOOLCHAIN_VERSION) go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	@mv $(dir $@)/govulncheck $@
	@test -x $@
	@touch $@

$(GOVULNCHECK): $(CACHE_VERSIONS)/govulncheck/govulncheck-$(GOVULNCHECK_VERSION)-go$(GOVULNCHECK_GO_VERSION)
	@mkdir -p $(dir $@)
	@ln -sf $< $@

dockerdeps:: $(GOVULNCHECK)

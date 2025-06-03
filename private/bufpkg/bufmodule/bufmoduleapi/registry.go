// Copyright 2020-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bufmoduleapi

import (
	"fmt"
	"strings"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/gen/data/datalegacyfederation"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// If we ever get to a case where we're supporting legacy federation, and we're moving buf.build,
// we have way bigger problems than this hardcoded variable.
const defaultPublicRegistry = "buf.build"

type hasFullName interface {
	FullName() bufparse.FullName
}

// getPrimaryRegistryAndLegacyFederationAllowed returns the primary registry and whether
// legacyFederation is allowed.
//
// If there is only a single registry for all the input values, this registry is returned as
// the primary.
//
// If there is more than one registry, in the case where legacy federation is not allowed,
// we return an error to the user.
//
// If legacy federation is allowed, a check is made on the number of registries parsed
// among input values.
//
// If there are two registries, the primary will be the non-public registry, and we validate
// the secondary is the public registry (buf.build).
//
// If there are more than two registries, an error is returned - we have never supported
// federation beyond a single non-public registry depending on the public registry (buf.build).
//
// This is used to support legacy federation.
func getPrimaryRegistryAndLegacyFederationAllowed[T hasFullName](
	s []T,
	publicRegistry string,
	additionalLegacyFederationRegistry string,
) (string, bool, error) {
	if len(s) == 0 {
		return "", false, syserror.New("must have at least one value in getPrimarySecondaryRegistry")
	}
	registries, err := getRegistries(s)
	if err != nil {
		return "", false, err
	}
	legacyFederationAllowed, err := isLegacyFederationAllowed(registries, additionalLegacyFederationRegistry)
	if err != nil {
		return "", false, err
	}
	switch len(registries) {
	case 0:
		return "", false, syserror.New("no registries detected in getPrimarySecondaryRegistry")
	case 1:
		return registries[0], legacyFederationAllowed, nil
	case 2:
		if legacyFederationAllowed {
			if registries[0] != publicRegistry && registries[1] != publicRegistry {
				return "", legacyFederationAllowed, fmt.Errorf("cannot use federation between two non-public registries: %s, %s", registries[0], registries[1])
			}
			if registries[0] == publicRegistry {
				return registries[1], legacyFederationAllowed, nil
			}
			return registries[0], legacyFederationAllowed, nil
		}
		fallthrough
	default:
		return "", false, fmt.Errorf("dependencies across multiple registries are not allowed: %s", strings.Join(registries, ", "))
	}
}

func isLegacyFederationAllowed(registries []string, additionalLegacyFederationRegistry string) (bool, error) {
	for _, registry := range registries {
		exists, err := datalegacyfederation.Exists(registry)
		if err != nil {
			return false, err
		}
		if exists {
			return true, nil
		}
		// Checking that additionalLegacyFederationRegistry != "" just as a defensive measure, even though
		// nothing in registries should be empty.
		if additionalLegacyFederationRegistry != "" && registry == additionalLegacyFederationRegistry {
			return true, nil
		}
	}
	return false, nil
}

func getRegistries[T hasFullName](s []T) ([]string, error) {
	registryMap, err := xslices.ToValuesMapError(
		s,
		func(e T) (string, error) {
			moduleFullName := e.FullName()
			if moduleFullName == nil {
				return "", syserror.Newf("no FullName for %v", e)
			}
			registry := moduleFullName.Registry()
			if registry == "" {
				return "", syserror.Newf("no registry for %v", e)
			}
			return registry, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return xslices.MapKeysToSortedSlice(registryMap), nil
}

// getSingleRegistryForContentModules returns the single registry for the content modules in Upload.
//
// Returns error if there is more than one module.
func getSingleRegistryForContentModules(contentModules []bufmodule.Module) (string, error) {
	if len(contentModules) == 0 {
		return "", syserror.New("requires at least one module to resolve registry")
	}
	var registry string
	for _, module := range contentModules {
		moduleFullName := module.FullName()
		if moduleFullName == nil {
			return "", syserror.Newf("expected module name for %s", module.Description())
		}
		moduleRegistry := moduleFullName.Registry()
		if registry != "" && moduleRegistry != registry {
			// We don't allow the upload of content across multiple registries, but in the legacy federation
			// case, we DO allow for depending on other registries.
			return "", fmt.Errorf(
				"cannot upload content for multiple registries at once: %s, %s",
				registry,
				moduleRegistry,
			)
		}
		registry = moduleRegistry
	}
	return registry, nil
}

func validateDepRegistries(primaryRegistry string, depRegistries []string, publicRegistry string) error {
	switch len(depRegistries) {
	case 0:
		return nil
	case 1, 2:
		for _, depRegistry := range depRegistries {
			if depRegistry != publicRegistry && depRegistry != primaryRegistry {
				return fmt.Errorf("dependency must be on either %s or %s but was on %s", publicRegistry, primaryRegistry, depRegistry)
			}
			if primaryRegistry == publicRegistry && depRegistry != publicRegistry {
				// Public to private was never allowed.
				return fmt.Errorf("cannot have dependencies on %s modules from %s modules", primaryRegistry, depRegistry)
			}
		}
		return nil
	default:
		return fmt.Errorf("dependencies across multiple registries are not allowed: %s", strings.Join(depRegistries, ", "))
	}
}

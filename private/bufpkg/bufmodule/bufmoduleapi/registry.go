// Copyright 2020-2024 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// If we ever get to a case where we're supporting legacy federation, and we're moving buf.build,
// we have way bigger problems than this hardcoded variable.
const publicRegistry = "buf.build"

type hasModuleFullName interface {
	ModuleFullName() bufmodule.ModuleFullName
}

// getPrimarySecondaryRegistry returns the primary and secondary registry for a call that supports
// federation.
//
// If there is only a single registry for all the input values, this registry is returned as
// the primary, and empty is returned for the secondary.
//
// If there are two registries, the primary will be the non-public registry, the secondary
// will be buf.build.
//
// If there are more than two registries, an error is returned - we have never supported federation
// beyond a non-public registry depending on buf.build.
//
// This is used to support legacy federation.
func getPrimarySecondaryRegistry[T hasModuleFullName](s []T) (string, string, error) {
	if len(s) == 0 {
		return "", "", syserror.New("must have at least one value in getPrimarySecondaryRegistry")
	}
	registryMap, err := slicesext.ToUniqueValuesMapError(
		s,
		func(e T) (string, error) {
			moduleFullName := e.ModuleFullName()
			if moduleFullName == nil {
				return "", syserror.New("expected non-nil ModuleFullName in getPrimarySecondaryRegistry")
			}
			return moduleFullName.Registry(), nil
		},
	)
	if err != nil {
		return "", "", err
	}
	registries := slicesext.MapKeysToSortedSlice(registryMap)
	switch len(registries) {
	case 0:
		return "", "", syserror.New("no registries detected in getPrimarySecondaryRegistry")
	case 1:
		return registries[0], "", nil
	case 2:
		if registries[0] != publicRegistry && registries[1] != publicRegistry {
			return "", "", fmt.Errorf("cannot use federation between two non-public registries: %s, %s", registries[0], registries[1])
		}
		if registries[0] == publicRegistry {
			return registries[1], registries[0], nil
		}
		return registries[0], registries[1], nil
	default:
		return "", "", fmt.Errorf("attempting to perform a BSR operation for more than two registries: %s. You may be attempting to use dependencies between registries - this is not allowed outside of a few early customers.", strings.Join(registries, ", "))
	}
}

func validateDepRegistries(primaryRegistry string, depRegistries []string) error {
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
		return fmt.Errorf("attempting to perform a BSR operation for more than two registries: %s. You may be attempting to use dependencies between registries - this is not allowed outside of a few early customers.", strings.Join(depRegistries, ", "))
	}
}

func validateRegistryIsPrimaryOrSecondary(registry string, primaryRegistry string, secondaryRegistry string) error {
	if registry != primaryRegistry && registry != secondaryRegistry {
		// Could borderline be a system error, regardless we should enforce this so this doesn't propagate.
		return fmt.Errorf("expected to only have return values in registries %s and %s but found value in %s", primaryRegistry, secondaryRegistry, registry)
	}
	return nil
}
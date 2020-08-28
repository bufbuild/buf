// Copyright 2020 Buf Technologies, Inc.
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

package bufmodule

// errNoDigest is returned when a resolved module was required.
type errNoDigest struct {
	moduleName ModuleName
}

func (e *errNoDigest) Error() string {
	if e.moduleName == nil {
		return "an errNoDigest type requires a ModuleName"
	}
	return "module does not include a resolved digest: " + e.moduleName.String()
}

// Is implements errors.Is for errNoDigest
func (e *errNoDigest) Is(err error) bool {
	_, ok := err.(*errNoDigest)
	return ok
}

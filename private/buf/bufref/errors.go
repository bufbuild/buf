// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufref

import (
	"errors"
	"fmt"
)

func newValueEmptyError() error {
	return errors.New("required")
}

func newValueMultipleHashtagsError(value string) error {
	return fmt.Errorf("%q has multiple #s which is invalid", value)
}

func newValueStartsWithHashtagError(value string) error {
	return fmt.Errorf("%q starts with # which is invalid", value)
}

func newValueEndsWithHashtagError(value string) error {
	return fmt.Errorf("%q ends with # which is invalid", value)
}

func newOptionsInvalidError(s string) error {
	return fmt.Errorf("invalid options: %q", s)
}

func newOptionsDuplicateKeyError(key string) error {
	return fmt.Errorf("duplicate options key: %q", key)
}

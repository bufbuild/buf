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

package bufconfig

// TODO: need to handle bufmigrate, that likely moves into this package.
// TODO: need to handle buf mod init --doc
// TODO: All migration code between v1beta1, v1, v2 should live within this package, so that
// we can expose less public types.
// TODO: optimally, we can split this package into bufconfig, bufconfigfile, and ban
// core from depending on bufconfigfile. See how it ends up.

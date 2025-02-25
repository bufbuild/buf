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

package bufimageutil

const (
	// These constants are tag numbers for fields of messages in descriptor.proto.
	// We use them to construct source code info paths, which must be re-written
	// when we filter out elements of an image.

	fileDependencyTag         = 3
	filePublicDependencyTag   = 10
	fileWeakDependencyTag     = 11
	fileMessagesTag           = 4
	fileEnumsTag              = 5
	fileServicesTag           = 6
	fileExtensionsTag         = 7
	fileOptionsTag            = 8
	messageFieldsTag          = 2
	messageNestedMessagesTag  = 3
	messageEnumsTag           = 4
	messageExtensionsTag      = 6
	messageOptionsTag         = 7
	messageOneofsTag          = 8
	messageExtensionRangesTag = 5
	messageReservedRangesTag  = 9
	messageReservedNamesTag   = 10
	extensionRangeOptionsTag  = 3
	fieldOptionsTag           = 8
	oneofOptionsTag           = 2
	enumOptionsTag            = 3
	enumValuesTag             = 2
	enumValueOptionsTag       = 3
	serviceMethodsTag         = 2
	serviceOptionsTag         = 3
	methodOptionsTag          = 4
)

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
	messageFieldsTag          = 2
	messageNestedMessagesTag  = 3
	messageEnumsTag           = 4
	messageExtensionsTag      = 6
	messageOneofsTag          = 8
	messageExtensionRangesTag = 5
	messageReservedRangesTag  = 9
	messageReservedNamesTag   = 10
	enumValuesTag             = 2
	serviceMethodsTag         = 2
)

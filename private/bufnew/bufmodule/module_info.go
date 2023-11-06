package bufmodule

// ModuleInfo contains identifying information for a Module.
//
// It is embedded inside a Module, and therefore is always available from FileInfos as well.
type ModuleInfo interface {
	// ModuleSetID returns a identifier for this Module in the context if its enclosing ModuleSet.
	//
	// This will always be non-empty.
	// The shape of this should not be relied on outside of this being non-empty.
	// Unique within a given ModuleSet.
	//
	// While the shape should not be relied upon, the current semantics are:
	//   - If ModuleFullName and CommitID are present, this is "registry/owner/name:commit".
	//   - If only ModuleFullName is present, this is "registry/owner/name".
	//   - If neither are present, the constructor is responsible for coming up with
	//	   a unique ID, usually related to the location on disk of the Module.
	ModuleSetID() string
	// ModuleFullName returns the full name of the Module, if present.
	//
	// May be nil.
	ModuleFullName() ModuleFullName
	// CommitID returns the ID of the Commit, if present.
	//
	// This is a BSR API ID.
	// May be empty.
	// If ModuleFullName is nil, this will always be empty.
	CommitID() string

	isModuleInfo()
}

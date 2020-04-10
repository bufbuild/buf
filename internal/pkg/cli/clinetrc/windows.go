// +build windows

package clinetrc

// netrcFilename is the netrc filename.
//
// This will be .netrc for darwin and linux.
// This will be _netrc for windows.
const netrcFilename = "_netrc"

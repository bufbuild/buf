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

package bufplugin

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/google/uuid"
)

// Plugin presents a BSR plugin.
type Plugin interface {
	// OpaqueID returns an unstructured ID that can uniquely identify a Plugin
	// relative to the Workspace.
	//
	// An OpaqueID's structure should not be relied upon, and is not a
	// globally-unique identifier. It's uniqueness property only applies to
	// the lifetime of the Plugin, and only within the Workspace the Plugin
	// is defined in.
	//
	// If two Plugins have the same Name and Args, they will have the same OpaqueID.
	OpaqueID() string
	// Name returns the name of the Plugin.
	//  - For local Plugins, this is the path to the executable binary.
	//  - For local Wasm Plugins, this is the path to the Wasm binary.
	//  - For remote Plugins, this is the FullName of the Plugin in the form
	//    remote/owner/name.
	//
	// This is never empty.
	Name() string
	// Args returns the arguments to invoke the Plugin.
	//
	// May be nil.
	Args() []string
	// FullName returns the full name of the Plugin.
	//
	// May be nil. Callers should not rely on this value being present.
	// However, this is always present for remote Plugins.
	//
	// Use OpaqueID as an always-present identifier.
	FullName() bufparse.FullName
	// CommitID returns the BSR ID of the Commit.
	//
	// It is up to the caller to convert this to a dashless ID when necessary.
	//
	// May be empty, that is CommitID() == uuid.Nil may be true.
	// Callers should not rely on this value being present.
	//
	// If FullName is nil, this will always be empty.
	CommitID() uuid.UUID
	// Description returns a human-readable description of the Plugin.
	//
	// This is used to construct descriptive error messages pointing to configured plugins.
	//
	// This will never be empty. If a description was not explicitly set, this falls back to
	// OpaqueID.
	Description() string
	// Digest returns the Plugin digest for the given DigestType.
	//
	// Note this is *not* a bufcas.Digest - this is a Digest.
	// bufcas.Digests are a lower-level type that just deal in terms of
	// files and content. A Digest is a specific algorithm applied to the
	// content of a Plugin.
	//
	// Will return an error if the Plugin is not a Wasm Plugin.
	Digest(DigestType) (Digest, error)
	// Data returns the bytes of the Plugin as a Wasm module.
	//
	// This is the raw bytes of the Wasm module in an uncompressed form.
	//
	// Will return an error if the Plugin is not a Wasm Plugin.
	Data() ([]byte, error)
	// IsWasm returns true if the Plugin is a Wasm Plugin.
	//
	// Plugins are either Wasm or local.
	//
	// A Wasm Plugin is a Plugin that is a Wasm module. Wasm Plugins are invoked
	// with the wasm.Runtime. The Plugin will have Data and will be able to
	// calculate Digests.
	//
	// Wasm Plugins will always have Data.
	IsWasm() bool
	// IsLocal returns true if the Plugin is a local Plugin.
	//
	// Plugins are either local or remote.
	//
	// A local Plugin is one that is built from sources from the "local context",
	// such as a Workspace. Local Plugins are important for understanding what Plugins
	// to push.
	//
	// Remote Plugins will always have FullNames.
	IsLocal() bool

	isPlugin()
}

// NewLocalPlugin returns a new Plugin for a local plugin.
//
// The name is the path to the executable binary.
// The args are the arguments to invoke the Plugin. These are passed to the Plugin
// as command line arguments.
func NewLocalPlugin(
	name string,
	args []string,
) (Plugin, error) {
	return newPlugin(
		"", // description
		nil,
		name,
		args,
		uuid.Nil, // commitID
		false,    // isWasm
		true,     // isLocal
		nil,      // getData
	)
}

// NewLocalWasmPlugin returns a new Plugin for a local Wasm plugin.
//
// The pluginFullName may be nil.
// The name is the path to the Wasm plugin and must end with .wasm.
// The args are the arguments to the Wasm plugin. These are passed to the Wasm plugin
// as command line arguments.
// The getData function is called to get the bytes of the Wasm plugin.
// This is the raw bytes of the Wasm module in an uncompressed form.
func NewLocalWasmPlugin(
	pluginFullName bufparse.FullName,
	name string,
	args []string,
	getData func() ([]byte, error),
) (Plugin, error) {
	return newPlugin(
		"", // description
		pluginFullName,
		name,
		args,
		uuid.Nil, // commitID
		true,     // isWasm
		true,     // isLocal
		getData,
	)
}

// NewRemoteWasmPlugin returns a new Plugin for a remote Wasm plugin.
//
// The pluginFullName is the remote reference to the plugin.
// The args are the arguments to the remote plugin. These are passed to the remote plugin
// as command line arguments.
// The commitID is the BSR ID of the Commit.
// It is up to the caller to convert this to a dashless ID when necessary.
// The getData function is called to get the bytes of the Wasm plugin.
// This is the raw bytes of the Wasm module in an uncompressed form.
func NewRemoteWasmPlugin(
	pluginFullName bufparse.FullName,
	args []string,
	commitID uuid.UUID,
	getData func() ([]byte, error),
) (Plugin, error) {
	return newPlugin(
		"", // description
		pluginFullName,
		pluginFullName.String(),
		args,
		commitID,
		true,  // isWasm
		false, // isLocal
		getData,
	)
}

// *** PRIVATE ***

type plugin struct {
	description    string
	pluginFullName bufparse.FullName
	name           string
	args           []string
	commitID       uuid.UUID
	isWasm         bool
	isLocal        bool
	getData        func() ([]byte, error)

	digestTypeToGetDigest map[DigestType]func() (Digest, error)
}

func newPlugin(
	description string,
	pluginFullName bufparse.FullName,
	name string,
	args []string,
	commitID uuid.UUID,
	isWasm bool,
	isLocal bool,
	getData func() ([]byte, error),
) (*plugin, error) {
	if name == "" {
		return nil, syserror.New("name not present when constructing a Plugin")
	}
	if isWasm && getData == nil {
		return nil, syserror.Newf("getData not present when constructing a Wasm Plugin")
	}
	if !isLocal && pluginFullName == nil {
		return nil, syserror.New("pluginFullName not present when constructing a remote Plugin")
	}
	if !isLocal && !isWasm {
		return nil, syserror.New("remote non-Wasm Plugins are not supported")
	}
	if isLocal && commitID != uuid.Nil {
		return nil, syserror.New("commitID present when constructing a local Plugin")
	}
	if pluginFullName == nil && commitID != uuid.Nil {
		return nil, syserror.New("pluginFullName not present and commitID present when constructing a remote Plugin")
	}
	plugin := &plugin{
		description:    description,
		pluginFullName: pluginFullName,
		name:           name,
		args:           args,
		commitID:       commitID,
		isWasm:         isWasm,
		isLocal:        isLocal,
		getData:        sync.OnceValues(getData),
	}
	plugin.digestTypeToGetDigest = newSyncOnceValueDigestTypeToGetDigestFuncForPlugin(plugin)
	return plugin, nil
}

func (p *plugin) OpaqueID() string {
	return strings.Join(append([]string{p.name}, p.args...), " ")
}

func (p *plugin) Name() string {
	return p.name
}

func (p *plugin) Args() []string {
	return p.args
}

func (p *plugin) FullName() bufparse.FullName {
	return p.pluginFullName
}

func (p *plugin) CommitID() uuid.UUID {
	return p.commitID
}

func (p *plugin) Description() string {
	if p.description != "" {
		return p.description
	}
	return p.OpaqueID()
}

func (p *plugin) Data() ([]byte, error) {
	if !p.isWasm {
		return nil, fmt.Errorf("Plugin is not a Wasm Plugin")
	}
	return p.getData()
}

func (p *plugin) Digest(digestType DigestType) (Digest, error) {
	getDigest, ok := p.digestTypeToGetDigest[digestType]
	if !ok {
		return nil, syserror.Newf("DigestType %v was not in plugin.digestTypeToGetDigest", digestType)
	}
	return getDigest()
}

func (p *plugin) IsWasm() bool {
	return p.isWasm
}

func (p *plugin) IsLocal() bool {
	return p.isLocal
}

func (p *plugin) isPlugin() {}

func newSyncOnceValueDigestTypeToGetDigestFuncForPlugin(plugin *plugin) map[DigestType]func() (Digest, error) {
	m := make(map[DigestType]func() (Digest, error))
	for digestType := range digestTypeToString {
		m[digestType] = sync.OnceValues(newGetDigestFuncForPluginAndDigestType(plugin, digestType))
	}
	return m
}

func newGetDigestFuncForPluginAndDigestType(plugin *plugin, digestType DigestType) func() (Digest, error) {
	return func() (Digest, error) {
		data, err := plugin.getData()
		if err != nil {
			return nil, err
		}
		bufcasDigest, err := bufcas.NewDigestForContent(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		return NewDigest(digestType, bufcasDigest)
	}
}

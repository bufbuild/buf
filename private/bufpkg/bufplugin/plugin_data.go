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

package bufplugin

import (
	"bytes"
	"context"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
)

// TODO(ed): comment.
type PluginData interface {
	// PluginKey used to downoad this PluginData.
	//
	// The Digest from this PluginKey is used for tamper-proofing.
	PluginKey() PluginKey
	// Data returns the raw bytes for the plugin data.
	Data() ([]byte, error)

	isPluginData()
}

// NewPluginData returns a new PluginData.
func NewPluginData(
	ctx context.Context,
	pluginKey PluginKey,
	getData func() ([]byte, error),
) (PluginData, error) {
	return newPluginData(
		ctx,
		pluginKey,
		getData,
	)
}

// *** PRIVATE ***

type pluginData struct {
	pluginKey PluginKey
	getData   func() ([]byte, error)

	checkDigest func() error
}

func newPluginData(
	ctx context.Context,
	pluginKey PluginKey,
	getData func() ([]byte, error),
) (*pluginData, error) {

	pluginData := &pluginData{
		pluginKey: pluginKey,
		getData:   getData,
	}
	pluginData.checkDigest = sync.OnceValue(func() error {
		pluginData, err := pluginData.getData()
		if err != nil {
			return err
		}
		bufcasDigest, err := bufcas.NewDigestForContent(
			bytes.NewReader(pluginData),
		)
		if err != nil {
			return err
		}
		actualDigest, err := NewDigest(DigestTypeP1, bufcasDigest)
		if err != nil {
			return err
		}
		expectedDigest, err := pluginKey.Digest()
		if err != nil {
			return err
		}
		if !DigestEqual(actualDigest, expectedDigest) {
			return &DigestMismatchError{
				PluginFullName: pluginKey.PluginFullName(),
				CommitID:       pluginKey.CommitID(),
				ExpectedDigest: expectedDigest,
				ActualDigest:   actualDigest,
			}

		}
		return nil
	})
	return pluginData, nil
}

func (p *pluginData) PluginKey() PluginKey {
	return p.pluginKey
}

func (p *pluginData) Data() ([]byte, error) {
	if err := p.checkDigest(); err != nil {
		return nil, err
	}
	return p.getData()
}

func (*pluginData) isPluginData() {}

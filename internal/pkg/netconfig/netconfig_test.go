// Copyright 2020-2021 Buf Technologies, Inc.
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

package netconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	externalRemotes := []ExternalRemote{
		{
			Address: "goo",
		},
		{
			Address: "foo",
			Login: ExternalLogin{
				Token: "baz",
			},
		},
	}

	remoteProvider, err := NewRemoteProvider(externalRemotes)
	require.NoError(t, err)

	remote, ok := remoteProvider.GetRemote("foo")
	require.True(t, ok)
	require.Equal(t, "foo", remote.Address())
	login, ok := remote.Login()
	require.True(t, ok)
	require.Equal(t, "baz", login.Token())
	remote, ok = remoteProvider.GetRemote("goo")
	require.True(t, ok)
	require.Equal(t, "goo", remote.Address())
	_, ok = remote.Login()
	require.False(t, ok)
	_, ok = remoteProvider.GetRemote("hoo")
	require.False(t, ok)

	updatedRemoteProvider, err := remoteProvider.WithUpdatedRemote(
		"foo",
		"ban",
	)
	require.NoError(t, err)
	remote, ok = remoteProvider.GetRemote("foo")
	require.True(t, ok)
	require.Equal(t, "foo", remote.Address())
	login, ok = remote.Login()
	require.True(t, ok)
	require.Equal(t, "baz", login.Token())
	remote, ok = updatedRemoteProvider.GetRemote("foo")
	require.True(t, ok)
	require.Equal(t, "foo", remote.Address())
	login, ok = remote.Login()
	require.True(t, ok)
	require.Equal(t, "ban", login.Token())

	deleteRemoteProvider, ok := remoteProvider.WithoutRemote("foo")
	require.True(t, ok)
	_, ok = deleteRemoteProvider.GetRemote("foo")
	require.False(t, ok)
	_, ok = deleteRemoteProvider.GetRemote("goo")
	require.True(t, ok)
	deleteRemoteProvider, ok = deleteRemoteProvider.WithoutRemote("foo")
	require.False(t, ok)
	_, ok = deleteRemoteProvider.GetRemote("foo")
	require.False(t, ok)
	_, ok = deleteRemoteProvider.GetRemote("goo")
	require.True(t, ok)

	outputExternalRemotes := remoteProvider.ToExternalRemotes()
	require.Equal(
		t,
		[]ExternalRemote{
			{
				Address: "foo",
				Login: ExternalLogin{
					Token: "baz",
				},
			},
			{
				Address: "goo",
			},
		},
		outputExternalRemotes,
	)
}

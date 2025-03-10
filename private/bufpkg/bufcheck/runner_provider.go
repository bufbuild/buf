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

package bufcheck

import (
	"context"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/pkg/pluginrpcutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"pluginrpc.com/pluginrpc"
)

type runnerProvider struct {
	wasmRuntime        wasm.Runtime
	pluginKeyProvider  bufplugin.PluginKeyProvider
	pluginDataProvider bufplugin.PluginDataProvider
}

func newRunnerProvider(
	wasmRuntime wasm.Runtime,
	pluginKeyProvider bufplugin.PluginKeyProvider,
	pluginDataProvider bufplugin.PluginDataProvider,
) *runnerProvider {
	return &runnerProvider{
		wasmRuntime:        wasmRuntime,
		pluginKeyProvider:  pluginKeyProvider,
		pluginDataProvider: pluginDataProvider,
	}
}

func (r *runnerProvider) NewRunner(pluginConfig bufconfig.PluginConfig) (pluginrpc.Runner, error) {
	switch pluginConfig.Type() {
	case bufconfig.PluginConfigTypeLocal:
		return pluginrpcutil.NewLocalRunner(
			pluginConfig.Name(),
			pluginConfig.Args()...,
		), nil
	case bufconfig.PluginConfigTypeLocalWasm:
		return pluginrpcutil.NewLocalWasmRunner(
			r.wasmRuntime,
			pluginConfig.Name(),
			pluginConfig.Args()...,
		), nil
	case bufconfig.PluginConfigTypeRemoteWasm:
		return newRemoteWasmPluginRunner(
			r.wasmRuntime,
			r.pluginKeyProvider,
			r.pluginDataProvider,
			pluginConfig.Ref(),
			pluginConfig.Args(),
		)
	default:
		return nil, syserror.Newf("unknown PluginConfigType: %v", pluginConfig.Type())
	}
}

// *** PRIVATE ***

// remoteWasmPluginRunner is a Runner that runs a remote Wasm plugin.
//
// This is a wrapper around a pluginrpc.Runner that first resolves the Ref to
// a PluginKey using the PluginKeyProvider. It then loads the PluginData for
// the PluginKey using the PluginDataProvider. The PluginData is then used to
// create the pluginrpc.Runner. The Runner is only loaded once and is cached
// for future calls. However, if the Runner fails to load it will try to
// reload on the next call.
type remoteWasmPluginRunner struct {
	wasmRuntime        wasm.Runtime
	pluginKeyProvider  bufplugin.PluginKeyProvider
	pluginDataProvider bufplugin.PluginDataProvider
	pluginRef          bufparse.Ref
	pluginArgs         []string
	// lock protects runner.
	lock   sync.RWMutex
	runner pluginrpc.Runner
}

func newRemoteWasmPluginRunner(
	wasmRuntime wasm.Runtime,
	pluginKeyProvider bufplugin.PluginKeyProvider,
	pluginDataProvider bufplugin.PluginDataProvider,
	pluginRef bufparse.Ref,
	pluginArgs []string,
) (*remoteWasmPluginRunner, error) {
	return &remoteWasmPluginRunner{
		wasmRuntime:        wasmRuntime,
		pluginKeyProvider:  pluginKeyProvider,
		pluginDataProvider: pluginDataProvider,
		pluginRef:          pluginRef,
		pluginArgs:         pluginArgs,
	}, nil
}

func (r *remoteWasmPluginRunner) Run(ctx context.Context, env pluginrpc.Env) (retErr error) {
	delegate, err := r.loadRunnerOnce(ctx)
	if err != nil {
		return err
	}
	return delegate.Run(ctx, env)
}

func (r *remoteWasmPluginRunner) loadRunnerOnce(ctx context.Context) (pluginrpc.Runner, error) {
	r.lock.RLock()
	if r.runner != nil {
		r.lock.RUnlock()
		return r.runner, nil
	}
	r.lock.RUnlock()
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.runner == nil {
		runner, err := r.loadRunner(ctx)
		if err != nil {
			// The error isn't stored to avoid ctx cancellation issues. On the next call,
			// the runner will be reloaded instead of returning the error.
			return nil, err
		}
		r.runner = runner
	}
	return r.runner, nil
}

func (r *remoteWasmPluginRunner) loadRunner(ctx context.Context) (pluginrpc.Runner, error) {
	pluginKeys, err := r.pluginKeyProvider.GetPluginKeysForPluginRefs(ctx, []bufparse.Ref{r.pluginRef}, bufplugin.DigestTypeP1)
	if err != nil {
		return nil, err
	}
	if len(pluginKeys) != 1 {
		return nil, syserror.Newf("expected 1 PluginKey, got %d", len(pluginKeys))
	}
	// Load the data for the plugin now to ensure the context is valid for the entire operation.
	pluginDatas, err := r.pluginDataProvider.GetPluginDatasForPluginKeys(ctx, pluginKeys)
	if err != nil {
		return nil, err
	}
	if len(pluginDatas) != 1 {
		return nil, syserror.Newf("expected 1 PluginData, got %d", len(pluginDatas))
	}
	data := pluginDatas[0]
	// The program name is the FullName of the plugin.
	programName := r.pluginRef.FullName().String()
	return pluginrpcutil.NewWasmRunner(r.wasmRuntime, data.Data, programName, r.pluginArgs...), nil
}

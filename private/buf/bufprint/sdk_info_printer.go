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

package bufprint

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

type sdkInfoPrinter struct {
	writer io.Writer
}

func newSDKInfoPrinter(writer io.Writer) *sdkInfoPrinter {
	return &sdkInfoPrinter{
		writer: writer,
	}
}

func (p *sdkInfoPrinter) PrintSDKInfo(
	ctx context.Context,
	format Format,
	sdkInfo *registryv1alpha1.GetSDKInfoResponse,
) error {
	output := newOutputSDKInfo(sdkInfo)
	switch format {
	case FormatText:
		if _, err := fmt.Fprintf(
			p.writer,
			`Module
Owner:  %s
Name:   %s
Commit: %s

`,
			output.ModuleInfo.Owner,
			output.ModuleInfo.Name,
			output.ModuleInfo.Commit,
		); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(
			p.writer,
			`Plugin
Owner:    %s
Name:     %s
Version:  %s
Revision: %d

`,
			output.PluginInfo.Owner,
			output.PluginInfo.Name,
			output.PluginInfo.Version,
			output.PluginInfo.Revision,
		); err != nil {
			return err
		}
		_, err := fmt.Fprintf(p.writer, "Version: %s\n", output.Version)
		return err
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(output)
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

type outputSDKInfo struct {
	ModuleInfo outputSDKModuleInfo `json:"module,omitempty"`
	PluginInfo outputSDKPluginInfo `json:"plugin,omitempty"`
	Version    string              `json:"version,omitempty"`
}

type outputSDKModuleInfo struct {
	Owner            string    `json:"owner,omitempty"`
	Name             string    `json:"name,omitempty"`
	Commit           string    `json:"commit,omitempty"`
	CommitCreateTime time.Time `json:"commit_create_time,omitempty"`
}

type outputSDKPluginInfo struct {
	Owner    string `json:"owner,omitempty"`
	Name     string `json:"name,omitempty"`
	Version  string `json:"version,omitempty"`
	Revision uint32 `json:"revision,omitempty"`
}

func newOutputSDKInfo(sdkInfo *registryv1alpha1.GetSDKInfoResponse) outputSDKInfo {
	return outputSDKInfo{
		ModuleInfo: outputSDKModuleInfo{
			Owner:            sdkInfo.GetModuleInfo().GetOwner(),
			Name:             sdkInfo.GetModuleInfo().GetName(),
			Commit:           sdkInfo.GetModuleInfo().GetCommit(),
			CommitCreateTime: sdkInfo.GetModuleInfo().GetModuleCommitCreateTime().AsTime(),
		},
		PluginInfo: outputSDKPluginInfo{
			Owner:    sdkInfo.GetPluginInfo().GetOwner(),
			Name:     sdkInfo.GetPluginInfo().GetName(),
			Version:  sdkInfo.GetPluginInfo().GetVersion(),
			Revision: sdkInfo.GetPluginInfo().GetPluginRevision(),
		},
		Version: sdkInfo.GetSdkVersion(),
	}
}

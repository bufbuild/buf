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

package bufprint

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

type curatedPluginPrinter struct {
	writer io.Writer
}

func newCuratedPluginPrinter(writer io.Writer) *curatedPluginPrinter {
	return &curatedPluginPrinter{
		writer: writer,
	}
}

func (p *curatedPluginPrinter) PrintCuratedPlugin(_ context.Context, format Format, plugin *registryv1alpha1.CuratedPlugin) error {
	switch format {
	case FormatText:
		return p.printCuratedPluginsText(plugin)
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(
			registryCuratedPluginToOutputCuratedPlugin(plugin),
		)
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *curatedPluginPrinter) PrintCuratedPlugins(_ context.Context, format Format, nextPageToken string, plugins ...*registryv1alpha1.CuratedPlugin) error {
	switch format {
	case FormatText:
		return p.printCuratedPluginsText(plugins...)
	case FormatJSON:
		outputPlugins := make([]outputCuratedPlugin, 0, len(plugins))
		for _, plugin := range plugins {
			outputPlugins = append(outputPlugins, registryCuratedPluginToOutputCuratedPlugin(plugin))
		}
		return json.NewEncoder(p.writer).Encode(paginationWrapper{
			NextPage: nextPageToken,
			Results:  outputPlugins,
		})
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *curatedPluginPrinter) printCuratedPluginsText(plugins ...*registryv1alpha1.CuratedPlugin) error {
	if len(plugins) == 0 {
		return nil
	}
	return WithTabWriter(
		p.writer,
		[]string{
			"Owner",
			"Name",
			"Version",
			"Revision",
		},
		func(tabWriter TabWriter) error {
			for _, plugin := range plugins {
				if err := tabWriter.Write(
					plugin.Owner,
					plugin.Name,
					plugin.Version,
					strconv.FormatInt(int64(plugin.Revision), 10),
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

type outputCuratedPlugin struct {
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Revision    uint32 `json:"revision"`
	ImageDigest string `json:"image_digest"`
}

func registryCuratedPluginToOutputCuratedPlugin(plugin *registryv1alpha1.CuratedPlugin) outputCuratedPlugin {
	return outputCuratedPlugin{
		Owner:       plugin.Owner,
		Name:        plugin.Name,
		Version:     plugin.Version,
		Revision:    plugin.Revision,
		ImageDigest: plugin.ContainerImageDigest,
	}
}

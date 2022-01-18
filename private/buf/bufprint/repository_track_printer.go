// Copyright 2020-2022 Buf Technologies, Inc.
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
	"encoding/json"
	"fmt"
	"io"
	"time"

	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

type repositoryTrackPrinter struct {
	writer io.Writer
}

func newRepositoryTrackPrinter(
	writer io.Writer,
) *repositoryTrackPrinter {
	return &repositoryTrackPrinter{
		writer: writer,
	}
}

func (p *repositoryTrackPrinter) PrintRepositoryTrack(format Format, message *registryv1alpha1.RepositoryTrack) error {
	outputTrack := registryTrackToOutputTrack(message)
	switch format {
	case FormatText:
		return p.printRepositoryTracksText([]outputRepositoryTrack{outputTrack})
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(outputTrack)
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *repositoryTrackPrinter) PrintRepositoryTracks(
	format Format,
	nextPageToken string,
	messages ...*registryv1alpha1.RepositoryTrack,
) error {
	if len(messages) == 0 {
		return nil
	}
	tracks := make([]outputRepositoryTrack, len(messages))
	for i, repositoryTrack := range messages {
		tracks[i] = registryTrackToOutputTrack(repositoryTrack)
	}
	switch format {
	case FormatText:
		return p.printRepositoryTracksText(tracks)
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(paginationWrapper{
			NextPage: nextPageToken,
			Results:  tracks,
		})
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *repositoryTrackPrinter) printRepositoryTracksText(outputRepositoryTracks []outputRepositoryTrack) error {
	return WithTabWriter(
		p.writer,
		[]string{"Name", "Created"},
		func(tabWriter TabWriter) error {
			for _, track := range outputRepositoryTracks {
				if err := tabWriter.Write(
					track.Name,
					track.CreateTime.Format(time.RFC3339),
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

type outputRepositoryTrack struct {
	ID         string    `json:"id,omitempty"`
	Name       string    `json:"name,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty"`
}

func registryTrackToOutputTrack(repositoryTrack *registryv1alpha1.RepositoryTrack) outputRepositoryTrack {
	return outputRepositoryTrack{
		ID:         repositoryTrack.Id,
		Name:       repositoryTrack.Name,
		CreateTime: repositoryTrack.CreateTime.AsTime(),
	}
}

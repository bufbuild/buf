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

	"github.com/bufbuild/buf/private/pkg/protostat"
)

type statsPrinter struct {
	writer io.Writer
}

func newStatsPrinter(writer io.Writer) *statsPrinter {
	return &statsPrinter{
		writer: writer,
	}
}

func (p *statsPrinter) PrintStats(ctx context.Context, format Format, stats *protostat.Stats) error {
	switch format {
	case FormatText:
		return WithTabWriter(
			p.writer,
			[]string{
				"Files",
				"Packages",
				"Messages",
				"Fields",
				"Enums",
				"Enum Values",
				"Extensions",
				"Services",
				"Methods",
				"Files With Errors",
			},
			func(tabWriter TabWriter) error {
				return tabWriter.Write(
					strconv.Itoa(stats.NumFiles),
					strconv.Itoa(stats.NumPackages),
					strconv.Itoa(stats.NumMessages),
					strconv.Itoa(stats.NumFields),
					strconv.Itoa(stats.NumEnums),
					strconv.Itoa(stats.NumEnumValues),
					strconv.Itoa(stats.NumExtensions),
					strconv.Itoa(stats.NumServices),
					strconv.Itoa(stats.NumMethods),
					strconv.Itoa(stats.NumFilesWithSyntaxErrors),
				)
			},
		)
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(stats)
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

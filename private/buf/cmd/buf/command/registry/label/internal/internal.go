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

package internal

import (
	v1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
)

// GetLabelRefsForModuleFullNamesAndLabels takes a slice of moduleFullNames and a slice
// of labels and creates LabelRefs for all combinations of moduleFullNames and labels.
// This function is shared by `buf beta registry archive`and `buf beta registry unarchive`.
func GetLabelRefsForModuleFullNamesAndLabels(
	moduleFullNames []bufmodule.ModuleFullName,
	labels []string,
) []*v1.LabelRef {
	var labelRefs []*v1.LabelRef
	for _, moduleFullName := range moduleFullNames {
		for _, label := range labels {
			labelRefs = append(labelRefs, &v1.LabelRef{
				Value: &v1.LabelRef_Name_{
					Name: &v1.LabelRef_Name{
						Owner:  moduleFullName.Owner(),
						Module: moduleFullName.Name(),
						Label:  label,
					},
				},
			})
		}
	}
	return labelRefs
}

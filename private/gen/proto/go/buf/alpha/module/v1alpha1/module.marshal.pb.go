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

// Code generated by protoc-gen-buf-encoder-go. DO NOT EDIT.

package modulev1alpha1

import (
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/reflect/v1alpha1"
	proto "google.golang.org/protobuf/proto"
)

func (x *Module) MarshalWithDescriptorInfo() ([]byte, error) {
	bytes, err := proto.Marshal(x)
	if err != nil {
		return nil, err
	}
	descriptorInfoBytes, err := proto.Marshal(
		&v1alpha1.Reflector{
			DescriptorInfo: &v1alpha1.DescriptorInfo{
				ModuleInfo: &v1alpha1.ModuleInfo{
					Name: &v1alpha1.ModuleName{
						Remote:     "buf.build",
						Owner:      "bufbuild",
						Repository: "buf",
					},
					Commit: "53fae5b0c13c448c82f650a2e4da89da",
				},
				TypeName: "buf.alpha.module.v1alpha1.Module",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ModuleFile) MarshalWithDescriptorInfo() ([]byte, error) {
	bytes, err := proto.Marshal(x)
	if err != nil {
		return nil, err
	}
	descriptorInfoBytes, err := proto.Marshal(
		&v1alpha1.Reflector{
			DescriptorInfo: &v1alpha1.DescriptorInfo{
				ModuleInfo: &v1alpha1.ModuleInfo{
					Name: &v1alpha1.ModuleName{
						Remote:     "buf.build",
						Owner:      "bufbuild",
						Repository: "buf",
					},
					Commit: "53fae5b0c13c448c82f650a2e4da89da",
				},
				TypeName: "buf.alpha.module.v1alpha1.ModuleFile",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ModuleReference) MarshalWithDescriptorInfo() ([]byte, error) {
	bytes, err := proto.Marshal(x)
	if err != nil {
		return nil, err
	}
	descriptorInfoBytes, err := proto.Marshal(
		&v1alpha1.Reflector{
			DescriptorInfo: &v1alpha1.DescriptorInfo{
				ModuleInfo: &v1alpha1.ModuleInfo{
					Name: &v1alpha1.ModuleName{
						Remote:     "buf.build",
						Owner:      "bufbuild",
						Repository: "buf",
					},
					Commit: "53fae5b0c13c448c82f650a2e4da89da",
				},
				TypeName: "buf.alpha.module.v1alpha1.ModuleReference",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ModulePin) MarshalWithDescriptorInfo() ([]byte, error) {
	bytes, err := proto.Marshal(x)
	if err != nil {
		return nil, err
	}
	descriptorInfoBytes, err := proto.Marshal(
		&v1alpha1.Reflector{
			DescriptorInfo: &v1alpha1.DescriptorInfo{
				ModuleInfo: &v1alpha1.ModuleInfo{
					Name: &v1alpha1.ModuleName{
						Remote:     "buf.build",
						Owner:      "bufbuild",
						Repository: "buf",
					},
					Commit: "53fae5b0c13c448c82f650a2e4da89da",
				},
				TypeName: "buf.alpha.module.v1alpha1.ModulePin",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

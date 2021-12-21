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

package registryv1alpha1

import (
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/reflect/v1alpha1"
	proto "google.golang.org/protobuf/proto"
)

func (x *Token) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.registry.v1alpha1.Token",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *CreateTokenRequest) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.registry.v1alpha1.CreateTokenRequest",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *CreateTokenResponse) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.registry.v1alpha1.CreateTokenResponse",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *GetTokenRequest) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.registry.v1alpha1.GetTokenRequest",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *GetTokenResponse) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.registry.v1alpha1.GetTokenResponse",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ListTokensRequest) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.registry.v1alpha1.ListTokensRequest",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ListTokensResponse) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.registry.v1alpha1.ListTokensResponse",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *DeleteTokenRequest) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.registry.v1alpha1.DeleteTokenRequest",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *DeleteTokenResponse) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.registry.v1alpha1.DeleteTokenResponse",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

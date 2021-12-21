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

package auditv1alpha1

import (
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/reflect/v1alpha1"
	proto "google.golang.org/protobuf/proto"
)

func (x *ActionBufAlphaRegistryV1Alpha1DownloadInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1DownloadInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1GetImageInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1GetImageInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1CreateOrganizationInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1CreateOrganizationInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1DeleteOrganizationInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1DeleteOrganizationInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1DeleteOrganizationByNameInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1DeleteOrganizationByNameInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1AddOrganizationMemberInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1AddOrganizationMemberInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1UpdateOrganizationMemberInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1UpdateOrganizationMemberInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1RemoveOrganizationMemberInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1RemoveOrganizationMemberInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1UpdateOrganizationSettingsInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1UpdateOrganizationSettingsInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1CreatePluginInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1CreatePluginInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1DeletePluginInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1DeletePluginInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1GetTemplateVersionInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1GetTemplateVersionInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1CreateTemplateInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1CreateTemplateInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1DeleteTemplateInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1DeleteTemplateInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1CreateTemplateVersionInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1CreateTemplateVersionInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1PushInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1PushInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1GetReferenceByNameInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1GetReferenceByNameInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1CreateRepositoryBranchInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1CreateRepositoryBranchInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1ListRepositoryCommitsByBranchInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1ListRepositoryCommitsByBranchInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1ListRepositoryCommitsByReferenceInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1ListRepositoryCommitsByReferenceInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1GetRepositoryCommitByReferenceInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1GetRepositoryCommitByReferenceInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1GetRepositoryCommitBySequenceIDInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1GetRepositoryCommitBySequenceIDInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1CreateRepositoryTagInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1CreateRepositoryTagInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1CreateRepositoryByFullNameInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1CreateRepositoryByFullNameInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1DeleteRepositoryInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1DeleteRepositoryInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1DeleteRepositoryByFullNameInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1DeleteRepositoryByFullNameInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1DeprecateRepositoryByNameInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1DeprecateRepositoryByNameInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1UndeprecateRepositoryByNameInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1UndeprecateRepositoryByNameInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1GetModulePinsInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1GetModulePinsInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1GetLocalModulePinsInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1GetLocalModulePinsInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1SearchInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1SearchInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1CreateTokenInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1CreateTokenInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1DeleteTokenInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1DeleteTokenInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1CreateUserInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1CreateUserInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1ListUsersInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1ListUsersInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1DeactivateUserInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1DeactivateUserInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1UpdateUserServerRoleInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1UpdateUserServerRoleInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryinternalV1Alpha1CreatePluginVersionInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryinternalV1Alpha1CreatePluginVersionInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryinternalV1Alpha1CreatePluginVersionMetadataInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryinternalV1Alpha1CreatePluginVersionMetadataInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryinternalV1Alpha1DeletePluginVersionInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryinternalV1Alpha1DeletePluginVersionInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1SetRepositoryContributorInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1SetRepositoryContributorInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1SetPluginContributorInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1SetPluginContributorInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1SetTemplateContributorInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1SetTemplateContributorInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *ActionBufAlphaRegistryV1Alpha1CreateRepositoryTrackInfo) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.ActionBufAlphaRegistryV1Alpha1CreateRepositoryTrackInfo",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *Event) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.Event",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *UserActor) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.UserActor",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *UserObject) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.UserObject",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *OrganizationObject) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.OrganizationObject",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *RepositoryObject) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.RepositoryObject",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *PluginObject) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.PluginObject",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *TemplateObject) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.TemplateObject",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *TokenObject) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.TokenObject",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

func (x *Object) MarshalWithDescriptorInfo() ([]byte, error) {
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
				TypeName: "buf.alpha.audit.v1alpha1.Object",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}

// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufwire

import (
	"context"
	"io"
	"sort"
	"testing"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestExcludePathsForModule(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description              string
		moduleContent            map[string][]byte
		targetPaths              []string
		excludePaths             []string
		expectedFilesAfterFilter []string
	}{
		{
			description: "target nil exclude nil",
			moduleContent: map[string][]byte{
				"dir/foo.proto": nil,
				"dir/bar.proto": nil,
			},
			targetPaths:  nil,
			excludePaths: nil,
			expectedFilesAfterFilter: []string{
				"dir/foo.proto",
				"dir/bar.proto",
			},
		},
		{
			description: "target empty exclude nil",
			moduleContent: map[string][]byte{
				"dir/foo.proto": nil,
				"dir/bar.proto": nil,
			},
			targetPaths:  make([]string, 0),
			excludePaths: nil,
			expectedFilesAfterFilter: []string{
				"dir/foo.proto",
				"dir/bar.proto",
			},
		},
		{
			description: "target empty with exclude",
			moduleContent: map[string][]byte{
				"dir/foo.proto": nil,
				"dir/bar.proto": nil,
			},
			targetPaths: make([]string, 0),
			excludePaths: []string{
				"dir/bar.proto",
			},
			expectedFilesAfterFilter: []string{
				"dir/foo.proto",
			},
		},
		{
			description: "target nil with exclude",
			moduleContent: map[string][]byte{
				"dir/foo.proto": nil,
				"dir/bar.proto": nil,
			},
			targetPaths: nil,
			excludePaths: []string{
				"dir/bar.proto",
			},
			expectedFilesAfterFilter: []string{
				"dir/foo.proto",
			},
		},
		{
			description: "target paths with exclude nil",
			moduleContent: map[string][]byte{
				"dir/foo.proto":  nil,
				"dir/bar.proto":  nil,
				"dir2/foo.proto": nil,
				"dir2/bar.proto": nil,
			},
			targetPaths: []string{
				"dir2",
				"dir/bar.proto",
			},
			excludePaths: nil,
			expectedFilesAfterFilter: []string{
				"dir/bar.proto",
				"dir2/foo.proto",
				"dir2/bar.proto",
			},
		},
		{
			description: "target paths with exclude paths",
			moduleContent: map[string][]byte{
				"dir/foo/bar.proto":  nil,
				"dir/foo/baz.proto":  []byte(""),
				"dir/foo/qux.proto":  []byte(""),
				"dir/foo2/bar.proto": []byte(""),
				"dir/foo2/baz.proto": []byte(""),
				"dir2/bar.proto":     nil,
				"dir2/baz.proto":     nil,
			},
			targetPaths: []string{
				"dir/",
				"dir2/baz.proto",
			},
			excludePaths: []string{
				"dir/foo2",
				"dir/foo/qux.proto",
			},
			expectedFilesAfterFilter: []string{
				"dir/foo/bar.proto",
				"dir/foo/baz.proto",
				"dir2/baz.proto",
			},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			moduleConfigReader := NewModuleConfigReader(
				zap.NewNop(),
				storageos.NewProvider(),
				&fetchReader{
					fileContent: testcase.moduleContent,
				},
				nil,
			)
			someModuleRef, err := buffetch.NewModuleRefParser(zap.NewNop()).GetModuleRef(context.Background(), "buf.build/foo/bar")
			require.NoError(t, err)
			moduleConfigSet, err := moduleConfigReader.GetModuleConfigSet(
				context.Background(),
				nil,
				someModuleRef, // the fake module fetcher doesn't care what ref this is
				"",
				testcase.targetPaths,
				testcase.excludePaths,
				true,
			)
			require.NoError(t, err)
			require.Len(t, moduleConfigSet.ModuleConfigs(), 1)
			moduleConfig := moduleConfigSet.ModuleConfigs()[0]
			require.NotNil(t, moduleConfig)
			module := moduleConfig.Module()
			require.NotNil(t, module)
			targetFileInfos, err := module.TargetFileInfos(context.Background())
			require.NoError(t, err)
			targetFilePaths := make([]string, len(targetFileInfos))
			for i := range targetFileInfos {
				targetFilePaths[i] = targetFileInfos[i].Path()
			}
			sort.Strings(targetFilePaths)
			sort.Strings(testcase.expectedFilesAfterFilter)
			require.Equal(t, testcase.expectedFilesAfterFilter, targetFilePaths)
		})
	}
}

func TestModuleIDsFromGetModuleConfigSet(t *testing.T) {
	t.Parallel()
	workspaceConfigContent := `
version: v1
directories:
  - dir1
  - dir2
`
	dir1ConfigContent := `
version: v1
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
`
	dir2ConfigContent := `
version: v1
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
`
	testcases := []struct {
		description                string
		input                      string
		content                    map[string][]byte
		subDir                     string
		expectedIDs                []string
		expectedWorkspaceModuleIds []string
	}{
		{
			description: "git",
			input:       ".git",
			content:     map[string][]byte{},
			expectedIDs: []string{".git"},
		},
		{
			description: "git_target_the_entire_workspace",
			input:       ".git",
			content: map[string][]byte{
				"buf.work.yaml":  []byte(workspaceConfigContent),
				"dir1/buf.yaml":  []byte(dir1ConfigContent),
				"dir1/foo.proto": []byte("message Foo {}"),
				"dir2/buf.yaml":  []byte(dir2ConfigContent),
				"dir2/bar.proto": []byte("message Bar {}"),
			},
			// the module is for the entire workspace, and ID should be the ref's ID itself
			expectedIDs:                []string{".git"},
			expectedWorkspaceModuleIds: []string{".git:dir1", ".git:dir2"},
		},
		{
			description: "git_target_the_entire_workspace",
			input:       ".git#branch=main",
			content: map[string][]byte{
				"buf.work.yaml":  []byte(workspaceConfigContent),
				"dir1/buf.yaml":  []byte(dir1ConfigContent),
				"dir1/foo.proto": []byte("message Foo {}"),
				"dir2/buf.yaml":  []byte(dir2ConfigContent),
				"dir2/bar.proto": []byte("message Bar {}"),
			},
			expectedIDs:                []string{".git#name=main"},
			expectedWorkspaceModuleIds: []string{".git#name=main:dir1", ".git#name=main:dir2"},
		},
		{
			description: "git_target_a_module_in_workspace",
			input:       ".git#subdir=dir1",
			content: map[string][]byte{
				"buf.work.yaml":  []byte(workspaceConfigContent),
				"dir1/buf.yaml":  []byte(dir1ConfigContent),
				"dir1/foo.proto": []byte("message Foo {}"),
				"dir2/buf.yaml":  []byte(dir2ConfigContent),
				"dir2/bar.proto": []byte("message Bar {}"),
			},
			subDir: "dir1",
			// the module is for the entire workspace, and ID should be the ref's ID itself
			expectedIDs:                []string{".git:dir1"},
			expectedWorkspaceModuleIds: []string{".git:dir1", ".git:dir2"},
		},
		{
			description: "non_module_directory",
			input:       "foo",
			content: map[string][]byte{
				"baz.proto": []byte("message Baz {}"),
			},
			expectedIDs: []string{"foo"},
		},
		{
			description: "module_directory",
			input:       "foo",
			content: map[string][]byte{
				"buf.yaml":  []byte(dir1ConfigContent),
				"baz.proto": []byte("message Baz {}"),
			},
			expectedIDs: []string{"foo"},
		},
		{
			description: "workspace_directory",
			input:       "dir",
			content: map[string][]byte{
				"buf.work.yaml":  []byte(workspaceConfigContent),
				"dir1/buf.yaml":  []byte(dir1ConfigContent),
				"dir1/foo.proto": []byte("message Foo {}"),
				"dir2/buf.yaml":  []byte(dir2ConfigContent),
				"dir2/bar.proto": []byte("message Bar {}"),
			},
			expectedIDs:                []string{"dir"},
			expectedWorkspaceModuleIds: []string{"dir:dir1", "dir:dir2"},
		},
		{
			description: "module_directory_in_workspace",
			input:       "foo",
			content: map[string][]byte{
				"buf.work.yaml":  []byte(workspaceConfigContent),
				"dir1/buf.yaml":  []byte(dir1ConfigContent),
				"dir1/foo.proto": []byte("message Foo {}"),
				"dir2/buf.yaml":  []byte(dir2ConfigContent),
				"dir2/bar.proto": []byte("message Bar {}"),
			},
			subDir:                     "dir1",
			expectedIDs:                []string{"foo:dir1"},
			expectedWorkspaceModuleIds: []string{"foo:dir1", "foo:dir2"},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			nopLogger := zap.NewNop()
			readBucket, err := storagemem.NewReadBucket(testcase.content)
			require.NoError(t, err)
			moduleConfigReader := NewModuleConfigReader(
				nopLogger,
				storageos.NewProvider(),
				&fetchReader{
					readBucketCloserWithTerminateFileProvider: readBucketCloserWithTerminateFileProvider{
						ReadBucketCloser: storage.NopReadBucketCloser(readBucket),
						subDirPath:       testcase.subDir,
					},
					fileContent: testcase.content,
				},
				nil,
			)
			ref, err := buffetch.NewRefParser(nopLogger).GetSourceOrModuleRef(context.Background(), testcase.input)
			require.NoError(t, err)
			moduleConfigSet, err := moduleConfigReader.GetModuleConfigSet(
				context.Background(),
				nil,
				ref,
				"",
				nil,
				nil,
				true,
			)
			require.NoError(t, err)
			require.NotNil(t, moduleConfigSet)
			require.Len(t, moduleConfigSet.ModuleConfigs(), len(testcase.expectedIDs))
			for i, expexpectedID := range testcase.expectedIDs {
				require.Equal(t, expexpectedID, moduleConfigSet.ModuleConfigs()[i].Module().ID())
			}
			if len(testcase.expectedWorkspaceModuleIds) > 0 {
				workspace := moduleConfigSet.Workspace()
				require.NotNil(t, workspace)
				require.Len(t, workspace.GetModules(), len(testcase.expectedWorkspaceModuleIds))
				for i, expectedWorkspaceModuleID := range testcase.expectedWorkspaceModuleIds {
					require.Equal(t, expectedWorkspaceModuleID, workspace.GetModules()[i].ID())
				}
			}
		})
	}
}

type fetchReader struct {
	fileContent map[string][]byte
	readBucketCloserWithTerminateFileProvider
}

func (r *fetchReader) GetModule(
	ctx context.Context,
	container app.EnvStdinContainer,
	moduleRef buffetch.ModuleRef,
) (bufmodule.Module, error) {
	moduleBucket, err := storagemem.NewReadBucket(
		r.fileContent,
	)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleForBucket(
		context.Background(),
		moduleBucket,
	)
}

func (r *fetchReader) GetImageFile(
	ctx context.Context,
	container app.EnvStdinContainer,
	imageRef buffetch.ImageRef,
) (io.ReadCloser, error) {
	return nil, nil
}

func (r *fetchReader) GetSourceBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceRef buffetch.SourceRef,
	options ...buffetch.GetSourceBucketOption,
) (buffetch.ReadBucketCloserWithTerminateFileProvider, error) {
	return &r.readBucketCloserWithTerminateFileProvider, nil
}

type readBucketCloserWithTerminateFileProvider struct {
	storage.ReadBucketCloser
	relativeRootPath string
	subDirPath       string
}

func (r *readBucketCloserWithTerminateFileProvider) TerminateFileProvider() buffetch.TerminateFileProvider {
	return r
}

func (r *readBucketCloserWithTerminateFileProvider) GetTerminateFiles() []buffetch.TerminateFile {
	// Logic on terminateFiles depend on the call site, see terminateFilesOnOS in private/buf/buffetch/internal/reader.go.
	return nil
}

func (r *readBucketCloserWithTerminateFileProvider) RelativeRootPath() string {
	return r.relativeRootPath
}

func (r *readBucketCloserWithTerminateFileProvider) SubDirPath() string {
	return r.subDirPath
}

func (r *readBucketCloserWithTerminateFileProvider) SetSubDirPath(subDirPath string) {
	r.subDirPath = subDirPath
}

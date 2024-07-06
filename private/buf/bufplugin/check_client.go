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

package bufplugin

import (
	"context"
	"fmt"

	checkv1beta1 "buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go/buf/plugin/check/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/gen/proto/pluginrpc/buf/plugin/check/v1beta1/v1beta1pluginrpc"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/pluginrpc-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
)

const (
	checkPageSize     = 250
	listRulesPageSize = 250
)

type checkClient struct {
	logger      *zap.Logger
	client      pluginrpc.Client
	programName string
}

func newCheckClient(
	logger *zap.Logger,
	runner pluginrpc.Runner,
	programName string,
) *checkClient {
	return &checkClient{
		logger:      logger,
		client:      pluginrpc.NewClient(runner, programName),
		programName: programName,
	}
}

func (c *checkClient) Check(
	ctx context.Context,
	image bufimage.Image,
	againstImage bufimage.Image,
) error {
	checkServiceClient, err := c.newCheckServiceClient()
	if err != nil {
		return err
	}
	protoRules, err := listProtoRules(ctx, checkServiceClient)
	if err != nil {
		return err
	}
	if againstImage == nil {
		for _, protoRule := range protoRules {
			switch protoRule.GetType() {
			case checkv1beta1.RuleType_RULE_TYPE_LINT:
			case checkv1beta1.RuleType_RULE_TYPE_BREAKING:
				// Pretty janky.
				return fmt.Errorf("plugin %q had breaking change rule %q and we are not running breaking change checks currently", c.programName, protoRule.GetId())
			default:
				return fmt.Errorf("unknown checkv1beta1.RuleType: %v", protoRule.GetType())
			}
		}
	}

	var allFileAnnotations []bufanalysis.FileAnnotation
	var protoregistryFiles *protoregistry.Files
	for i := 0; i < len(protoRules); i += checkPageSize {
		start := i
		end := start + checkPageSize
		if end > len(protoRules) {
			end = len(protoRules)
		}
		iProtoRules := protoRules[start:end]

		files := imageToProtoFiles(image)
		var againstFiles []*checkv1beta1.File
		if againstImage != nil {
			againstFiles = imageToProtoFiles(againstImage)
		}

		response, err := checkServiceClient.Check(
			ctx,
			&checkv1beta1.CheckRequest{
				RuleIds:      slicesext.Map(iProtoRules, func(protoRule *checkv1beta1.Rule) string { return protoRule.GetId() }),
				Files:        files,
				AgainstFiles: againstFiles,
			},
		)
		if err != nil {
			return err
		}
		if protoAnnotations := response.GetAnnotations(); len(protoAnnotations) > 0 {
			if protoregistryFiles == nil {
				protoregistryFiles, err = protodesc.NewFiles(bufimage.ImageToFileDescriptorSet(image))
				if err != nil {
					return err
				}
			}
			fileAnnotatations, err := protoAnnotationsToFileAnnotations(
				protoregistryFiles,
				protoAnnotations,
			)
			if err != nil {
				return err
			}
			allFileAnnotations = append(allFileAnnotations, fileAnnotatations...)
		}
	}
	if len(allFileAnnotations) > 0 {
		return bufanalysis.NewFileAnnotationSet(allFileAnnotations...)
	}
	return nil
}

func (c *checkClient) newCheckServiceClient() (v1beta1pluginrpc.CheckServiceClient, error) {
	return v1beta1pluginrpc.NewCheckServiceClient(c.client)
}

func listProtoRules(ctx context.Context, checkServiceClient v1beta1pluginrpc.CheckServiceClient) ([]*checkv1beta1.Rule, error) {
	var protoRules []*checkv1beta1.Rule
	var pageToken string
	for {
		response, err := checkServiceClient.ListRules(
			ctx,
			&checkv1beta1.ListRulesRequest{
				PageSize:  listRulesPageSize,
				PageToken: pageToken,
			},
		)
		if err != nil {
			return nil, err
		}
		protoRules = append(protoRules, response.GetRules()...)
		pageToken = response.GetNextPageToken()
		if pageToken == "" {
			break
		}
	}
	return protoRules, nil
}

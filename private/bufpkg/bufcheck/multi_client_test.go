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

package bufcheck

import (
	"context"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/bufplugin-go/check"
	"github.com/bufbuild/bufplugin-go/check/checktest"
	"github.com/bufbuild/bufplugin-go/check/checkutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	fieldLowerSnakeCaseRuleID = "FIELD_LOWER_SNAKE_CASE"
	timestampSuffixRuleID     = "TIMESTAMP_SUFFIX"
	timestampSuffixOptionKey  = "timestamp_suffix"
	defaultTimestampSuffix    = "_time"
)

var (
	fieldLowerSnakeCaseRuleSpec = &check.RuleSpec{
		ID:        fieldLowerSnakeCaseRuleID,
		IsDefault: true,
		Purpose:   "Checks that all field names are lower_snake_case.",
		Type:      check.RuleTypeLint,
		Handler:   checkutil.NewFieldRuleHandler(checkFieldLowerSnakeCase),
	}

	fieldLowerSnakeCaseSpec = &check.Spec{
		Rules: []*check.RuleSpec{
			fieldLowerSnakeCaseRuleSpec,
		},
	}

	timestampSuffixRuleSpec = &check.RuleSpec{
		ID:        timestampSuffixRuleID,
		IsDefault: true,
		Purpose:   `Checks that all google.protobuf.Timestamps end in a specific suffix (default is "_time").`,
		Type:      check.RuleTypeLint,
		Handler:   checkutil.NewFieldRuleHandler(checkTimestampSuffix),
	}

	timestampSuffixSpec = &check.Spec{
		Rules: []*check.RuleSpec{
			timestampSuffixRuleSpec,
		},
	}
)

func TestMultiClientSimple(t *testing.T) {
	t.Parallel()

	testMultiClientSimple(t, false)
}

func TestMultiClientSimpleCacheRules(t *testing.T) {
	t.Parallel()

	testMultiClientSimple(t, true)
}

func testMultiClientSimple(t *testing.T, cacheRules bool) {
	ctx := context.Background()

	requestSpec := &checktest.RequestSpec{
		Files: &checktest.ProtoFileSpec{
			DirPaths:  []string{"testdata/multi_client/simple"},
			FilePaths: []string{"simple.proto"},
		},
	}
	request, err := requestSpec.ToRequest(ctx)
	require.NoError(t, err)

	var clientOptions []check.ClientOption
	if cacheRules {
		clientOptions = append(clientOptions, check.ClientWithCacheRulesAndCategories())
	}
	fieldLowerSnakeCaseClient, err := check.NewClientForSpec(fieldLowerSnakeCaseSpec, clientOptions...)
	require.NoError(t, err)
	timestampSuffixClient, err := check.NewClientForSpec(timestampSuffixSpec, clientOptions...)
	require.NoError(t, err)
	emptyOptions, err := check.NewOptions(nil)
	require.NoError(t, err)
	multiClient := newMultiClient(
		zap.NewNop(),
		[]*checkClientSpec{
			newCheckClientSpec("buf-plugin-field-lower-snake-case", fieldLowerSnakeCaseClient, emptyOptions),
			newCheckClientSpec("buf-plugin-timestamp-suffix", timestampSuffixClient, emptyOptions),
		},
	)

	rules, _, err := multiClient.ListRulesAndCategories(ctx)
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{
			fieldLowerSnakeCaseRuleID,
			timestampSuffixRuleID,
		},
		slicesext.Map(rules, Rule.ID),
	)
	annotations, err := multiClient.Check(ctx, request)
	require.NoError(t, err)
	checktest.AssertAnnotationsEqual(
		t,
		[]checktest.ExpectedAnnotation{
			{
				RuleID: fieldLowerSnakeCaseRuleID,
				Location: &checktest.ExpectedLocation{
					FileName:    "simple.proto",
					StartLine:   10,
					StartColumn: 2,
					EndLine:     10,
					EndColumn:   23,
				},
			},
			{
				RuleID: timestampSuffixRuleID,
				Location: &checktest.ExpectedLocation{
					FileName:    "simple.proto",
					StartLine:   9,
					StartColumn: 2,
					EndLine:     9,
					EndColumn:   50,
				},
			},
		},
		slicesext.Map(
			annotations,
			func(annotation *annotation) check.Annotation {
				return annotation
			},
		),
	)
}

func TestMultiClientCannotHaveOverlappingRules(t *testing.T) {
	t.Parallel()

	fieldLowerSnakeCaseClient, err := check.NewClientForSpec(fieldLowerSnakeCaseSpec)
	require.NoError(t, err)
	emptyOptions, err := check.NewOptions(nil)
	require.NoError(t, err)
	multiClient := newMultiClient(
		zap.NewNop(),
		[]*checkClientSpec{
			newCheckClientSpec("buf-plugin-field-lower-snake-case", fieldLowerSnakeCaseClient, emptyOptions),
			newCheckClientSpec("buf-plugin-field-lower-snake-case", fieldLowerSnakeCaseClient, emptyOptions),
		},
	)

	_, _, err = multiClient.ListRulesAndCategories(context.Background())
	duplicateRuleOrCategoryError := &duplicateRuleOrCategoryError{}
	require.ErrorAs(t, err, &duplicateRuleOrCategoryError)
	require.Equal(t, []string{fieldLowerSnakeCaseRuleID}, duplicateRuleOrCategoryError.duplicateIDs())
}

func TestMultiClientCannotHaveOverlappingCategories(t *testing.T) {
	t.Parallel()

	client1Spec := &check.Spec{
		Rules: []*check.RuleSpec{
			{
				ID:          timestampSuffixRuleID,
				IsDefault:   true,
				CategoryIDs: []string{"FOO"},
				Purpose:     `Checks that all google.protobuf.Timestamps end in a specific suffix (default is "_time").`,
				Type:        check.RuleTypeLint,
				Handler:     checkutil.NewFieldRuleHandler(checkTimestampSuffix),
			},
		},
		Categories: []*check.CategorySpec{
			{
				ID:      "FOO",
				Purpose: "Checks foo.",
			},
		},
	}
	client2Spec := &check.Spec{
		Rules: []*check.RuleSpec{
			{
				ID:          fieldLowerSnakeCaseRuleID,
				IsDefault:   true,
				CategoryIDs: []string{"FOO"},
				Purpose:     "Checks that all field names are lower_snake_case.",
				Type:        check.RuleTypeLint,
				Handler:     checkutil.NewFieldRuleHandler(checkFieldLowerSnakeCase),
			},
		},
		Categories: []*check.CategorySpec{
			{
				ID:      "FOO",
				Purpose: "Checks foo.",
			},
		},
	}

	client1, err := check.NewClientForSpec(client1Spec)
	require.NoError(t, err)
	client2, err := check.NewClientForSpec(client2Spec)
	require.NoError(t, err)
	emptyOptions, err := check.NewOptions(nil)
	require.NoError(t, err)
	multiClient := newMultiClient(
		zap.NewNop(),
		[]*checkClientSpec{
			newCheckClientSpec("buf-plugin-1", client1, emptyOptions),
			newCheckClientSpec("buf-plugin-2", client2, emptyOptions),
		},
	)

	_, _, err = multiClient.ListRulesAndCategories(context.Background())
	duplicateRuleOrCategoryError := &duplicateRuleOrCategoryError{}
	require.ErrorAs(t, err, &duplicateRuleOrCategoryError)
	require.Equal(t, []string{"FOO"}, duplicateRuleOrCategoryError.duplicateIDs())
}

func checkFieldLowerSnakeCase(
	_ context.Context,
	responseWriter check.ResponseWriter,
	_ check.Request,
	fieldDescriptor protoreflect.FieldDescriptor,
) error {
	fieldName := string(fieldDescriptor.Name())
	fieldNameToLowerSnakeCase := stringutil.ToLowerSnakeCase(fieldName)
	if fieldName != fieldNameToLowerSnakeCase {
		responseWriter.AddAnnotation(
			check.WithMessagef("Field name %q should be lower_snake_case, such as %q.", fieldName, fieldNameToLowerSnakeCase),
			check.WithDescriptor(fieldDescriptor),
		)
	}
	return nil
}

func checkTimestampSuffix(
	_ context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
	fieldDescriptor protoreflect.FieldDescriptor,
) error {
	timestampSuffix := defaultTimestampSuffix
	timestampSuffixOptionValue, err := check.GetStringValue(request.Options(), timestampSuffixOptionKey)
	if err != nil {
		return err
	}
	if timestampSuffixOptionValue != "" {
		timestampSuffix = timestampSuffixOptionValue
	}

	fieldDescriptorType := fieldDescriptor.Message()
	if fieldDescriptorType == nil {
		return nil
	}
	if string(fieldDescriptorType.FullName()) != "google.protobuf.Timestamp" {
		return nil
	}
	if !strings.HasSuffix(string(fieldDescriptor.Name()), timestampSuffix) {
		responseWriter.AddAnnotation(
			check.WithMessagef("Fields of type google.protobuf.Timestamp must end in %q but field name was %q.", timestampSuffix, string(fieldDescriptor.Name())),
			check.WithDescriptor(fieldDescriptor),
		)
	}
	return nil
}

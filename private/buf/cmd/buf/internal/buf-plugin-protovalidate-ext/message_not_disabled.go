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

package protovalidateextplugin

import (
	"context"

	"github.com/bufbuild/bufplugin-go/check"
	"github.com/bufbuild/bufplugin-go/check/checkutil"
	"github.com/bufbuild/protovalidate-go/resolver"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	// MessageNotDisabled is the Rule ID of message not disabled rule.
	MessageNotDisabled = "MESSAGE_NOT_DISABLED"
)

var (
	// MessageNotDisabledRuleSpec is the RuleSpec for the ID field validation rule.
	MessageNotDisabledRuleSpec = &check.RuleSpec{
		ID:             MessageNotDisabled,
		CategoryIDs:    nil,
		IsDefault:      true,
		Purpose:        `Checks that no message has (buf.validate.message).disabled set`,
		Type:           check.RuleTypeLint,
		ReplacementIDs: nil,
		Handler:        checkutil.NewMessageRuleHandler(checkMessageNotDisabled),
	}
)

func checkMessageNotDisabled(
	_ context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
	messageDescriptor protoreflect.MessageDescriptor,
) error {
	constraints := resolver.DefaultResolver{}.ResolveMessageConstraints(messageDescriptor)
	if constraints.GetDisabled() {
		responseWriter.AddAnnotation(
			check.WithMessagef("%s has (buf.validate.message).disabled set to true", string(messageDescriptor.Name())),
			check.WithDescriptor(messageDescriptor),
		)
	}
	return nil
}

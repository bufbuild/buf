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

package rpcextplugin

import (
	"context"
	"strings"

	"github.com/bufbuild/bufplugin-go/check"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	pageTokenFieldNameOptionKey = "page_token_field_name"
	pageRPCPrefixOptionKey      = "page_rpc_prefix"
	defaultPageTokenFieldName   = "page_token"
)

var (
	defaultPageRPCPrefixes = []string{"List"}
)

func checkPageRequestHasToken(
	_ context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
	messageDescriptor protoreflect.MessageDescriptor,
) error {
	requestName := string(messageDescriptor.Name())
	if !strings.HasSuffix(requestName, "Request") {
		return nil
	}
	pageRPCPrefixes := defaultPageRPCPrefixes
	pageRPCPrefixesOptionValue, err := check.GetStringSliceValue(request.Options(), pageRPCPrefixOptionKey)
	if err != nil {
		return err
	}
	if len(pageRPCPrefixesOptionValue) > 0 {
		pageRPCPrefixes = pageRPCPrefixesOptionValue
	}
	var isPaginationPRC bool
	for _, prefx := range pageRPCPrefixes {
		if strings.HasPrefix(requestName, prefx) {
			isPaginationPRC = true
			break
		}
	}
	if !isPaginationPRC {
		return nil
	}
	pageTokenFieldName := defaultPageTokenFieldName
	pageTokenFieldNameOptionValue, err := check.GetStringValue(request.Options(), pageTokenFieldNameOptionKey)
	if err != nil {
		return err
	}
	if pageTokenFieldNameOptionValue != "" {
		pageTokenFieldName = pageTokenFieldNameOptionValue
	}
	fields := messageDescriptor.Fields()
	for i := 0; i < fields.Len(); i++ {
		fieldName := string(fields.Get(i).Name())
		if fieldName == pageTokenFieldName {
			return nil
		}
	}
	responseWriter.AddAnnotation(
		check.WithDescriptor(messageDescriptor),
		check.WithMessagef("%q is a pagination request without a page token field named %q", requestName, pageTokenFieldName),
	)
	return nil
}

func checkPageResponseHasToken(
	_ context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
	messageDescriptor protoreflect.MessageDescriptor,
) error {
	responseName := string(messageDescriptor.Name())
	if !strings.HasSuffix(responseName, "Response") {
		return nil
	}
	pageRPCPrefixes := defaultPageRPCPrefixes
	pageRPCPrefixesOptionValue, err := check.GetStringSliceValue(request.Options(), pageRPCPrefixOptionKey)
	if err != nil {
		return err
	}
	if len(pageRPCPrefixesOptionValue) > 0 {
		pageRPCPrefixes = pageRPCPrefixesOptionValue
	}
	var isPaginationPRC bool
	for _, prefx := range pageRPCPrefixes {
		if strings.HasPrefix(responseName, prefx) {
			isPaginationPRC = true
			break
		}
	}
	if !isPaginationPRC {
		return nil
	}
	pageTokenFieldName := defaultPageTokenFieldName
	pageTokenFieldNameOptionValue, err := check.GetStringValue(request.Options(), pageTokenFieldNameOptionKey)
	if err != nil {
		return err
	}
	if pageTokenFieldNameOptionValue != "" {
		pageTokenFieldName = pageTokenFieldNameOptionValue
	}
	fields := messageDescriptor.Fields()
	for i := 0; i < fields.Len(); i++ {
		fieldName := string(fields.Get(i).Name())
		if fieldName == pageTokenFieldName {
			return nil
		}
	}
	responseWriter.AddAnnotation(
		check.WithDescriptor(messageDescriptor),
		check.WithMessagef("%q is a pagination response without a page token field named %q", responseName, pageTokenFieldName),
	)
	return nil
}
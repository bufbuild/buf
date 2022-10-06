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

// Code generated by protoc-gen-go-apiclientconnect. DO NOT EDIT.

package auditv1alpha1apiclientconnect

import (
	context "context"
	auditv1alpha1connect "github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/audit/v1alpha1/auditv1alpha1connect"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/audit/v1alpha1"
	connect_go "github.com/bufbuild/connect-go"
	zap "go.uber.org/zap"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type auditServiceClient struct {
	logger *zap.Logger
	client auditv1alpha1connect.AuditServiceClient
}

// ListAuditedEvents lists audited events recorded in the BSR instance.
func (s *auditServiceClient) ListAuditedEvents(
	ctx context.Context,
	pageSize uint32,
	pageToken string,
	reverse bool,
	start *timestamppb.Timestamp,
	end *timestamppb.Timestamp,
) (events []*v1alpha1.Event, nextPageToken string, _ error) {
	response, err := s.client.ListAuditedEvents(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.ListAuditedEventsRequest{
				PageSize:  pageSize,
				PageToken: pageToken,
				Reverse:   reverse,
				Start:     start,
				End:       end,
			}),
	)
	if err != nil {
		return nil, "", err
	}
	return response.Msg.Events, response.Msg.NextPageToken, nil
}

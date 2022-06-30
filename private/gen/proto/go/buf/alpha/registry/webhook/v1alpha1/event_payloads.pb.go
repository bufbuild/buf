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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        (unknown)
// source: buf/alpha/registry/webhook/v1alpha1/event_payloads.proto

package webhookv1alpha1

import (
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// EventRequest is the request payload that will be sent to the customer that is
// subscribed to webhook events in the BSR. This request message serves both as
// (1) documentation of the payload that will be sent in a JSON blob, or (2)
// using this in an rpc signature in your own proto file.
type EventRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The event that was triggered.
	Event v1alpha1.WebhookEvent `protobuf:"varint,1,opt,name=event,proto3,enum=buf.alpha.registry.v1alpha1.WebhookEvent" json:"event,omitempty"`
	// The event payload depending on which event was triggered.
	//
	// Types that are assignable to Payload:
	//	*EventRequest_RepositoryPush
	Payload isEventRequest_Payload `protobuf_oneof:"payload"`
}

func (x *EventRequest) Reset() {
	*x = EventRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventRequest) ProtoMessage() {}

func (x *EventRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventRequest.ProtoReflect.Descriptor instead.
func (*EventRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_rawDescGZIP(), []int{0}
}

func (x *EventRequest) GetEvent() v1alpha1.WebhookEvent {
	if x != nil {
		return x.Event
	}
	return v1alpha1.WebhookEvent(0)
}

func (m *EventRequest) GetPayload() isEventRequest_Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (x *EventRequest) GetRepositoryPush() *RepositoryPushEvent {
	if x, ok := x.GetPayload().(*EventRequest_RepositoryPush); ok {
		return x.RepositoryPush
	}
	return nil
}

type isEventRequest_Payload interface {
	isEventRequest_Payload()
}

type EventRequest_RepositoryPush struct {
	RepositoryPush *RepositoryPushEvent `protobuf:"bytes,2,opt,name=repository_push,json=repositoryPush,proto3,oneof"`
}

func (*EventRequest_RepositoryPush) isEventRequest_Payload() {}

// Payload for the event WEBHOOK_EVENT_REPOSITORY_PUSH.
type RepositoryPushEvent struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The timestamp of the commit push.
	EventTime *timestamppb.Timestamp `protobuf:"bytes,1,opt,name=event_time,json=eventTime,proto3" json:"event_time,omitempty"`
	// The repository commit that was pushed.
	RepositoryCommit *v1alpha1.RepositoryCommit `protobuf:"bytes,2,opt,name=repository_commit,json=repositoryCommit,proto3" json:"repository_commit,omitempty"`
	// The repository that was pushed.
	Repository *v1alpha1.Repository `protobuf:"bytes,3,opt,name=repository,proto3" json:"repository,omitempty"`
}

func (x *RepositoryPushEvent) Reset() {
	*x = RepositoryPushEvent{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RepositoryPushEvent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RepositoryPushEvent) ProtoMessage() {}

func (x *RepositoryPushEvent) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RepositoryPushEvent.ProtoReflect.Descriptor instead.
func (*RepositoryPushEvent) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_rawDescGZIP(), []int{1}
}

func (x *RepositoryPushEvent) GetEventTime() *timestamppb.Timestamp {
	if x != nil {
		return x.EventTime
	}
	return nil
}

func (x *RepositoryPushEvent) GetRepositoryCommit() *v1alpha1.RepositoryCommit {
	if x != nil {
		return x.RepositoryCommit
	}
	return nil
}

func (x *RepositoryPushEvent) GetRepository() *v1alpha1.Repository {
	if x != nil {
		return x.Repository
	}
	return nil
}

var File_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto protoreflect.FileDescriptor

var file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_rawDesc = []byte{
	0x0a, 0x38, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2f, 0x76, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x5f, 0x70, 0x61, 0x79, 0x6c,
	0x6f, 0x61, 0x64, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x23, 0x62, 0x75, 0x66, 0x2e,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x77,
	0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x1a,
	0x2c, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x72, 0x65, 0x70,
	0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x33, 0x62,
	0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x72, 0x65, 0x70, 0x6f, 0x73,
	0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x29, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f,
	0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xbf,
	0x01, 0x0a, 0x0c, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x3f, 0x0a, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x29,
	0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x57, 0x65, 0x62,
	0x68, 0x6f, 0x6f, 0x6b, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74,
	0x12, 0x63, 0x0a, 0x0f, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x70,
	0x75, 0x73, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x38, 0x2e, 0x62, 0x75, 0x66, 0x2e,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x77,
	0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e,
	0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x50, 0x75, 0x73, 0x68, 0x45, 0x76,
	0x65, 0x6e, 0x74, 0x48, 0x00, 0x52, 0x0e, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72,
	0x79, 0x50, 0x75, 0x73, 0x68, 0x42, 0x09, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64,
	0x22, 0xf5, 0x01, 0x0a, 0x13, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x50,
	0x75, 0x73, 0x68, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x39, 0x0a, 0x0a, 0x65, 0x76, 0x65, 0x6e,
	0x74, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x54,
	0x69, 0x6d, 0x65, 0x12, 0x5a, 0x0a, 0x11, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72,
	0x79, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2d,
	0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x52, 0x65, 0x70,
	0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x52, 0x10, 0x72,
	0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x12,
	0x47, 0x0a, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x52, 0x0a, 0x72, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x42, 0xd0, 0x02, 0x0a, 0x27, 0x63, 0x6f, 0x6d,
	0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2e, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x76, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0x42, 0x12, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x50, 0x61, 0x79, 0x6c, 0x6f,
	0x61, 0x64, 0x73, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x60, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f,
	0x62, 0x75, 0x66, 0x2f, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x77, 0x65, 0x62, 0x68,
	0x6f, 0x6f, 0x6b, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x3b, 0x77, 0x65, 0x62,
	0x68, 0x6f, 0x6f, 0x6b, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xa2, 0x02, 0x04, 0x42,
	0x41, 0x52, 0x57, 0xaa, 0x02, 0x23, 0x42, 0x75, 0x66, 0x2e, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b,
	0x2e, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xca, 0x02, 0x23, 0x42, 0x75, 0x66, 0x5c,
	0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x57,
	0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xe2,
	0x02, 0x2f, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x52, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x5c, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x5c, 0x56, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74,
	0x61, 0xea, 0x02, 0x27, 0x42, 0x75, 0x66, 0x3a, 0x3a, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x3a, 0x3a,
	0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x3a, 0x3a, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f,
	0x6b, 0x3a, 0x3a, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_rawDescOnce sync.Once
	file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_rawDescData = file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_rawDesc
)

func file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_rawDescGZIP() []byte {
	file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_rawDescOnce.Do(func() {
		file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_rawDescData)
	})
	return file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_rawDescData
}

var file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_goTypes = []interface{}{
	(*EventRequest)(nil),              // 0: buf.alpha.registry.webhook.v1alpha1.EventRequest
	(*RepositoryPushEvent)(nil),       // 1: buf.alpha.registry.webhook.v1alpha1.RepositoryPushEvent
	(v1alpha1.WebhookEvent)(0),        // 2: buf.alpha.registry.v1alpha1.WebhookEvent
	(*timestamppb.Timestamp)(nil),     // 3: google.protobuf.Timestamp
	(*v1alpha1.RepositoryCommit)(nil), // 4: buf.alpha.registry.v1alpha1.RepositoryCommit
	(*v1alpha1.Repository)(nil),       // 5: buf.alpha.registry.v1alpha1.Repository
}
var file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_depIdxs = []int32{
	2, // 0: buf.alpha.registry.webhook.v1alpha1.EventRequest.event:type_name -> buf.alpha.registry.v1alpha1.WebhookEvent
	1, // 1: buf.alpha.registry.webhook.v1alpha1.EventRequest.repository_push:type_name -> buf.alpha.registry.webhook.v1alpha1.RepositoryPushEvent
	3, // 2: buf.alpha.registry.webhook.v1alpha1.RepositoryPushEvent.event_time:type_name -> google.protobuf.Timestamp
	4, // 3: buf.alpha.registry.webhook.v1alpha1.RepositoryPushEvent.repository_commit:type_name -> buf.alpha.registry.v1alpha1.RepositoryCommit
	5, // 4: buf.alpha.registry.webhook.v1alpha1.RepositoryPushEvent.repository:type_name -> buf.alpha.registry.v1alpha1.Repository
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_init() }
func file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_init() {
	if File_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RepositoryPushEvent); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*EventRequest_RepositoryPush)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_depIdxs,
		MessageInfos:      file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto = out.File
	file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_rawDesc = nil
	file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_goTypes = nil
	file_buf_alpha_registry_webhook_v1alpha1_event_payloads_proto_depIdxs = nil
}

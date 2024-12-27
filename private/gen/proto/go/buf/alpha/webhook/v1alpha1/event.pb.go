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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.1
// 	protoc        (unknown)
// source: buf/alpha/webhook/v1alpha1/event.proto

//go:build !protoopaque

package webhookv1alpha1

import (
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// EventRequest is the request payload that will be sent to the customer
// that is subscribed to webhook events in the BSR.
type EventRequest struct {
	state protoimpl.MessageState `protogen:"hybrid.v1"`
	// The webhook event that was triggered. This event is the same one that is
	// registered when creating a webhook in the BSR.
	Event v1alpha1.WebhookEvent `protobuf:"varint,1,opt,name=event,proto3,enum=buf.alpha.registry.v1alpha1.WebhookEvent" json:"event,omitempty"`
	// The event payload of the event was triggered.
	Payload       *EventPayload `protobuf:"bytes,2,opt,name=payload,proto3" json:"payload,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EventRequest) Reset() {
	*x = EventRequest{}
	mi := &file_buf_alpha_webhook_v1alpha1_event_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EventRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventRequest) ProtoMessage() {}

func (x *EventRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_webhook_v1alpha1_event_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *EventRequest) GetEvent() v1alpha1.WebhookEvent {
	if x != nil {
		return x.Event
	}
	return v1alpha1.WebhookEvent(0)
}

func (x *EventRequest) GetPayload() *EventPayload {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *EventRequest) SetEvent(v v1alpha1.WebhookEvent) {
	x.Event = v
}

func (x *EventRequest) SetPayload(v *EventPayload) {
	x.Payload = v
}

func (x *EventRequest) HasPayload() bool {
	if x == nil {
		return false
	}
	return x.Payload != nil
}

func (x *EventRequest) ClearPayload() {
	x.Payload = nil
}

type EventRequest_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// The webhook event that was triggered. This event is the same one that is
	// registered when creating a webhook in the BSR.
	Event v1alpha1.WebhookEvent
	// The event payload of the event was triggered.
	Payload *EventPayload
}

func (b0 EventRequest_builder) Build() *EventRequest {
	m0 := &EventRequest{}
	b, x := &b0, m0
	_, _ = b, x
	x.Event = b.Event
	x.Payload = b.Payload
	return m0
}

// EventPayload contains the actual event payload for all possible
// webhook event types.
type EventPayload struct {
	state protoimpl.MessageState `protogen:"hybrid.v1"`
	// Types that are valid to be assigned to Payload:
	//
	//	*EventPayload_RepositoryPush
	Payload       isEventPayload_Payload `protobuf_oneof:"payload"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EventPayload) Reset() {
	*x = EventPayload{}
	mi := &file_buf_alpha_webhook_v1alpha1_event_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EventPayload) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventPayload) ProtoMessage() {}

func (x *EventPayload) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_webhook_v1alpha1_event_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *EventPayload) GetPayload() isEventPayload_Payload {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *EventPayload) GetRepositoryPush() *RepositoryPushEvent {
	if x != nil {
		if x, ok := x.Payload.(*EventPayload_RepositoryPush); ok {
			return x.RepositoryPush
		}
	}
	return nil
}

func (x *EventPayload) SetRepositoryPush(v *RepositoryPushEvent) {
	if v == nil {
		x.Payload = nil
		return
	}
	x.Payload = &EventPayload_RepositoryPush{v}
}

func (x *EventPayload) HasPayload() bool {
	if x == nil {
		return false
	}
	return x.Payload != nil
}

func (x *EventPayload) HasRepositoryPush() bool {
	if x == nil {
		return false
	}
	_, ok := x.Payload.(*EventPayload_RepositoryPush)
	return ok
}

func (x *EventPayload) ClearPayload() {
	x.Payload = nil
}

func (x *EventPayload) ClearRepositoryPush() {
	if _, ok := x.Payload.(*EventPayload_RepositoryPush); ok {
		x.Payload = nil
	}
}

const EventPayload_Payload_not_set_case case_EventPayload_Payload = 0
const EventPayload_RepositoryPush_case case_EventPayload_Payload = 1

func (x *EventPayload) WhichPayload() case_EventPayload_Payload {
	if x == nil {
		return EventPayload_Payload_not_set_case
	}
	switch x.Payload.(type) {
	case *EventPayload_RepositoryPush:
		return EventPayload_RepositoryPush_case
	default:
		return EventPayload_Payload_not_set_case
	}
}

type EventPayload_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// Fields of oneof Payload:
	RepositoryPush *RepositoryPushEvent
	// -- end of Payload
}

func (b0 EventPayload_builder) Build() *EventPayload {
	m0 := &EventPayload{}
	b, x := &b0, m0
	_, _ = b, x
	if b.RepositoryPush != nil {
		x.Payload = &EventPayload_RepositoryPush{b.RepositoryPush}
	}
	return m0
}

type case_EventPayload_Payload protoreflect.FieldNumber

func (x case_EventPayload_Payload) String() string {
	md := file_buf_alpha_webhook_v1alpha1_event_proto_msgTypes[1].Descriptor()
	if x == 0 {
		return "not set"
	}
	return protoimpl.X.MessageFieldStringOf(md, protoreflect.FieldNumber(x))
}

type isEventPayload_Payload interface {
	isEventPayload_Payload()
}

type EventPayload_RepositoryPush struct {
	RepositoryPush *RepositoryPushEvent `protobuf:"bytes,1,opt,name=repository_push,json=repositoryPush,proto3,oneof"`
}

func (*EventPayload_RepositoryPush) isEventPayload_Payload() {}

// EventResponse is the empty response payload from the customer to Buf.
type EventResponse struct {
	state         protoimpl.MessageState `protogen:"hybrid.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EventResponse) Reset() {
	*x = EventResponse{}
	mi := &file_buf_alpha_webhook_v1alpha1_event_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EventResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventResponse) ProtoMessage() {}

func (x *EventResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_webhook_v1alpha1_event_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

type EventResponse_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

}

func (b0 EventResponse_builder) Build() *EventResponse {
	m0 := &EventResponse{}
	b, x := &b0, m0
	_, _ = b, x
	return m0
}

// Payload for the event WEBHOOK_EVENT_REPOSITORY_PUSH.
type RepositoryPushEvent struct {
	state protoimpl.MessageState `protogen:"hybrid.v1"`
	// The timestamp of the commit push.
	EventTime *timestamppb.Timestamp `protobuf:"bytes,1,opt,name=event_time,json=eventTime,proto3" json:"event_time,omitempty"`
	// The repository commit that was pushed.
	RepositoryCommit *v1alpha1.RepositoryCommit `protobuf:"bytes,2,opt,name=repository_commit,json=repositoryCommit,proto3" json:"repository_commit,omitempty"`
	// The repository that was pushed.
	Repository    *v1alpha1.Repository `protobuf:"bytes,3,opt,name=repository,proto3" json:"repository,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RepositoryPushEvent) Reset() {
	*x = RepositoryPushEvent{}
	mi := &file_buf_alpha_webhook_v1alpha1_event_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RepositoryPushEvent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RepositoryPushEvent) ProtoMessage() {}

func (x *RepositoryPushEvent) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_webhook_v1alpha1_event_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
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

func (x *RepositoryPushEvent) SetEventTime(v *timestamppb.Timestamp) {
	x.EventTime = v
}

func (x *RepositoryPushEvent) SetRepositoryCommit(v *v1alpha1.RepositoryCommit) {
	x.RepositoryCommit = v
}

func (x *RepositoryPushEvent) SetRepository(v *v1alpha1.Repository) {
	x.Repository = v
}

func (x *RepositoryPushEvent) HasEventTime() bool {
	if x == nil {
		return false
	}
	return x.EventTime != nil
}

func (x *RepositoryPushEvent) HasRepositoryCommit() bool {
	if x == nil {
		return false
	}
	return x.RepositoryCommit != nil
}

func (x *RepositoryPushEvent) HasRepository() bool {
	if x == nil {
		return false
	}
	return x.Repository != nil
}

func (x *RepositoryPushEvent) ClearEventTime() {
	x.EventTime = nil
}

func (x *RepositoryPushEvent) ClearRepositoryCommit() {
	x.RepositoryCommit = nil
}

func (x *RepositoryPushEvent) ClearRepository() {
	x.Repository = nil
}

type RepositoryPushEvent_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// The timestamp of the commit push.
	EventTime *timestamppb.Timestamp
	// The repository commit that was pushed.
	RepositoryCommit *v1alpha1.RepositoryCommit
	// The repository that was pushed.
	Repository *v1alpha1.Repository
}

func (b0 RepositoryPushEvent_builder) Build() *RepositoryPushEvent {
	m0 := &RepositoryPushEvent{}
	b, x := &b0, m0
	_, _ = b, x
	x.EventTime = b.EventTime
	x.RepositoryCommit = b.RepositoryCommit
	x.Repository = b.Repository
	return m0
}

var File_buf_alpha_webhook_v1alpha1_event_proto protoreflect.FileDescriptor

var file_buf_alpha_webhook_v1alpha1_event_proto_rawDesc = []byte{
	0x0a, 0x26, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x77, 0x65, 0x62, 0x68,
	0x6f, 0x6f, 0x6b, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x65, 0x76, 0x65,
	0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1a, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x76, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0x1a, 0x2c, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2f, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x33, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f,
	0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x69,
	0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x29, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0x2f, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0x93, 0x01, 0x0a, 0x0c, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x12, 0x3f, 0x0a, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0e, 0x32, 0x29, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2e, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x05,
	0x65, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x42, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64,
	0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x22, 0x75, 0x0a, 0x0c, 0x45, 0x76, 0x65,
	0x6e, 0x74, 0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x5a, 0x0a, 0x0f, 0x72, 0x65, 0x70,
	0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x70, 0x75, 0x73, 0x68, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x2f, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x77,
	0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e,
	0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x50, 0x75, 0x73, 0x68, 0x45, 0x76,
	0x65, 0x6e, 0x74, 0x48, 0x00, 0x52, 0x0e, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72,
	0x79, 0x50, 0x75, 0x73, 0x68, 0x42, 0x09, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64,
	0x22, 0x0f, 0x0a, 0x0d, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0xf5, 0x01, 0x0a, 0x13, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79,
	0x50, 0x75, 0x73, 0x68, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x39, 0x0a, 0x0a, 0x65, 0x76, 0x65,
	0x6e, 0x74, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x65, 0x76, 0x65, 0x6e, 0x74,
	0x54, 0x69, 0x6d, 0x65, 0x12, 0x5a, 0x0a, 0x11, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f,
	0x72, 0x79, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x2d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x52, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x52, 0x10,
	0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74,
	0x12, 0x47, 0x0a, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x52, 0x0a, 0x72,
	0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x32, 0x6c, 0x0a, 0x0c, 0x45, 0x76, 0x65,
	0x6e, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x5c, 0x0a, 0x05, 0x45, 0x76, 0x65,
	0x6e, 0x74, 0x12, 0x28, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x77,
	0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e,
	0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x29, 0x2e, 0x62,
	0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b,
	0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x90, 0x02, 0x0a, 0x1e, 0x63, 0x6f, 0x6d, 0x2e,
	0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f,
	0x6b, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x42, 0x0a, 0x45, 0x76, 0x65, 0x6e,
	0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x57, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x62, 0x75,
	0x66, 0x2f, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2f, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x3b, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0xa2, 0x02, 0x03, 0x42, 0x41, 0x57, 0xaa, 0x02, 0x1a, 0x42, 0x75, 0x66, 0x2e, 0x41, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x56, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0xca, 0x02, 0x1a, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61,
	0x5c, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0xe2, 0x02, 0x26, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x57, 0x65,
	0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x5c, 0x47,
	0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x1d, 0x42, 0x75, 0x66,
	0x3a, 0x3a, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x3a, 0x3a, 0x57, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b,
	0x3a, 0x3a, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var file_buf_alpha_webhook_v1alpha1_event_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_buf_alpha_webhook_v1alpha1_event_proto_goTypes = []any{
	(*EventRequest)(nil),              // 0: buf.alpha.webhook.v1alpha1.EventRequest
	(*EventPayload)(nil),              // 1: buf.alpha.webhook.v1alpha1.EventPayload
	(*EventResponse)(nil),             // 2: buf.alpha.webhook.v1alpha1.EventResponse
	(*RepositoryPushEvent)(nil),       // 3: buf.alpha.webhook.v1alpha1.RepositoryPushEvent
	(v1alpha1.WebhookEvent)(0),        // 4: buf.alpha.registry.v1alpha1.WebhookEvent
	(*timestamppb.Timestamp)(nil),     // 5: google.protobuf.Timestamp
	(*v1alpha1.RepositoryCommit)(nil), // 6: buf.alpha.registry.v1alpha1.RepositoryCommit
	(*v1alpha1.Repository)(nil),       // 7: buf.alpha.registry.v1alpha1.Repository
}
var file_buf_alpha_webhook_v1alpha1_event_proto_depIdxs = []int32{
	4, // 0: buf.alpha.webhook.v1alpha1.EventRequest.event:type_name -> buf.alpha.registry.v1alpha1.WebhookEvent
	1, // 1: buf.alpha.webhook.v1alpha1.EventRequest.payload:type_name -> buf.alpha.webhook.v1alpha1.EventPayload
	3, // 2: buf.alpha.webhook.v1alpha1.EventPayload.repository_push:type_name -> buf.alpha.webhook.v1alpha1.RepositoryPushEvent
	5, // 3: buf.alpha.webhook.v1alpha1.RepositoryPushEvent.event_time:type_name -> google.protobuf.Timestamp
	6, // 4: buf.alpha.webhook.v1alpha1.RepositoryPushEvent.repository_commit:type_name -> buf.alpha.registry.v1alpha1.RepositoryCommit
	7, // 5: buf.alpha.webhook.v1alpha1.RepositoryPushEvent.repository:type_name -> buf.alpha.registry.v1alpha1.Repository
	0, // 6: buf.alpha.webhook.v1alpha1.EventService.Event:input_type -> buf.alpha.webhook.v1alpha1.EventRequest
	2, // 7: buf.alpha.webhook.v1alpha1.EventService.Event:output_type -> buf.alpha.webhook.v1alpha1.EventResponse
	7, // [7:8] is the sub-list for method output_type
	6, // [6:7] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_buf_alpha_webhook_v1alpha1_event_proto_init() }
func file_buf_alpha_webhook_v1alpha1_event_proto_init() {
	if File_buf_alpha_webhook_v1alpha1_event_proto != nil {
		return
	}
	file_buf_alpha_webhook_v1alpha1_event_proto_msgTypes[1].OneofWrappers = []any{
		(*EventPayload_RepositoryPush)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_buf_alpha_webhook_v1alpha1_event_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_buf_alpha_webhook_v1alpha1_event_proto_goTypes,
		DependencyIndexes: file_buf_alpha_webhook_v1alpha1_event_proto_depIdxs,
		MessageInfos:      file_buf_alpha_webhook_v1alpha1_event_proto_msgTypes,
	}.Build()
	File_buf_alpha_webhook_v1alpha1_event_proto = out.File
	file_buf_alpha_webhook_v1alpha1_event_proto_rawDesc = nil
	file_buf_alpha_webhook_v1alpha1_event_proto_goTypes = nil
	file_buf_alpha_webhook_v1alpha1_event_proto_depIdxs = nil
}

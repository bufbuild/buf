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

package bufprotosource

import "math"

const (
	// MessageRangeInclusiveMax is the maximum allowed tag for a message field.
	MessageRangeInclusiveMax = 536870911 // 2^29 - 1
	// MessageSetRangeInclusiveMax is the maximum allowed tag for a message
	// field for a message that uses the message-set wire format.
	MessageSetRangeInclusiveMax = math.MaxInt32 - 1
)

type messageRange struct {
	locationDescriptor

	message Message
	start   int
	end     int
}

func newMessageRange(
	locationDescriptor locationDescriptor,
	message Message,
	start int,
	end int,
) *messageRange {
	return &messageRange{
		locationDescriptor: locationDescriptor,
		message:            message,
		start:              start,
		// end is exclusive for messages
		end: end - 1,
	}
}

func (r *messageRange) Message() Message {
	return r.message
}

func (r *messageRange) Start() int {
	return r.start
}

func (r *messageRange) End() int {
	return r.end
}

func (r *messageRange) Max() bool {
	if r.message.MessageSetWireFormat() {
		return r.end == MessageSetRangeInclusiveMax
	}
	return r.end == MessageRangeInclusiveMax
}

type extensionRange struct {
	*messageRange
	optionExtensionDescriptor
}

func newExtensionRange(
	locationDescriptor locationDescriptor,
	message Message,
	start int,
	end int,
	opts optionExtensionDescriptor,
) *extensionRange {
	return &extensionRange{
		messageRange: newMessageRange(
			locationDescriptor, message, start, end,
		),
		optionExtensionDescriptor: opts,
	}
}

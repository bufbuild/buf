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

import (
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type optionExtensionDescriptor struct {
	message       proto.Message
	optionsPath   []int32
	locationStore *locationStore
	featuresTag   protoreflect.FieldNumber
}

func newOptionExtensionDescriptor(message proto.Message, optionsPath []int32, locationStore *locationStore, featuresTag protoreflect.FieldNumber) optionExtensionDescriptor {
	return optionExtensionDescriptor{
		message:       message,
		optionsPath:   optionsPath,
		locationStore: locationStore,
		featuresTag:   featuresTag,
	}
}

func (o *optionExtensionDescriptor) OptionExtension(extensionType protoreflect.ExtensionType) (interface{}, bool) {
	if extensionType.TypeDescriptor().ContainingMessage().FullName() != o.message.ProtoReflect().Descriptor().FullName() {
		return nil, false
	}
	if !proto.HasExtension(o.message, extensionType) {
		return nil, false
	}
	return proto.GetExtension(o.message, extensionType), true
}

func (o *optionExtensionDescriptor) OptionExtensionLocation(extensionType protoreflect.ExtensionType, extraPath ...int32) Location {
	return o.OptionLocation(extensionType.TypeDescriptor(), extraPath...)
}

func (o *optionExtensionDescriptor) OptionLocation(field protoreflect.FieldDescriptor, extraPath ...int32) Location {
	if field.ContainingMessage().FullName() != o.message.ProtoReflect().Descriptor().FullName() {
		return nil
	}
	if o.locationStore == nil {
		return nil
	}
	path := make([]int32, len(o.optionsPath), len(o.optionsPath)+1+len(extraPath))
	copy(path, o.optionsPath)
	path = append(path, int32(field.Number()))
	extensionPathLen := len(path) // length of path to extension (without extraPath)
	path = append(path, extraPath...)
	loc := o.locationStore.getLocation(path)
	if loc != nil {
		// Found an exact match!
		return loc
	}
	// "Fuzzy" search: find a location whose path is at least extensionPathLen long,
	// preferring the longest matching ancestor path (i.e. as many extraPath elements
	// as can be found). If we find a *sub*path (a descendant path, that points INTO
	// the path we are trying to find), use the first such one encountered.
	var bestMatch *descriptorpb.SourceCodeInfo_Location
	var bestMatchPathLen int
	for _, loc := range o.locationStore.sourceCodeInfoLocations {
		if len(loc.Path) >= extensionPathLen && isDescendantPath(path, loc.Path) && len(loc.Path) > bestMatchPathLen {
			bestMatch = loc
			bestMatchPathLen = len(loc.Path)
		} else if isDescendantPath(loc.Path, path) {
			return newLocation(loc)
		}
	}
	if bestMatch != nil {
		return newLocation(bestMatch)
	}
	return nil
}

func (o *optionExtensionDescriptor) PresentExtensionNumbers() []int32 {
	fieldNumbersSet := map[int32]struct{}{}
	var fieldNumbers []int32
	addFieldNumber := func(fieldNo int32) {
		if _, ok := fieldNumbersSet[fieldNo]; !ok {
			fieldNumbersSet[fieldNo] = struct{}{}
			fieldNumbers = append(fieldNumbers, fieldNo)
		}
	}
	msg := o.message.ProtoReflect()
	extensionRanges := msg.Descriptor().ExtensionRanges()
	for b := msg.GetUnknown(); len(b) > 0; {
		fieldNo, _, n := protowire.ConsumeField(b)
		if extensionRanges.Has(fieldNo) {
			addFieldNumber(int32(fieldNo))
		}
		b = b[n:]
	}
	// Extensions for google.protobuf.*Options are a bit of a special case
	// as the extensions in a FileDescriptorSet message may differ with
	// the extensions defined in the proto with which buf is compiled.
	//
	// Also loop through known extensions here to get extension numbers.
	msg.Range(func(fieldDescriptor protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		if fieldDescriptor.IsExtension() {
			addFieldNumber(int32(fieldDescriptor.Number()))
		}
		return true
	})

	return fieldNumbers
}

func isDescendantPath(descendant, ancestor []int32) bool {
	if len(descendant) < len(ancestor) {
		return false
	}
	for i := range ancestor {
		if descendant[i] != ancestor[i] {
			return false
		}
	}
	return true
}

func (o *optionExtensionDescriptor) ForEachPresentOption(fn func(protoreflect.FieldDescriptor, protoreflect.Value) bool) {
	// Note: This does not bother to handle unrecognized fields.
	// Should not be a problem since descriptors models in the buf CLI codebase should have them
	// all correctly parsed and known.
	o.message.ProtoReflect().Range(fn)
}

func (o *optionExtensionDescriptor) Features() FeaturesDescriptor {
	return o
}

func (o *optionExtensionDescriptor) FieldPresenceLocation() Location {
	features := o.message.ProtoReflect().Descriptor().Fields().ByNumber(o.featuresTag)
	return o.OptionLocation(features, 1)
}

func (o *optionExtensionDescriptor) EnumTypeLocation() Location {
	features := o.message.ProtoReflect().Descriptor().Fields().ByNumber(o.featuresTag)
	return o.OptionLocation(features, 2)
}

func (o *optionExtensionDescriptor) RepeatedFieldEncodingLocation() Location {
	features := o.message.ProtoReflect().Descriptor().Fields().ByNumber(o.featuresTag)
	return o.OptionLocation(features, 3)
}

func (o *optionExtensionDescriptor) UTF8ValidationLocation() Location {
	features := o.message.ProtoReflect().Descriptor().Fields().ByNumber(o.featuresTag)
	return o.OptionLocation(features, 4)
}

func (o *optionExtensionDescriptor) MessageEncodingLocation() Location {
	features := o.message.ProtoReflect().Descriptor().Fields().ByNumber(o.featuresTag)
	return o.OptionLocation(features, 5)
}

func (o *optionExtensionDescriptor) JSONFormatLocation() Location {
	features := o.message.ProtoReflect().Descriptor().Fields().ByNumber(o.featuresTag)
	return o.OptionLocation(features, 6)
}

package bufimagemodifyv2

import "google.golang.org/protobuf/types/descriptorpb"

type prefixOverride string

func newPrefixOverride(prefix string) prefixOverride {
	return prefixOverride(prefix)
}

func (p prefixOverride) get() string {
	return string(p)
}

func (o prefixOverride) override() {}

type valueOverride[T string | bool | descriptorpb.FileOptions_OptimizeMode] struct {
	value T
}

func newValueOverride[T string | bool | descriptorpb.FileOptions_OptimizeMode](val T) valueOverride[T] {
	return valueOverride[T]{
		value: val,
	}
}

func (v valueOverride[T]) get() T {
	return v.value
}

func (v valueOverride[T]) override() {}

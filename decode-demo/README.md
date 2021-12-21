# Runtime Decoding Demo

This directory contains a simple demo for runtime decoding.
Descriptors encoded with DescriptorInfo can be decoded at runtime
with the BSR.

Here, we simply use `protoc --encode` (we could also use `buf encode`)
to create a Protobuf payload for a `pet.v1.Pet` message and decode it
with `buf decode` (using the `DescriptorInfo` that describes the
message's source).

The tests written here will become real integration tests - this is
only meant as a demonstration for now.

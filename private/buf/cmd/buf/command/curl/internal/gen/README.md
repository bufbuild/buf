## External Proto Sources

This directory contains sources and generated code for Google's
gRPC server reflection protocol.

The normal generated Go package for these sources has dependencies
on the gRPC runtime, which we want to exclude for Buf since we
use the Connect runtime instead. Also, there is no generated Go
package for v1 of the protocol, only for v1alpha (at least not
yet, as of December 2022).

These are not in the primary proto and private/gen folders in this
repo to avoid polluting our other proto sources with these vendored
copies of third-party sources.
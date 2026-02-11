package hyperpb

import (
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// buildFileDescriptor builds a protoreflect.FileDescriptor from a
// FileDescriptorProto. Used in tests to create descriptors without
// depending on generated code.
func buildFileDescriptor(fdp *descriptorpb.FileDescriptorProto) (protoreflect.FileDescriptor, error) {
	return protodesc.NewFile(fdp, nil)
}

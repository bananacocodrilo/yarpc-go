package hyperpb

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// buildTestFileDescriptor creates a FileDescriptorProto for a simple test
// service with a single unary method. This avoids depending on any internal
// generated proto package.
//
//	syntax = "proto3";
//	package test.v1;
//	message EchoRequest  { string value = 1; }
//	message EchoResponse { string value = 1; }
//	service Echo { rpc Ping(EchoRequest) returns (EchoResponse); }
func buildTestFileDescriptor() *descriptorpb.FileDescriptorProto {
	syntax := "proto3"
	pkg := "test.v1"
	fileName := "test.proto"

	stringType := descriptorpb.FieldDescriptorProto_TYPE_STRING
	labelOptional := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	fieldNum := int32(1)
	fieldName := "value"

	reqName := "EchoRequest"
	respName := "EchoResponse"
	svcName := "Echo"
	methodName := "Ping"
	inputType := ".test.v1.EchoRequest"
	outputType := ".test.v1.EchoResponse"

	return &descriptorpb.FileDescriptorProto{
		Name:    &fileName,
		Syntax:  &syntax,
		Package: &pkg,
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: &reqName,
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   &fieldName,
						Number: &fieldNum,
						Type:   &stringType,
						Label:  &labelOptional,
					},
				},
			},
			{
				Name: &respName,
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   &fieldName,
						Number: &fieldNum,
						Type:   &stringType,
						Label:  &labelOptional,
					},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: &svcName,
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       &methodName,
						InputType:  &inputType,
						OutputType: &outputType,
					},
				},
			},
		},
	}
}

// testDescriptors returns resolved file, message, and service descriptors
// from the test FileDescriptorProto.
func testDescriptors(t *testing.T) (
	protoreflect.FileDescriptor,
	protoreflect.MessageDescriptor,
	protoreflect.MessageDescriptor,
	protoreflect.ServiceDescriptor,
) {
	t.Helper()
	fdProto := buildTestFileDescriptor()

	// Build a protoreflect.FileDescriptor from the proto.
	fd, err := buildFileDescriptor(fdProto)
	require.NoError(t, err)

	reqMD := fd.Messages().ByName("EchoRequest")
	require.NotNil(t, reqMD)
	respMD := fd.Messages().ByName("EchoResponse")
	require.NotNil(t, respMD)
	sd := fd.Services().ByName("Echo")
	require.NotNil(t, sd)

	return fd, reqMD, respMD, sd
}

func TestTypeRegistry_RegisterAndNewMessage(t *testing.T) {
	_, reqMD, _, _ := testDescriptors(t)

	registry := NewTypeRegistry()
	registry.RegisterMessageDescriptor(reqMD)

	msg, err := registry.NewMessage(reqMD.FullName())
	require.NoError(t, err)
	require.NotNil(t, msg)

	// It should implement proto.Message.
	assert.Implements(t, (*proto.Message)(nil), msg)
}

func TestTypeRegistry_RegisterServiceDescriptor(t *testing.T) {
	_, _, _, sd := testDescriptors(t)

	registry := NewTypeRegistry()
	registry.RegisterServiceDescriptor(sd)

	// Both request and response types should be registered.
	msg, err := registry.NewMessage("test.v1.EchoRequest")
	require.NoError(t, err)
	require.NotNil(t, msg)

	msg2, err := registry.NewMessage("test.v1.EchoResponse")
	require.NoError(t, err)
	require.NotNil(t, msg2)
}

func TestTypeRegistry_NewMessage_NotRegistered(t *testing.T) {
	registry := NewTypeRegistry()
	_, err := registry.NewMessage("does.not.Exist")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestMarshalRoundTrip_Proto(t *testing.T) {
	_, reqMD, _, _ := testDescriptors(t)

	// Build a dynamicpb message as our "original" typed message.
	original := dynamicpb.NewMessage(reqMD)
	valueField := reqMD.Fields().ByName("value")
	original.Set(valueField, protoreflect.ValueOfString("hello"))

	data, cleanup, err := marshal(Encoding, original)
	require.NoError(t, err)
	defer cleanup()

	// Unmarshal into a hyperpb message.
	registry := NewTypeRegistry()
	registry.RegisterMessageDescriptor(reqMD)

	msg, err := registry.NewMessage(reqMD.FullName())
	require.NoError(t, err)

	err = proto.Unmarshal(data, msg)
	require.NoError(t, err)

	// Read the value field via reflection.
	assert.Equal(t, "hello", msg.ProtoReflect().Get(valueField).String())
}

func TestMarshalRoundTrip_JSON(t *testing.T) {
	_, reqMD, _, _ := testDescriptors(t)

	original := dynamicpb.NewMessage(reqMD)
	valueField := reqMD.Fields().ByName("value")
	original.Set(valueField, protoreflect.ValueOfString("world"))

	data, cleanup, err := marshal(JSONEncoding, original)
	require.NoError(t, err)
	defer cleanup()

	// JSON should produce readable output.
	assert.Contains(t, string(data), "world")

	// Unmarshal back into a dynamicpb message (JSON round-trip).
	restored := dynamicpb.NewMessage(reqMD)
	err = unmarshalBytes(JSONEncoding, data, restored)
	require.NoError(t, err)
	assert.Equal(t, "world", restored.Get(valueField).String())
}

func TestUnaryHandler_ProtoEncoding(t *testing.T) {
	_, reqMD, _, sd := testDescriptors(t)

	registry := NewTypeRegistry()
	registry.RegisterServiceDescriptor(sd)

	// Echo handler: reads request via reflection, builds response via dynamicpb.
	echoHandler := func(ctx context.Context, req proto.Message) (proto.Message, error) {
		fields := req.ProtoReflect().Descriptor().Fields()
		valueField := fields.ByName("value")

		respDesc := req.ProtoReflect().Descriptor()
		resp := dynamicpb.NewMessage(respDesc)
		resp.Set(valueField, req.ProtoReflect().Get(valueField))
		return resp, nil
	}

	procedures, err := BuildProceduresFromDescriptor(sd, registry, map[string]UnaryHandler{
		"Ping": echoHandler,
	})
	require.NoError(t, err)
	require.Len(t, procedures, 2) // proto + json

	// Find the proto-encoded procedure.
	var protoProcedure transport.Procedure
	for _, p := range procedures {
		if p.Encoding == Encoding {
			protoProcedure = p
			break
		}
	}
	assert.Equal(t, "test.v1.Echo::Ping", protoProcedure.Name)

	// Serialize a request using dynamicpb.
	valueField := reqMD.Fields().ByName("value")
	reqMsg := dynamicpb.NewMessage(reqMD)
	reqMsg.Set(valueField, protoreflect.ValueOfString("test-echo"))
	reqData, err := proto.Marshal(reqMsg)
	require.NoError(t, err)

	// Build a transport request.
	transportReq := &transport.Request{
		Caller:    "test-caller",
		Service:   "test-service",
		Procedure: protoProcedure.Name,
		Encoding:  Encoding,
		Body:      bytes.NewReader(reqData),
	}

	resw := &transporttest.FakeResponseWriter{}

	// Handle the request.
	handler := protoProcedure.HandlerSpec.Unary()
	err = handler.Handle(context.Background(), transportReq, resw)
	require.NoError(t, err)

	// Unmarshal the response.
	respMsg := dynamicpb.NewMessage(reqMD)
	err = proto.Unmarshal(resw.Body.Bytes(), respMsg)
	require.NoError(t, err)
	assert.Equal(t, "test-echo", respMsg.Get(valueField).String())
}

func TestBuildProcedures_NamingConvention(t *testing.T) {
	_, reqMD, _, _ := testDescriptors(t)

	registry := NewTypeRegistry()
	registry.RegisterMessageDescriptor(reqMD)

	noop := func(ctx context.Context, req proto.Message) (proto.Message, error) {
		return nil, nil
	}

	procedures := BuildProcedures(BuildProceduresParams{
		ServiceName: "my.package.v1.MyService",
		Registry:    registry,
		UnaryHandlers: map[string]UnaryHandler{
			"Echo": noop,
		},
	})

	require.Len(t, procedures, 2)

	encodings := map[transport.Encoding]bool{}
	for _, p := range procedures {
		assert.Equal(t, "my.package.v1.MyService::Echo", p.Name)
		encodings[p.Encoding] = true
	}
	assert.True(t, encodings[Encoding])
	assert.True(t, encodings[JSONEncoding])
}

func TestBuildProceduresFromDescriptor_InvalidMethod(t *testing.T) {
	_, _, _, sd := testDescriptors(t)

	registry := NewTypeRegistry()

	noop := func(ctx context.Context, req proto.Message) (proto.Message, error) {
		return nil, nil
	}

	_, err := BuildProceduresFromDescriptor(sd, registry, map[string]UnaryHandler{
		"NonExistent": noop,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

package yarpc_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"testing"

	lib "buf.build/go/hyperpb"
	gogoproto "github.com/gogo/protobuf/proto"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/hyperpb"
	gogoencoding "go.uber.org/yarpc/encoding/protobuf"
	v2encoding "go.uber.org/yarpc/encoding/protobuf/v2"
	"go.uber.org/yarpc/encoding/rawproto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ============================================================
// Shared benchmark harness — identical for all approaches
// ============================================================

var compareBenchSizes = []struct {
	name string
	size int
}{
	{"Small_350B", 300},
	{"Medium_10KB", 10 * 1024},
	{"Large_1MB", 1024 * 1024},
}

// runHandlerBench is the single loop shared by every benchmark.
// Only the handler and pre-serialized bytes differ.
func runHandlerBench(b *testing.B, handler transport.UnaryHandler, procedure string, reqData []byte) {
	b.Helper()
	b.ReportAllocs()

	ctx := context.Background()
	reader := bytes.NewReader(nil)

	for i := 0; i < b.N; i++ {
		reader.Reset(reqData)
		req := &transport.Request{
			Caller:    "bench-caller",
			Service:   "bench-service",
			Procedure: procedure,
			Encoding:  "proto",
			Body:      reader,
		}
		rw := &transporttest.FakeResponseWriter{}
		if err := handler.Handle(ctx, req, rw); err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================================
// Baseline — raw transport.UnaryHandler, no encoding layer
// ============================================================

type baselineHandler struct{}

func (baselineHandler) Handle(ctx context.Context, req *transport.Request, rw transport.ResponseWriter) error {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	_, err = rw.Write(body)
	return err
}

// ============================================================
// Gogo test message (hand-rolled, matches codegen output)
// ============================================================

type benchGogoMsg struct {
	Value string `protobuf:"bytes,1,opt,name=value,proto3"`
}

func (m *benchGogoMsg) Reset()         { *m = benchGogoMsg{} }
func (m *benchGogoMsg) String() string { return m.Value }
func (m *benchGogoMsg) ProtoMessage()  {}

func (m *benchGogoMsg) Marshal() ([]byte, error) {
	size := m.gogoSize()
	buf := make([]byte, size)
	n, err := m.MarshalToSizedBuffer(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (m *benchGogoMsg) MarshalTo(buf []byte) (int, error) {
	return m.MarshalToSizedBuffer(buf[:m.gogoSize()])
}

func (m *benchGogoMsg) MarshalToSizedBuffer(buf []byte) (int, error) {
	i := len(buf)
	if len(m.Value) > 0 {
		i -= len(m.Value)
		copy(buf[i:], m.Value)
		l := uint64(len(m.Value))
		for l >= 0x80 {
			i--
			buf[i] = byte(l) | 0x80
			l >>= 7
		}
		i--
		buf[i] = byte(l)
		i--
		buf[i] = 0x0a
	}
	return len(buf) - i, nil
}

func (m *benchGogoMsg) gogoSize() int {
	l := len(m.Value)
	if l == 0 {
		return 0
	}
	n := 1
	for x := uint64(l); x >= 0x80; x >>= 7 {
		n++
	}
	return n + 1 + l
}

func (m *benchGogoMsg) Unmarshal(data []byte) error {
	m.Value = ""
	i := 0
	for i < len(data) {
		tag, n := binary.Uvarint(data[i:])
		if n <= 0 {
			return fmt.Errorf("bad varint")
		}
		i += n
		wireType := tag & 0x7
		fieldNumber := tag >> 3
		switch wireType {
		case 2:
			length, n := binary.Uvarint(data[i:])
			if n <= 0 {
				return fmt.Errorf("bad varint")
			}
			i += n
			if fieldNumber == 1 {
				m.Value = string(data[i : i+int(length)])
			}
			i += int(length)
		case 0:
			for i < len(data) && data[i] >= 0x80 {
				i++
			}
			i++
		default:
			return fmt.Errorf("unsupported wire type %d", wireType)
		}
	}
	return nil
}

// ============================================================
// Descriptor helpers for hyperpb
// ============================================================

func compareBenchFileDescriptor() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("EchoRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("value"),
						Number: proto.Int32(1),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					},
				},
			},
			{
				Name: proto.String("EchoResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("value"),
						Number: proto.Int32(1),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("Echo"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("Ping"),
						InputType:  proto.String(".test.v1.EchoRequest"),
						OutputType: proto.String(".test.v1.EchoResponse"),
					},
				},
			},
		},
	}
}

// ============================================================
// Handler constructors
// ============================================================

func makeGogoHandler() transport.UnaryHandler {
	return gogoencoding.NewUnaryHandler(gogoencoding.UnaryHandlerParams{
		Handle: func(ctx context.Context, req gogoproto.Message) (gogoproto.Message, error) {
			return req, nil
		},
		NewRequest: func() gogoproto.Message {
			return &benchGogoMsg{}
		},
	})
}

func makeHyperpbHandler(b *testing.B) transport.UnaryHandler {
	b.Helper()
	fd, err := protodesc.NewFile(compareBenchFileDescriptor(), nil)
	if err != nil {
		b.Fatal(err)
	}
	sd := fd.Services().ByName("Echo")

	registry := hyperpb.NewTypeRegistry()
	registry.RegisterServiceDescriptor(sd)

	procs, err := hyperpb.BuildProceduresFromDescriptor(sd, registry, map[string]hyperpb.UnaryHandler{
		"Ping": func(ctx context.Context, req proto.Message) (proto.Message, error) {
			return req, nil
		},
	})
	if err != nil {
		b.Fatal(err)
	}
	for _, p := range procs {
		if p.Encoding == hyperpb.Encoding {
			return p.HandlerSpec.Unary()
		}
	}
	b.Fatal("proto procedure not found")
	return nil
}

func makeGolangProtoHandler(b *testing.B) transport.UnaryHandler {
	b.Helper()
	fd, err := protodesc.NewFile(compareBenchFileDescriptor(), nil)
	if err != nil {
		b.Fatal(err)
	}
	reqMD := fd.Messages().ByName("EchoRequest")
	return v2encoding.NewUnaryHandler(v2encoding.UnaryHandlerParams{
		Handle: func(ctx context.Context, req proto.Message) (proto.Message, error) {
			return req, nil
		},
		NewRequest: func() proto.Message {
			return dynamicpb.NewMessage(reqMD)
		},
	})
}

func makeRawProtoHandler() transport.UnaryHandler {
	procs := rawproto.Procedure("test.v1.Echo::Ping", func(ctx context.Context, body []byte) ([]byte, error) {
		return body, nil
	})
	return procs[0].HandlerSpec.Unary()
}

// makeProtoPayload builds valid proto wire bytes for { string value = 1; }.
func makeProtoPayload(b *testing.B, size int) []byte {
	b.Helper()
	fd, err := protodesc.NewFile(compareBenchFileDescriptor(), nil)
	if err != nil {
		b.Fatal(err)
	}
	reqMD := fd.Messages().ByName("EchoRequest")

	// Pre-compile hyperpb type so it's cached.
	lib.CompileMessageDescriptor(reqMD)

	msg := dynamicpb.NewMessage(reqMD)
	msg.Set(
		reqMD.Fields().ByName("value"),
		protoreflect.ValueOfString(strings.Repeat("x", size)),
	)
	data, err := proto.Marshal(msg)
	if err != nil {
		b.Fatal(err)
	}
	return data
}

// ============================================================
// Benchmarks — identical structure, different handlers
// ============================================================

func BenchmarkHandler_Baseline(b *testing.B) {
	h := baselineHandler{}
	for _, sz := range compareBenchSizes {
		reqData := makeProtoPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runHandlerBench(b, h, "test.v1.Echo::Ping", reqData)
		})
	}
}

func BenchmarkHandler_Gogo(b *testing.B) {
	h := makeGogoHandler()
	for _, sz := range compareBenchSizes {
		reqData := makeProtoPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runHandlerBench(b, h, "Test::Echo", reqData)
		})
	}
}

func BenchmarkHandler_Hyperpb(b *testing.B) {
	h := makeHyperpbHandler(b)
	for _, sz := range compareBenchSizes {
		reqData := makeProtoPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runHandlerBench(b, h, "test.v1.Echo::Ping", reqData)
		})
	}
}

func BenchmarkHandler_GolangProto(b *testing.B) {
	h := makeGolangProtoHandler(b)
	for _, sz := range compareBenchSizes {
		reqData := makeProtoPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runHandlerBench(b, h, "test.v1.Echo::Ping", reqData)
		})
	}
}

func BenchmarkHandler_RawProto(b *testing.B) {
	h := makeRawProtoHandler()
	for _, sz := range compareBenchSizes {
		reqData := makeProtoPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runHandlerBench(b, h, "test.v1.Echo::Ping", reqData)
		})
	}
}

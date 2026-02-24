package yarpc_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
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
// Deep nesting schema:
//
//   message DeepPing {
//     string id       = 1;  // small routing key (20 bytes)
//     Level1 payload  = 2;  // deeply nested bulk data
//   }
//   message Level1 { Level2 child = 1; string data = 2; }
//   message Level2 { Level3 child = 1; string data = 2; }
//   message Level3 { string data = 1; }
//
// The bulk of the payload is spread across the nested levels'
// "data" fields. The handler only reads "id" in the partial
// scenario, leaving the deeply nested payload untouched.
// ============================================================

func deepFileDescriptor() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("deep.proto"),
		Package: proto.String("deep.v1"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Level3"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("data"),
						Number: proto.Int32(1),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					},
				},
			},
			{
				Name: proto.String("Level2"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("child"),
						Number:   proto.Int32(1),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						TypeName: proto.String(".deep.v1.Level3"),
					},
					{
						Name:   proto.String("data"),
						Number: proto.Int32(2),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					},
				},
			},
			{
				Name: proto.String("Level1"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("child"),
						Number:   proto.Int32(1),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						TypeName: proto.String(".deep.v1.Level2"),
					},
					{
						Name:   proto.String("data"),
						Number: proto.Int32(2),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					},
				},
			},
			{
				Name: proto.String("DeepPing"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("id"),
						Number: proto.Int32(1),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					},
					{
						Name:     proto.String("payload"),
						Number:   proto.Int32(2),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						TypeName: proto.String(".deep.v1.Level1"),
					},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("DeepEcho"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("Ping"),
						InputType:  proto.String(".deep.v1.DeepPing"),
						OutputType: proto.String(".deep.v1.DeepPing"),
					},
				},
			},
		},
	}
}

// makeDeepPayload builds valid proto wire bytes for a DeepPing message.
// The bulk of the data is spread across all nesting levels' "data" fields.
func makeDeepPayload(b *testing.B, totalDataSize int) []byte {
	b.Helper()
	fd, err := protodesc.NewFile(deepFileDescriptor(), nil)
	if err != nil {
		b.Fatal(err)
	}

	// Pre-compile all message types for hyperpb caching.
	for i := 0; i < fd.Messages().Len(); i++ {
		lib.CompileMessageDescriptor(fd.Messages().Get(i))
	}

	// Spread data across 3 levels (roughly equal).
	perLevel := totalDataSize / 3
	remainder := totalDataSize - perLevel*3

	level3MD := fd.Messages().ByName("Level3")
	level3 := dynamicpb.NewMessage(level3MD)
	level3.Set(level3MD.Fields().ByName("data"),
		protoreflect.ValueOfString(strings.Repeat("3", perLevel)))

	level2MD := fd.Messages().ByName("Level2")
	level2 := dynamicpb.NewMessage(level2MD)
	level2.Set(level2MD.Fields().ByName("child"),
		protoreflect.ValueOfMessage(level3))
	level2.Set(level2MD.Fields().ByName("data"),
		protoreflect.ValueOfString(strings.Repeat("2", perLevel)))

	level1MD := fd.Messages().ByName("Level1")
	level1 := dynamicpb.NewMessage(level1MD)
	level1.Set(level1MD.Fields().ByName("child"),
		protoreflect.ValueOfMessage(level2))
	level1.Set(level1MD.Fields().ByName("data"),
		protoreflect.ValueOfString(strings.Repeat("1", perLevel+remainder)))

	deepMD := fd.Messages().ByName("DeepPing")
	msg := dynamicpb.NewMessage(deepMD)
	msg.Set(deepMD.Fields().ByName("id"),
		protoreflect.ValueOfString("routing-key-12345"))
	msg.Set(deepMD.Fields().ByName("payload"),
		protoreflect.ValueOfMessage(level1))

	data, err := proto.Marshal(msg)
	if err != nil {
		b.Fatal(err)
	}
	return data
}

// ============================================================
// Gogo hand-rolled types for deep nesting
// ============================================================

type gogoLevel3 struct {
	Data string `protobuf:"bytes,1,opt,name=data,proto3"`
}

func (m *gogoLevel3) Reset()         { *m = gogoLevel3{} }
func (m *gogoLevel3) String() string { return m.Data }
func (m *gogoLevel3) ProtoMessage()  {}

type gogoLevel2 struct {
	Child *gogoLevel3 `protobuf:"bytes,1,opt,name=child,proto3"`
	Data  string      `protobuf:"bytes,2,opt,name=data,proto3"`
}

func (m *gogoLevel2) Reset()         { *m = gogoLevel2{} }
func (m *gogoLevel2) String() string { return m.Data }
func (m *gogoLevel2) ProtoMessage()  {}

type gogoLevel1 struct {
	Child *gogoLevel2 `protobuf:"bytes,1,opt,name=child,proto3"`
	Data  string      `protobuf:"bytes,2,opt,name=data,proto3"`
}

func (m *gogoLevel1) Reset()         { *m = gogoLevel1{} }
func (m *gogoLevel1) String() string { return m.Data }
func (m *gogoLevel1) ProtoMessage()  {}

type gogoDeepPing struct {
	Id      string      `protobuf:"bytes,1,opt,name=id,proto3"`
	Payload *gogoLevel1 `protobuf:"bytes,2,opt,name=payload,proto3"`
}

func (m *gogoDeepPing) Reset()         { *m = gogoDeepPing{} }
func (m *gogoDeepPing) String() string { return m.Id }
func (m *gogoDeepPing) ProtoMessage()  {}

// Marshal / MarshalToSizedBuffer / Size — hand-rolled for gogoDeepPing

func (m *gogoLevel3) Size() int {
	l := len(m.Data)
	if l == 0 {
		return 0
	}
	return 1 + gogoVarIntSize(uint64(l)) + l
}

func (m *gogoLevel3) MarshalToSizedBuffer(buf []byte) (int, error) {
	i := len(buf)
	if len(m.Data) > 0 {
		i -= len(m.Data)
		copy(buf[i:], m.Data)
		i = gogoAppendVarintReverse(buf, i, uint64(len(m.Data)))
		i--
		buf[i] = 0x0a // field 1, wire type 2
	}
	return len(buf) - i, nil
}

func (m *gogoLevel2) Size() int {
	n := 0
	if m.Child != nil {
		cs := m.Child.Size()
		if cs > 0 {
			n += 1 + gogoVarIntSize(uint64(cs)) + cs
		}
	}
	l := len(m.Data)
	if l > 0 {
		n += 1 + gogoVarIntSize(uint64(l)) + l
	}
	return n
}

func (m *gogoLevel2) MarshalToSizedBuffer(buf []byte) (int, error) {
	i := len(buf)
	if len(m.Data) > 0 {
		i -= len(m.Data)
		copy(buf[i:], m.Data)
		i = gogoAppendVarintReverse(buf, i, uint64(len(m.Data)))
		i--
		buf[i] = 0x12 // field 2, wire type 2
	}
	if m.Child != nil {
		cs := m.Child.Size()
		if cs > 0 {
			n, err := m.Child.MarshalToSizedBuffer(buf[i-cs : i])
			if err != nil {
				return 0, err
			}
			i -= n
			i = gogoAppendVarintReverse(buf, i, uint64(n))
			i--
			buf[i] = 0x0a // field 1, wire type 2
		}
	}
	return len(buf) - i, nil
}

func (m *gogoLevel1) Size() int {
	n := 0
	if m.Child != nil {
		cs := m.Child.Size()
		if cs > 0 {
			n += 1 + gogoVarIntSize(uint64(cs)) + cs
		}
	}
	l := len(m.Data)
	if l > 0 {
		n += 1 + gogoVarIntSize(uint64(l)) + l
	}
	return n
}

func (m *gogoLevel1) MarshalToSizedBuffer(buf []byte) (int, error) {
	i := len(buf)
	if len(m.Data) > 0 {
		i -= len(m.Data)
		copy(buf[i:], m.Data)
		i = gogoAppendVarintReverse(buf, i, uint64(len(m.Data)))
		i--
		buf[i] = 0x12 // field 2, wire type 2
	}
	if m.Child != nil {
		cs := m.Child.Size()
		if cs > 0 {
			n, err := m.Child.MarshalToSizedBuffer(buf[i-cs : i])
			if err != nil {
				return 0, err
			}
			i -= n
			i = gogoAppendVarintReverse(buf, i, uint64(n))
			i--
			buf[i] = 0x0a // field 1, wire type 2
		}
	}
	return len(buf) - i, nil
}

func (m *gogoDeepPing) Size() int {
	n := 0
	if len(m.Id) > 0 {
		n += 1 + gogoVarIntSize(uint64(len(m.Id))) + len(m.Id)
	}
	if m.Payload != nil {
		ps := m.Payload.Size()
		if ps > 0 {
			n += 1 + gogoVarIntSize(uint64(ps)) + ps
		}
	}
	return n
}

func (m *gogoDeepPing) Marshal() ([]byte, error) {
	size := m.Size()
	buf := make([]byte, size)
	n, err := m.MarshalToSizedBuffer(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (m *gogoDeepPing) MarshalTo(buf []byte) (int, error) {
	return m.MarshalToSizedBuffer(buf[:m.Size()])
}

func (m *gogoDeepPing) MarshalToSizedBuffer(buf []byte) (int, error) {
	i := len(buf)
	if m.Payload != nil {
		ps := m.Payload.Size()
		if ps > 0 {
			n, err := m.Payload.MarshalToSizedBuffer(buf[i-ps : i])
			if err != nil {
				return 0, err
			}
			i -= n
			i = gogoAppendVarintReverse(buf, i, uint64(n))
			i--
			buf[i] = 0x12 // field 2, wire type 2
		}
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(buf[i:], m.Id)
		i = gogoAppendVarintReverse(buf, i, uint64(len(m.Id)))
		i--
		buf[i] = 0x0a // field 1, wire type 2
	}
	return len(buf) - i, nil
}

// Unmarshal for gogoDeepPing — recursively parses nested messages.
func (m *gogoDeepPing) Unmarshal(data []byte) error {
	return gogoUnmarshalDeepPing(data, m)
}

func gogoUnmarshalDeepPing(data []byte, m *gogoDeepPing) error {
	m.Id = ""
	m.Payload = nil
	i := 0
	for i < len(data) {
		tag, n := binary.Uvarint(data[i:])
		if n <= 0 {
			return fmt.Errorf("bad varint at offset %d", i)
		}
		i += n
		wireType := tag & 0x7
		fieldNumber := tag >> 3
		if wireType != 2 {
			return fmt.Errorf("unsupported wire type %d for field %d", wireType, fieldNumber)
		}
		length, n := binary.Uvarint(data[i:])
		if n <= 0 {
			return fmt.Errorf("bad varint at offset %d", i)
		}
		i += n
		switch fieldNumber {
		case 1: // id
			m.Id = string(data[i : i+int(length)])
		case 2: // payload
			m.Payload = &gogoLevel1{}
			if err := gogoUnmarshalLevel1(data[i:i+int(length)], m.Payload); err != nil {
				return err
			}
		}
		i += int(length)
	}
	return nil
}

func gogoUnmarshalLevel1(data []byte, m *gogoLevel1) error {
	i := 0
	for i < len(data) {
		tag, n := binary.Uvarint(data[i:])
		if n <= 0 {
			return fmt.Errorf("bad varint")
		}
		i += n
		if tag&0x7 != 2 {
			return fmt.Errorf("unsupported wire type")
		}
		length, n := binary.Uvarint(data[i:])
		if n <= 0 {
			return fmt.Errorf("bad varint")
		}
		i += n
		switch tag >> 3 {
		case 1:
			m.Child = &gogoLevel2{}
			if err := gogoUnmarshalLevel2(data[i:i+int(length)], m.Child); err != nil {
				return err
			}
		case 2:
			m.Data = string(data[i : i+int(length)])
		}
		i += int(length)
	}
	return nil
}

func gogoUnmarshalLevel2(data []byte, m *gogoLevel2) error {
	i := 0
	for i < len(data) {
		tag, n := binary.Uvarint(data[i:])
		if n <= 0 {
			return fmt.Errorf("bad varint")
		}
		i += n
		if tag&0x7 != 2 {
			return fmt.Errorf("unsupported wire type")
		}
		length, n := binary.Uvarint(data[i:])
		if n <= 0 {
			return fmt.Errorf("bad varint")
		}
		i += n
		switch tag >> 3 {
		case 1:
			m.Child = &gogoLevel3{}
			if err := gogoUnmarshalLevel3(data[i:i+int(length)], m.Child); err != nil {
				return err
			}
		case 2:
			m.Data = string(data[i : i+int(length)])
		}
		i += int(length)
	}
	return nil
}

func gogoUnmarshalLevel3(data []byte, m *gogoLevel3) error {
	i := 0
	for i < len(data) {
		tag, n := binary.Uvarint(data[i:])
		if n <= 0 {
			return fmt.Errorf("bad varint")
		}
		i += n
		if tag&0x7 != 2 {
			return fmt.Errorf("unsupported wire type")
		}
		length, n := binary.Uvarint(data[i:])
		if n <= 0 {
			return fmt.Errorf("bad varint")
		}
		i += n
		if tag>>3 == 1 {
			m.Data = string(data[i : i+int(length)])
		}
		i += int(length)
	}
	return nil
}

// ============================================================
// Helper for gogo varint encoding (reverse direction)
// ============================================================

func gogoVarIntSize(x uint64) int {
	n := 1
	for x >= 0x80 {
		n++
		x >>= 7
	}
	return n
}

func gogoAppendVarintReverse(buf []byte, i int, x uint64) int {
	for x >= 0x80 {
		i--
		buf[i] = byte(x) | 0x80
		x >>= 7
	}
	i--
	buf[i] = byte(x)
	return i
}

// ============================================================
// Handlers for deep nesting benchmarks
// ============================================================

// Gogo — full echo (unmarshal all, marshal all)
func makeDeepGogoHandler() transport.UnaryHandler {
	return gogoencoding.NewUnaryHandler(gogoencoding.UnaryHandlerParams{
		Handle: func(ctx context.Context, req gogoproto.Message) (gogoproto.Message, error) {
			return req, nil
		},
		NewRequest: func() gogoproto.Message {
			return &gogoDeepPing{}
		},
	})
}

// Gogo — partial (unmarshal all, but handler only reads id, still must marshal all)
func makeDeepGogoPartialHandler() transport.UnaryHandler {
	return gogoencoding.NewUnaryHandler(gogoencoding.UnaryHandlerParams{
		Handle: func(ctx context.Context, req gogoproto.Message) (gogoproto.Message, error) {
			// Simulate reading just the routing key.
			msg := req.(*gogoDeepPing)
			_ = msg.Id
			return req, nil
		},
		NewRequest: func() gogoproto.Message {
			return &gogoDeepPing{}
		},
	})
}

// Hyperpb — full echo (unmarshal all, marshal all)
func makeDeepHyperpbHandler(b *testing.B) transport.UnaryHandler {
	b.Helper()
	return makeDeepHyperpbHandlerWithFunc(b, func(ctx context.Context, req proto.Message) (proto.Message, error) {
		return req, nil
	})
}

// Hyperpb — partial (unmarshal lazily, only access "id", echo whole message)
func makeDeepHyperpbPartialHandler(b *testing.B) transport.UnaryHandler {
	b.Helper()
	return makeDeepHyperpbHandlerWithFunc(b, func(ctx context.Context, req proto.Message) (proto.Message, error) {
		// Only access the "id" field — the nested payload stays lazy.
		// Must get the field descriptor from the message's own descriptor
		// since hyperpb uses compiled types.
		idField := req.ProtoReflect().Descriptor().Fields().ByName("id")
		req.ProtoReflect().Get(idField)
		return req, nil
	})
}

func makeDeepHyperpbHandlerWithFunc(b *testing.B, handler hyperpb.UnaryHandler) transport.UnaryHandler {
	b.Helper()
	fd, err := protodesc.NewFile(deepFileDescriptor(), nil)
	if err != nil {
		b.Fatal(err)
	}
	sd := fd.Services().ByName("DeepEcho")

	registry := hyperpb.NewTypeRegistry()
	registry.RegisterServiceDescriptor(sd)

	procs, err := hyperpb.BuildProceduresFromDescriptor(sd, registry, map[string]hyperpb.UnaryHandler{
		"Ping": handler,
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

// Golang proto (google protobuf-go via v2 encoding) — uses dynamicpb.Message
func makeDeepGolangProtoHandler(b *testing.B) transport.UnaryHandler {
	b.Helper()
	fd, err := protodesc.NewFile(deepFileDescriptor(), nil)
	if err != nil {
		b.Fatal(err)
	}
	reqMD := fd.Messages().ByName("DeepPing")
	return v2encoding.NewUnaryHandler(v2encoding.UnaryHandlerParams{
		Handle: func(ctx context.Context, req proto.Message) (proto.Message, error) {
			return req, nil
		},
		NewRequest: func() proto.Message {
			return dynamicpb.NewMessage(reqMD)
		},
	})
}

// Golang proto — partial (only reads id field)
func makeDeepGolangProtoPartialHandler(b *testing.B) transport.UnaryHandler {
	b.Helper()
	fd, err := protodesc.NewFile(deepFileDescriptor(), nil)
	if err != nil {
		b.Fatal(err)
	}
	reqMD := fd.Messages().ByName("DeepPing")
	idField := reqMD.Fields().ByName("id")
	return v2encoding.NewUnaryHandler(v2encoding.UnaryHandlerParams{
		Handle: func(ctx context.Context, req proto.Message) (proto.Message, error) {
			req.ProtoReflect().Get(idField)
			return req, nil
		},
		NewRequest: func() proto.Message {
			return dynamicpb.NewMessage(reqMD)
		},
	})
}

// RawProto — zero-parse passthrough
func makeDeepRawProtoHandler() transport.UnaryHandler {
	procs := rawproto.Procedure("deep.v1.DeepEcho::Ping", func(ctx context.Context, body []byte) ([]byte, error) {
		return body, nil
	})
	return procs[0].HandlerSpec.Unary()
}

// ============================================================
// Deep nesting benchmark sizes
// ============================================================

var deepBenchSizes = []struct {
	name string
	size int
}{
	{"10KB", 10 * 1024},
	{"100KB", 100 * 1024},
	{"1MB", 1024 * 1024},
}

// ============================================================
// Benchmarks
// ============================================================

func runDeepBench(b *testing.B, handler transport.UnaryHandler, reqData []byte) {
	b.Helper()
	b.ReportAllocs()
	ctx := context.Background()
	reader := bytes.NewReader(nil)
	for i := 0; i < b.N; i++ {
		reader.Reset(reqData)
		req := &transport.Request{
			Caller:    "bench-caller",
			Service:   "bench-service",
			Procedure: "deep.v1.DeepEcho::Ping",
			Encoding:  "proto",
			Body:      reader,
		}
		rw := &transporttest.FakeResponseWriter{}
		if err := handler.Handle(ctx, req, rw); err != nil {
			b.Fatal(err)
		}
	}
}

// — Full echo: unmarshal everything, marshal everything —

func BenchmarkDeep_Baseline(b *testing.B) {
	h := baselineHandler{}
	for _, sz := range deepBenchSizes {
		reqData := makeDeepPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runDeepBench(b, h, reqData)
		})
	}
}

func BenchmarkDeep_Gogo_Echo(b *testing.B) {
	h := makeDeepGogoHandler()
	for _, sz := range deepBenchSizes {
		reqData := makeDeepPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runDeepBench(b, h, reqData)
		})
	}
}

func BenchmarkDeep_Hyperpb_Echo(b *testing.B) {
	h := makeDeepHyperpbHandler(b)
	for _, sz := range deepBenchSizes {
		reqData := makeDeepPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runDeepBench(b, h, reqData)
		})
	}
}

func BenchmarkDeep_GolangProto_Echo(b *testing.B) {
	h := makeDeepGolangProtoHandler(b)
	for _, sz := range deepBenchSizes {
		reqData := makeDeepPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runDeepBench(b, h, reqData)
		})
	}
}

func BenchmarkDeep_RawProto(b *testing.B) {
	h := makeDeepRawProtoHandler()
	for _, sz := range deepBenchSizes {
		reqData := makeDeepPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runDeepBench(b, h, reqData)
		})
	}
}

// — Partial access: handler only reads "id", ignores nested payload —

func BenchmarkDeep_Gogo_Partial(b *testing.B) {
	h := makeDeepGogoPartialHandler()
	for _, sz := range deepBenchSizes {
		reqData := makeDeepPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runDeepBench(b, h, reqData)
		})
	}
}

func BenchmarkDeep_Hyperpb_Partial(b *testing.B) {
	h := makeDeepHyperpbPartialHandler(b)
	for _, sz := range deepBenchSizes {
		reqData := makeDeepPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runDeepBench(b, h, reqData)
		})
	}
}

func BenchmarkDeep_GolangProto_Partial(b *testing.B) {
	h := makeDeepGolangProtoPartialHandler(b)
	for _, sz := range deepBenchSizes {
		reqData := makeDeepPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runDeepBench(b, h, reqData)
		})
	}
}

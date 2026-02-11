// Copyright (c) 2026 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package hyperpb

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
	"go.uber.org/yarpc/encoding/protobuf"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// gogoTestMessage is a minimal gogo-compatible protobuf message with the
// same schema as our test descriptors: message { string value = 1; }
//
// It has hand-rolled Marshal/Unmarshal to match what gogoproto codegen
// produces, for a fair benchmark.
type gogoTestMessage struct {
	Value string `protobuf:"bytes,1,opt,name=value,proto3"`
}

func (m *gogoTestMessage) Reset()         { *m = gogoTestMessage{} }
func (m *gogoTestMessage) String() string { return m.Value }
func (m *gogoTestMessage) ProtoMessage()  {}

func (m *gogoTestMessage) Marshal() ([]byte, error) {
	size := m.Size()
	buf := make([]byte, size)
	n, err := m.MarshalToSizedBuffer(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (m *gogoTestMessage) MarshalTo(buf []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(buf[:size])
}

func (m *gogoTestMessage) MarshalToSizedBuffer(buf []byte) (int, error) {
	i := len(buf)
	if len(m.Value) > 0 {
		i -= len(m.Value)
		copy(buf[i:], m.Value)
		// encode varint length
		l := uint64(len(m.Value))
		for l >= 0x80 {
			i--
			buf[i] = byte(l) | 0x80
			l >>= 7
		}
		i--
		buf[i] = byte(l)
		i--
		buf[i] = 0x0a // field 1, wire type 2
	}
	return len(buf) - i, nil
}

func (m *gogoTestMessage) Size() int {
	n := 0
	l := len(m.Value)
	if l > 0 {
		n += 1 + l + sovSize(uint64(l))
	}
	return n
}

func (m *gogoTestMessage) Unmarshal(data []byte) error {
	m.Value = ""
	i := 0
	for i < len(data) {
		tag, n := binary.Uvarint(data[i:])
		if n <= 0 {
			return fmt.Errorf("bad varint for tag")
		}
		i += n
		fieldNumber := tag >> 3
		wireType := tag & 0x7
		switch wireType {
		case 2: // length-delimited
			length, n := binary.Uvarint(data[i:])
			if n <= 0 {
				return fmt.Errorf("bad varint for length")
			}
			i += n
			if fieldNumber == 1 {
				m.Value = string(data[i : i+int(length)])
			}
			i += int(length)
		case 0: // varint – skip
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

func sovSize(x uint64) int {
	n := 0
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}

// ============================================================
// Benchmark: Pure Marshal/Unmarshal
// ============================================================

// benchSizes defines the message sizes to benchmark.
var benchSizes = []struct {
	name string
	size int
}{
	{"Small_350B", 300},
	{"Medium_10KB", 10 * 1024},
	{"Large_1MB", 1024 * 1024},
}

func BenchmarkMarshal_Gogo(b *testing.B) {
	for _, sz := range benchSizes {
		b.Run(sz.name, func(b *testing.B) {
			msg := &gogoTestMessage{Value: strings.Repeat("x", sz.size)}
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := gogoproto.Marshal(msg)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkMarshal_Hyperpb(b *testing.B) {
	fdProto := buildTestFileDescriptor()
	fd, err := buildFileDescriptor(fdProto)
	if err != nil {
		b.Fatal(err)
	}
	reqMD := fd.Messages().ByName("EchoRequest")
	valueField := reqMD.Fields().ByName("value")
	msgType := lib.CompileMessageDescriptor(reqMD)

	for _, sz := range benchSizes {
		b.Run(sz.name, func(b *testing.B) {
			// Build bytes via dynamicpb, then unmarshal into hyperpb.
			dynMsg := dynamicpb.NewMessage(reqMD)
			dynMsg.Set(valueField, protoreflect.ValueOfString(strings.Repeat("x", sz.size)))
			data, err := proto.Marshal(dynMsg)
			if err != nil {
				b.Fatal(err)
			}
			msg := lib.NewMessage(msgType)
			if err := proto.Unmarshal(data, msg); err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := proto.Marshal(msg)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkUnmarshal_Gogo(b *testing.B) {
	for _, sz := range benchSizes {
		b.Run(sz.name, func(b *testing.B) {
			msg := &gogoTestMessage{Value: strings.Repeat("x", sz.size)}
			data, err := gogoproto.Marshal(msg)
			if err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				out := &gogoTestMessage{}
				if err := gogoproto.Unmarshal(data, out); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkUnmarshal_Hyperpb(b *testing.B) {
	fdProto := buildTestFileDescriptor()
	fd, err := buildFileDescriptor(fdProto)
	if err != nil {
		b.Fatal(err)
	}
	reqMD := fd.Messages().ByName("EchoRequest")
	valueField := reqMD.Fields().ByName("value")

	// Compile the hyperpb type.
	msgType := lib.CompileMessageDescriptor(reqMD)

	for _, sz := range benchSizes {
		b.Run(sz.name, func(b *testing.B) {
			// Serialize using dynamicpb so we have valid proto bytes.
			dynMsg := dynamicpb.NewMessage(reqMD)
			dynMsg.Set(valueField, protoreflect.ValueOfString(strings.Repeat("x", sz.size)))
			data, err := proto.Marshal(dynMsg)
			if err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				msg := lib.NewMessage(msgType)
				if err := proto.Unmarshal(data, msg); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// ============================================================
// Benchmark: YARPC Handler-level round-trip
// ============================================================

func BenchmarkHandlerRoundTrip_Gogo(b *testing.B) {
	for _, sz := range benchSizes {
		b.Run(sz.name, func(b *testing.B) {
			benchGogoHandler(b, sz.size)
		})
	}
}

func BenchmarkHandlerRoundTrip_Hyperpb(b *testing.B) {
	for _, sz := range benchSizes {
		b.Run(sz.name, func(b *testing.B) {
			benchHyperpbHandler(b, sz.size)
		})
	}
}

func benchGogoHandler(b *testing.B, payloadSize int) {
	// Build a gogo handler using the real YARPC encoding/protobuf package.
	handler := protobuf.NewUnaryHandler(protobuf.UnaryHandlerParams{
		Handle: func(ctx context.Context, req gogoproto.Message) (gogoproto.Message, error) {
			// Echo: return the request as-is.
			return req, nil
		},
		NewRequest: func() gogoproto.Message {
			return &gogoTestMessage{}
		},
	})

	// Serialize the request.
	reqMsg := &gogoTestMessage{Value: strings.Repeat("x", payloadSize)}
	reqData, err := gogoproto.Marshal(reqMsg)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		transportReq := &transport.Request{
			Caller:    "bench-caller",
			Service:   "bench-service",
			Procedure: "Test::Echo",
			Encoding:  "proto",
			Body:      bytes.NewReader(reqData),
		}
		resw := &transporttest.FakeResponseWriter{}
		if err := handler.Handle(context.Background(), transportReq, resw); err != nil {
			b.Fatal(err)
		}
	}
}

func benchHyperpbHandler(b *testing.B, payloadSize int) {
	fdProto := buildTestFileDescriptor()
	fd, err := buildFileDescriptor(fdProto)
	if err != nil {
		b.Fatal(err)
	}
	reqMD := fd.Messages().ByName("EchoRequest")
	valueField := reqMD.Fields().ByName("value")
	sd := fd.Services().ByName("Echo")

	registry := NewTypeRegistry()
	registry.RegisterServiceDescriptor(sd)

	// Echo handler: return the hyperpb request as-is. This is valid because
	// hyperpb messages support proto.Marshal (marshalling is a read operation).
	echoHandler := func(ctx context.Context, req proto.Message) (proto.Message, error) {
		return req, nil
	}

	procedures, err := BuildProceduresFromDescriptor(sd, registry, map[string]UnaryHandler{
		"Ping": echoHandler,
	})
	if err != nil {
		b.Fatal(err)
	}

	// Find proto procedure.
	var handler transport.UnaryHandler
	for _, p := range procedures {
		if p.Encoding == Encoding {
			handler = p.HandlerSpec.Unary()
			break
		}
	}

	// Serialize a request.
	dynMsg := dynamicpb.NewMessage(reqMD)
	dynMsg.Set(valueField, protoreflect.ValueOfString(strings.Repeat("x", payloadSize)))
	reqData, err := proto.Marshal(dynMsg)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		transportReq := &transport.Request{
			Caller:    "bench-caller",
			Service:   "bench-service",
			Procedure: "test.v1.Echo::Ping",
			Encoding:  "proto",
			Body:      bytes.NewReader(reqData),
		}
		resw := &transporttest.FakeResponseWriter{}
		if err := handler.Handle(context.Background(), transportReq, resw); err != nil {
			b.Fatal(err)
		}
	}
}

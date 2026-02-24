package v2_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	gogoproto "github.com/gogo/protobuf/proto"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	gogoencoding "go.uber.org/yarpc/encoding/protobuf"
	gogotestpb "go.uber.org/yarpc/encoding/protobuf/internal/testpb"
	gogoprobepb "go.uber.org/yarpc/encoding/protobuf/internal/testpb/probepb"
	testpb "go.uber.org/yarpc/encoding/protobuf/internal/testpb/v2"
	probepb "go.uber.org/yarpc/encoding/protobuf/internal/testpb/v2/probepb"
	v2 "go.uber.org/yarpc/encoding/protobuf/v2"
	"google.golang.org/protobuf/proto"
)

// This benchmark compares gogo (protoc-gen-gogo) vs golang proto
// (protoc-gen-go) using REAL generated types — no dynamicpb.
//
// Both testpb.TestMessage types have the same schema:
//   message TestMessage { string value = 1; }

var genBenchSizes = []struct {
	name string
	size int
}{
	{"Small_350B", 300},
	{"Medium_10KB", 10 * 1024},
	{"Large_1MB", 1024 * 1024},
}

func runGenBench(b *testing.B, handler transport.UnaryHandler, reqData []byte) {
	b.Helper()
	b.ReportAllocs()
	ctx := context.Background()
	reader := bytes.NewReader(nil)
	for i := 0; i < b.N; i++ {
		reader.Reset(reqData)
		req := &transport.Request{
			Caller:    "bench-caller",
			Service:   "bench-service",
			Procedure: "uber.yarpc.encoding.protobuf.Test::Unary",
			Encoding:  "proto",
			Body:      reader,
		}
		rw := &transporttest.FakeResponseWriter{}
		if err := handler.Handle(ctx, req, rw); err != nil {
			b.Fatal(err)
		}
	}
}

// makeGenPayload creates a proto-encoded TestMessage with the given data size.
// Uses the v2 (protobuf-go) generated type to marshal — wire-compatible.
func makeGenPayload(b *testing.B, size int) []byte {
	b.Helper()
	msg := &testpb.TestMessage{
		Value: strings.Repeat("x", size),
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		b.Fatal(err)
	}
	return data
}

// ============================================================
// Gogo — real generated types (protoc-gen-gogo)
// ============================================================

func BenchmarkGenerated_Gogo(b *testing.B) {
	h := gogoencoding.NewUnaryHandler(gogoencoding.UnaryHandlerParams{
		Handle: func(ctx context.Context, req gogoproto.Message) (gogoproto.Message, error) {
			return req, nil
		},
		NewRequest: func() gogoproto.Message {
			return &gogotestpb.TestMessage{}
		},
	})
	for _, sz := range genBenchSizes {
		reqData := makeGenPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runGenBench(b, h, reqData)
		})
	}
}

// ============================================================
// Golang Proto — real generated types (protoc-gen-go)
// ============================================================

func BenchmarkGenerated_GolangProto(b *testing.B) {
	h := v2.NewUnaryHandler(v2.UnaryHandlerParams{
		Handle: func(ctx context.Context, req proto.Message) (proto.Message, error) {
			return req, nil
		},
		NewRequest: func() proto.Message {
			return &testpb.TestMessage{}
		},
	})
	for _, sz := range genBenchSizes {
		reqData := makeGenPayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runGenBench(b, h, reqData)
		})
	}
}

// ============================================================
// Deep nesting benchmarks — real Probe/Ping/Pong types
// ============================================================

// makeDeepProbePayload builds a serialized EchoRequest with a deeply
// nested Ping payload of approximately the given total data size.
// The data is spread across nested PingMessage and PingString fields.
func makeDeepProbePayload(b *testing.B, totalDataSize int) []byte {
	b.Helper()
	// Build a deeply nested Ping: 3 levels of ping_message nesting,
	// each level carrying a chunk of data in ping_string and ping_bytes.
	perLevel := totalDataSize / 4
	remainder := totalDataSize - perLevel*4

	level3 := &probepb.Ping{
		PingString: strings.Repeat("3", perLevel),
		PingBytes:  []byte(strings.Repeat("b", perLevel)),
		PingInt32:  42,
		PingBool:   true,
		PingEnum:   probepb.PingEnum_PING_ENUM_PING,
	}
	level2 := &probepb.Ping{
		PingString:  strings.Repeat("2", perLevel),
		PingMessage: level3,
		PingInt64:   1234567890,
		PingDouble:  3.14159,
	}
	level1 := &probepb.Ping{
		PingString:  strings.Repeat("1", perLevel+remainder),
		PingMessage: level2,
		PingFloat:   2.718,
		PingOneOf:   &probepb.Ping_PingOneOfString{PingOneOfString: "oneof-value"},
	}

	msg := &probepb.EchoRequest{
		Payload:          level1,
		RespondWithError: false,
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		b.Fatal(err)
	}
	return data
}

var deepProbeSizes = []struct {
	name string
	size int
}{
	{"10KB", 10 * 1024},
	{"100KB", 100 * 1024},
	{"1MB", 1024 * 1024},
}

func runDeepProbeBench(b *testing.B, handler transport.UnaryHandler, reqData []byte) {
	b.Helper()
	b.ReportAllocs()
	ctx := context.Background()
	reader := bytes.NewReader(nil)
	for i := 0; i < b.N; i++ {
		reader.Reset(reqData)
		req := &transport.Request{
			Caller:    "bench-caller",
			Service:   "bench-service",
			Procedure: "uber.infra.net.rpc.probe.Probe::Echo",
			Encoding:  "proto",
			Body:      reader,
		}
		rw := &transporttest.FakeResponseWriter{}
		if err := handler.Handle(ctx, req, rw); err != nil {
			b.Fatal(err)
		}
	}
}

// Deep — Gogo echo (full unmarshal + marshal)
func BenchmarkDeepProbe_Gogo_Echo(b *testing.B) {
	h := gogoencoding.NewUnaryHandler(gogoencoding.UnaryHandlerParams{
		Handle: func(ctx context.Context, req gogoproto.Message) (gogoproto.Message, error) {
			return req, nil
		},
		NewRequest: func() gogoproto.Message {
			return &gogoprobepb.EchoRequest{}
		},
	})
	for _, sz := range deepProbeSizes {
		reqData := makeDeepProbePayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runDeepProbeBench(b, h, reqData)
		})
	}
}

// Deep — Golang Proto echo (full unmarshal + marshal)
func BenchmarkDeepProbe_GolangProto_Echo(b *testing.B) {
	h := v2.NewUnaryHandler(v2.UnaryHandlerParams{
		Handle: func(ctx context.Context, req proto.Message) (proto.Message, error) {
			return req, nil
		},
		NewRequest: func() proto.Message {
			return &probepb.EchoRequest{}
		},
	})
	for _, sz := range deepProbeSizes {
		reqData := makeDeepProbePayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runDeepProbeBench(b, h, reqData)
		})
	}
}

// Deep — Gogo partial (unmarshal all, but handler only reads one field)
func BenchmarkDeepProbe_Gogo_Partial(b *testing.B) {
	h := gogoencoding.NewUnaryHandler(gogoencoding.UnaryHandlerParams{
		Handle: func(ctx context.Context, req gogoproto.Message) (gogoproto.Message, error) {
			msg := req.(*gogoprobepb.EchoRequest)
			_ = msg.RespondWithError // only read a top-level scalar
			return req, nil
		},
		NewRequest: func() gogoproto.Message {
			return &gogoprobepb.EchoRequest{}
		},
	})
	for _, sz := range deepProbeSizes {
		reqData := makeDeepProbePayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runDeepProbeBench(b, h, reqData)
		})
	}
}

// Deep — Golang Proto partial (unmarshal all, handler only reads one field)
func BenchmarkDeepProbe_GolangProto_Partial(b *testing.B) {
	h := v2.NewUnaryHandler(v2.UnaryHandlerParams{
		Handle: func(ctx context.Context, req proto.Message) (proto.Message, error) {
			msg := req.(*probepb.EchoRequest)
			_ = msg.RespondWithError // only read a top-level scalar
			return req, nil
		},
		NewRequest: func() proto.Message {
			return &probepb.EchoRequest{}
		},
	})
	for _, sz := range deepProbeSizes {
		reqData := makeDeepProbePayload(b, sz.size)
		b.Run(sz.name, func(b *testing.B) {
			runDeepProbeBench(b, h, reqData)
		})
	}
}

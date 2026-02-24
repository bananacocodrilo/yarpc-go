package rawproto

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
)

func TestProcedure(t *testing.T) {
	handler := func(ctx context.Context, body []byte) ([]byte, error) {
		return body, nil
	}
	procs := Procedure("MyService::Echo", handler)
	require.Len(t, procs, 1)
	assert.Equal(t, "MyService::Echo", procs[0].Name)
	assert.Equal(t, Encoding, procs[0].Encoding)
}

func TestHandle_Echo(t *testing.T) {
	handler := func(ctx context.Context, body []byte) ([]byte, error) {
		return body, nil
	}
	procs := Procedure("Echo::Ping", handler)
	h := procs[0].HandlerSpec.Unary()

	payload := []byte("hello world")
	req := &transport.Request{
		Caller:    "test-caller",
		Service:   "test-service",
		Procedure: "Echo::Ping",
		Encoding:  "proto",
		Body:      bytes.NewReader(payload),
	}
	rw := &transporttest.FakeResponseWriter{}
	err := h.Handle(context.Background(), req, rw)
	require.NoError(t, err)
	assert.Equal(t, payload, rw.Body.Bytes())
}

func TestHandle_AcceptsAnyEncoding(t *testing.T) {
	handler := func(ctx context.Context, body []byte) ([]byte, error) {
		return body, nil
	}
	procs := Procedure("Echo::Ping", handler)
	h := procs[0].HandlerSpec.Unary()

	for _, enc := range []transport.Encoding{"proto", "json", "raw", "thrift"} {
		t.Run(string(enc), func(t *testing.T) {
			req := &transport.Request{
				Caller:    "test-caller",
				Service:   "test-service",
				Procedure: "Echo::Ping",
				Encoding:  enc,
				Body:      bytes.NewReader([]byte("data")),
			}
			rw := &transporttest.FakeResponseWriter{}
			err := h.Handle(context.Background(), req, rw)
			require.NoError(t, err)
			assert.Equal(t, []byte("data"), rw.Body.Bytes())
		})
	}
}

func TestOnewayProcedure(t *testing.T) {
	var received []byte
	handler := func(ctx context.Context, body []byte) error {
		received = body
		return nil
	}
	procs := OnewayProcedure("Fire::Event", handler)
	require.Len(t, procs, 1)
	h := procs[0].HandlerSpec.Oneway()

	req := &transport.Request{
		Caller:    "test-caller",
		Service:   "test-service",
		Procedure: "Fire::Event",
		Encoding:  "proto",
		Body:      bytes.NewReader([]byte("event")),
	}
	err := h.HandleOneway(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, []byte("event"), received)
}

var benchSizes = []struct {
	name string
	size int
}{
	{"Small_350B", 300},
	{"Medium_10KB", 10 * 1024},
	{"Large_1MB", 1024 * 1024},
}

func BenchmarkHandlerRoundTrip_RawProto(b *testing.B) {
	handler := func(ctx context.Context, body []byte) ([]byte, error) {
		return body, nil
	}
	procs := Procedure("Echo::Ping", handler)
	h := procs[0].HandlerSpec.Unary()

	for _, sz := range benchSizes {
		b.Run(sz.name, func(b *testing.B) {
			payload := []byte(strings.Repeat("x", sz.size))
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				req := &transport.Request{
					Caller:    "bench-caller",
					Service:   "bench-service",
					Procedure: "Echo::Ping",
					Encoding:  "proto",
					Body:      bytes.NewReader(payload),
				}
				rw := &transporttest.FakeResponseWriter{}
				if err := h.Handle(context.Background(), req, rw); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

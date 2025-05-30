// Copyright (c) 2025 Uber Technologies, Inc.
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

package grpc_test

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/encoding/protobuf"
	"go.uber.org/yarpc/peer/direct"
	"go.uber.org/yarpc/transport/grpc"
	. "go.uber.org/yarpc/x/yarpctest"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestStreaming(t *testing.T) {
	newDirectChooser := func(id peer.Identifier, transport peer.Transport) (peer.Chooser, error) {
		trans, ok := transport.(*grpc.Transport)
		if !ok {
			return nil, fmt.Errorf("transport was not a grpc.Transport")
		}
		return direct.New(direct.Configuration{}, trans.NewDialer())
	}
	protoerr := protobuf.NewError(yarpcerrors.CodeAborted, "test error",
		protobuf.WithErrorDetails(&types.StringValue{Value: "val"}))
	p := NewPortProvider(t)
	tests := []struct {
		name     string
		services Lifecycle
		requests Action
	}{
		{
			name: "stream requests",
			services: Lifecycles(
				GRPCService(
					Name("myservice"),
					p.NamedPort("1"),
					Proc(
						Name("proc"),
						EchoStreamHandler(),
					),
				),
			),
			requests: ConcurrentAction(
				RepeatAction(
					GRPCStreamRequest(
						p.NamedPort("1"),
						Service("myservice"),
						Procedure("proc"),
						ClientStreamActions(
							SendStreamMsg("test"),
							RecvStreamMsg("test"),
							SendStreamMsg("test2"),
							RecvStreamMsg("test2"),
							CloseStream(),
						),
					),
					10,
				),
				3,
			),
		},
		{
			name: "stream close from client",
			services: Lifecycles(
				GRPCService(
					Name("myservice"),
					p.NamedPort("2"),
					Proc(
						Name("proc"),
						OrderedStreamHandler(
							RecvStreamMsg("test"),
							SendStreamMsg("test1"),
							RecvStreamMsg("test2"),
							SendStreamMsg("test3"),
							RecvStreamErr(io.EOF.Error()),
							SendStreamMsg("test4"),
							StreamHandlerError(io.EOF),
						),
					),
				),
			),
			requests: Actions(
				GRPCStreamRequest(
					p.NamedPort("2"),
					Service("myservice"),
					Procedure("proc"),
					ClientStreamActions(
						SendStreamMsg("test"),
						RecvStreamMsg("test1"),
						SendStreamMsg("test2"),
						RecvStreamMsg("test3"),
						CloseStream(),
						RecvStreamMsg("test4"),
					),
				),
			),
		},
		{
			name: "stream close from server",
			services: Lifecycles(
				GRPCService(
					Name("myservice"),
					p.NamedPort("3"),
					Proc(
						Name("proc"),
						OrderedStreamHandler(
							RecvStreamMsg("test"),
							SendStreamMsg("test1"),
							RecvStreamMsg("test2"),
							SendStreamMsg("test3"),
						), // End of Stream
					),
				),
			),
			requests: Actions(
				GRPCStreamRequest(
					p.NamedPort("3"),
					Service("myservice"),
					Procedure("proc"),
					ClientStreamActions(
						SendStreamMsg("test"),
						RecvStreamMsg("test1"),
						SendStreamMsg("test2"),
						RecvStreamMsg("test3"),
						RecvStreamErr(io.EOF.Error()),
					),
				),
			),
		},
		{
			name: "stream close from server with error",
			services: Lifecycles(
				GRPCService(
					Name("myservice"),
					p.NamedPort("4"),
					Proc(
						Name("proc"),
						OrderedStreamHandler(
							RecvStreamMsg("test"),
							SendStreamMsg("test1"),
							RecvStreamMsg("test2"),
							SendStreamMsg("test3"),
							StreamHandlerError(yarpcerrors.InternalErrorf("myerroooooor")),
						),
					),
				),
			),
			requests: Actions(
				GRPCStreamRequest(
					p.NamedPort("4"),
					Service("myservice"),
					Procedure("proc"),
					ClientStreamActions(
						SendStreamMsg("test"),
						RecvStreamMsg("test1"),
						SendStreamMsg("test2"),
						RecvStreamMsg("test3"),
						RecvStreamErr(yarpcerrors.InternalErrorf("myerroooooor").Error()),
					),
				),
			),
		},
		{
			name: "stream recv after close",
			services: Lifecycles(
				GRPCService(
					Name("myservice"),
					p.NamedPort("5"),
					Proc(
						Name("proc"),
						OrderedStreamHandler(
							RecvStreamMsg("test"),
							RecvStreamErr(io.EOF.Error()),
							SendStreamMsg("test1"),
							SendStreamMsg("test2"),
							SendStreamMsg("test3"),
							StreamHandlerError(yarpcerrors.InternalErrorf("test")),
						),
					),
				),
			),
			requests: Actions(
				GRPCStreamRequest(
					p.NamedPort("5"),
					Service("myservice"),
					Procedure("proc"),
					ClientStreamActions(
						SendStreamMsg("test"),
						CloseStream(),
						SendStreamMsgAndExpectError("lala", io.EOF.Error()),
						RecvStreamMsg("test1"),
						RecvStreamMsg("test2"),
						RecvStreamMsg("test3"),
						RecvStreamErr(yarpcerrors.InternalErrorf("test").Error()),
					),
				),
			),
		},
		{
			name: "stream header test",
			services: Lifecycles(
				GRPCService(
					Name("myservice"),
					p.NamedPort("6"),
					Proc(
						Name("proc"),
						OrderedStreamHandler(
							StreamSendHeaders(map[string]string{"key": "value"}),
							WantHeader("req_key", "req_val"),
							WantHeader("req_key2", "req_val2"),
							RecvStreamMsg("test"),
						), // End of Stream
					),
				),
			),
			requests: Actions(
				GRPCStreamRequest(
					p.NamedPort("6"),
					Service("myservice"),
					Procedure("proc"),
					WithHeader("req_key", "req_val"),
					WithHeader("req_key2", "req_val2"),
					ClientStreamActions(
						WantHeaders(map[string]string{"key": "value"}),
						SendStreamMsg("test"),
						RecvStreamErr(io.EOF.Error()),
					),
				),
			),
		},
		{
			name: "stream invalid request",
			services: Lifecycles(
				GRPCService(
					Name("myservice"),
					p.NamedPort("7"),
				),
			),
			requests: Actions(
				GRPCStreamRequest(
					p.NamedPort("7"),
					Service("myservice"),
					Procedure("proc"),
					ClientStreamActions(
						RecvStreamErr(yarpcerrors.UnimplementedErrorf("unrecognized procedure \"proc\" for service \"myservice\"").Error()),
					),
				),
			),
		},
		{
			name: "stream invalid client decode",
			services: Lifecycles(
				GRPCService(
					Name("myservice"),
					p.NamedPort("8"),
				),
			),
			requests: Actions(
				GRPCStreamRequest(
					p.NamedPort("8"),
					Service("myservice"),
					Procedure("proc"),
					ClientStreamActions(
						SendStreamDecodeErrorAndExpectError(errors.New("nooooo"), "nooooo", "unknown"),
					),
				),
			),
		},
		{
			name: "stream invalid client decode yarpcerr",
			services: Lifecycles(
				GRPCService(
					Name("myservice"),
					p.NamedPort("9"),
				),
			),
			requests: Actions(
				GRPCStreamRequest(
					p.NamedPort("9"),
					Service("myservice"),
					Procedure("proc"),
					ClientStreamActions(
						SendStreamDecodeErrorAndExpectError(yarpcerrors.InternalErrorf("test"), yarpcerrors.InternalErrorf("test").Error()),
					),
				),
			),
		},
		{
			// The direct chooser is rather unique in that it releases the peer in
			// the onFinish function. Other choosers reuse the peer across calls and
			// only release it as part of chooser.Stop(). This case ensures we don't
			// call onFinish prematurely.
			name: "single use chooser",
			services: Lifecycles(
				GRPCService(
					Name("myservice"),
					p.NamedPort("10"),
					Proc(
						Name("proc"),
						EchoStreamHandler(),
					),
				),
			),
			requests: ConcurrentAction(
				RepeatAction(
					GRPCStreamRequest(
						p.NamedPort("10"),
						Service("myservice"),
						Procedure("proc"),
						ShardKey(fmt.Sprintf("127.0.0.1:%d", p.NamedPort("10").Port)),
						Chooser(newDirectChooser),
						ClientStreamActions(
							SendStreamMsg("test"),
							RecvStreamMsg("test"),
							SendStreamMsg("test2"),
							RecvStreamMsg("test2"),
							CloseStream(),
						),
					),
					10,
				),
				3,
			),
		},
		{
			name: "server invalid send read",
			services: Lifecycles(
				GRPCService(
					Name("myservice"),
					p.NamedPort("12"),
					Proc(
						Name("proc"),
						OrderedStreamHandler(
							SendStreamDecodeErrorAndExpectError(yarpcerrors.InternalErrorf("test"), yarpcerrors.InternalErrorf("test").Error()),
						),
					),
				),
			),
			requests: Actions(
				GRPCStreamRequest(
					p.NamedPort("12"),
					Service("myservice"),
					Procedure("proc"),
					ClientStreamActions(
						RecvStreamErr(io.EOF.Error()),
					),
				),
			),
		},

		{
			name: "stream with error details",
			services: Lifecycles(
				GRPCService(
					Name("myservice"),
					p.NamedPort("13"),
					Proc(
						Name("proc"),
						OrderedStreamHandler(
							SendStreamMsg("test1"),
							StreamHandlerError(protoerr),
						),
					),
				),
			),
			requests: Actions(
				GRPCStreamRequest(
					p.NamedPort("13"),
					Service("myservice"),
					Procedure("proc"),
					ClientStreamActions(
						RecvStreamMsg("test1"),
						RecvStreamErrInstance(yarpcerrors.FromError(protoerr)),
					),
				),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.services.Start(t))
			tt.requests.Run(t)
			require.NoError(t, tt.services.Stop(t))
		})
	}
}

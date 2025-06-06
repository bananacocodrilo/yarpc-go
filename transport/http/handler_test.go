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

package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpc "go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/routertest"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestHandlerSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	headers := make(http.Header)
	headers.Set(CallerHeader, "moe")
	headers.Set(EncodingHeader, "raw")
	headers.Set(TTLMSHeader, "1000")
	headers.Set(ProcedureHeader, "nyuck")
	headers.Set(ServiceHeader, "curly")
	headers.Set(ShardKeyHeader, "shard")
	headers.Set(RoutingKeyHeader, "routekey")
	headers.Set(RoutingDelegateHeader, "routedelegate")
	headers.Set(CallerProcedureHeader, "callerprocedure")

	router := transporttest.NewMockRouter(mockCtrl)
	rpcHandler := transporttest.NewMockUnaryHandler(mockCtrl)
	spec := transport.NewUnaryHandlerSpec(rpcHandler)

	router.EXPECT().Choose(gomock.Any(), routertest.NewMatcher().
		WithService("curly").
		WithProcedure("nyuck"),
	).Return(spec, nil)

	rpcHandler.EXPECT().Handle(
		transporttest.NewContextMatcher(t,
			transporttest.ContextTTL(time.Second),
		),
		transporttest.NewRequestMatcher(
			t, &transport.Request{
				Caller:          "moe",
				Service:         "curly",
				Transport:       "http",
				Encoding:        raw.Encoding,
				Procedure:       "nyuck",
				ShardKey:        "shard",
				RoutingKey:      "routekey",
				RoutingDelegate: "routedelegate",
				CallerProcedure: "callerprocedure",
				Body:            strings.NewReader("Nyuck Nyuck"),
			},
		),
		gomock.Any(),
	).Return(nil)

	httpHandler := handler{
		router:            router,
		tracer:            &opentracing.NoopTracer{},
		bothResponseError: true,
		transport:         NewTransport(),
	}
	req := &http.Request{
		Method: "POST",
		Header: headers,
		Body:   io.NopCloser(strings.NewReader("Nyuck Nyuck")),
	}
	rw := httptest.NewRecorder()
	httpHandler.ServeHTTP(rw, req)
	code := rw.Code
	assert.Equal(t, code, 200, "expected 200 code")
	assert.Equal(t, rw.Body.String(), "")
}

func TestHandlerHeaders(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		giveEncoding          string
		giveRequestProtoMajor int
		giveHeaders           http.Header
		grabHeaders           map[string]struct{}

		wantTTL       time.Duration
		wantTransport string
		wantHeaders   map[string]string
	}{
		{
			giveEncoding:          "json",
			giveRequestProtoMajor: 1,
			giveHeaders: http.Header{
				TTLMSHeader:      {"1000"},
				"Rpc-Header-Foo": {"bar"},
				"X-Baz":          {"bat"},
			},
			grabHeaders:   map[string]struct{}{"x-baz": {}},
			wantTTL:       time.Second,
			wantTransport: TransportName,
			wantHeaders: map[string]string{
				"foo":   "bar",
				"x-baz": "bat",
			},
		},
		{
			giveEncoding:          "json",
			giveRequestProtoMajor: 2,
			giveHeaders: http.Header{
				TTLMSHeader:      {"1000"},
				"Rpc-Header-Foo": {"bar"},
				"X-Baz":          {"bat"},
			},
			grabHeaders:   map[string]struct{}{"x-baz": {}},
			wantTTL:       time.Second,
			wantTransport: TransportHTTP2Name,
			wantHeaders: map[string]string{
				"foo":   "bar",
				"x-baz": "bat",
			},
		},
		{
			giveEncoding: "raw",
			giveHeaders: http.Header{
				TTLMSHeader: {"100"},
				"Rpc-Foo":   {"ignored"},
			},
			wantTTL:       100 * time.Millisecond,
			wantTransport: TransportName,
			wantHeaders:   map[string]string{},
		},
		{
			giveEncoding: "thrift",
			giveHeaders: http.Header{
				TTLMSHeader: {"1000"},
			},
			wantTTL:       time.Second,
			wantTransport: TransportName,
			wantHeaders:   map[string]string{},
		},
		{
			giveEncoding: "proto",
			giveHeaders: http.Header{
				TTLMSHeader: {"1000"},
			},
			wantTTL:       time.Second,
			wantTransport: TransportName,
			wantHeaders:   map[string]string{},
		},
	}

	for _, tt := range tests {
		router := transporttest.NewMockRouter(mockCtrl)
		rpcHandler := transporttest.NewMockUnaryHandler(mockCtrl)
		spec := transport.NewUnaryHandlerSpec(rpcHandler)

		router.EXPECT().Choose(gomock.Any(), routertest.NewMatcher().
			WithService("service").
			WithProcedure("hello"),
		).Return(spec, nil)

		httpHandler := handler{
			router:            router,
			tracer:            &opentracing.NoopTracer{},
			grabHeaders:       tt.grabHeaders,
			bothResponseError: true,
			transport:         NewTransport(),
		}

		rpcHandler.EXPECT().Handle(
			transporttest.NewContextMatcher(t,
				transporttest.ContextTTL(tt.wantTTL),
			),
			transporttest.NewRequestMatcher(t,
				&transport.Request{
					Caller:    "caller",
					Service:   "service",
					Transport: tt.wantTransport,
					Encoding:  transport.Encoding(tt.giveEncoding),
					Procedure: "hello",
					Headers:   transport.HeadersFromMap(tt.wantHeaders),
					Body:      strings.NewReader("world"),
				}),
			gomock.Any(),
		).Return(nil)

		headers := http.Header{}
		for k, vs := range tt.giveHeaders {
			for _, v := range vs {
				headers.Add(k, v)
			}
		}
		headers.Set(CallerHeader, "caller")
		headers.Set(ServiceHeader, "service")
		headers.Set(EncodingHeader, tt.giveEncoding)
		headers.Set(ProcedureHeader, "hello")

		req := &http.Request{
			Method:     "POST",
			Header:     headers,
			Body:       io.NopCloser(strings.NewReader("world")),
			ProtoMajor: tt.giveRequestProtoMajor,
		}
		rw := httptest.NewRecorder()
		httpHandler.ServeHTTP(rw, req)
		assert.Equal(t, 200, rw.Code, "expected 200 status code")
		assert.Equal(t, getContentType(transport.Encoding(tt.giveEncoding)), rw.Header().Get("Content-Type"))
	}
}

func TestHandlerFailures(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	service, procedure := "fake", "hello"

	baseHeaders := make(http.Header)
	baseHeaders.Set(CallerHeader, "somecaller")
	baseHeaders.Set(EncodingHeader, "raw")
	baseHeaders.Set(TTLMSHeader, "1000")
	baseHeaders.Set(ProcedureHeader, procedure)
	baseHeaders.Set(ServiceHeader, service)

	headersWithBadTTL := headerCopyWithout(baseHeaders, TTLMSHeader)
	headersWithBadTTL.Set(TTLMSHeader, "not a number")

	tests := []struct {
		req *http.Request

		// if we expect an error as a result of the TTL
		errTTL   bool
		wantCode yarpcerrors.Code
	}{
		{
			req:      &http.Request{Method: "GET"},
			wantCode: yarpcerrors.CodeNotFound,
		},
		{
			req: &http.Request{
				Method: "POST",
				Header: headerCopyWithout(baseHeaders, CallerHeader),
			},
			wantCode: yarpcerrors.CodeInvalidArgument,
		},
		{
			req: &http.Request{
				Method: "POST",
				Header: headerCopyWithout(baseHeaders, ServiceHeader),
			},
			wantCode: yarpcerrors.CodeInvalidArgument,
		},
		{
			req: &http.Request{
				Method: "POST",
				Header: headerCopyWithout(baseHeaders, ProcedureHeader),
			},
			wantCode: yarpcerrors.CodeInvalidArgument,
		},
		{
			req: &http.Request{
				Method: "POST",
				Header: headerCopyWithout(baseHeaders, TTLMSHeader),
			},
			wantCode: yarpcerrors.CodeInvalidArgument,
			errTTL:   true,
		},
		{
			req: &http.Request{
				Method: "POST",
			},
			wantCode: yarpcerrors.CodeInvalidArgument,
		},
		{
			req: &http.Request{
				Method: "POST",
				Header: headersWithBadTTL,
			},
			wantCode: yarpcerrors.CodeInvalidArgument,
			errTTL:   true,
		},
	}

	for _, tt := range tests {
		req := tt.req
		if req.Body == nil {
			req.Body = io.NopCloser(bytes.NewReader(nil))
		}

		reg := transporttest.NewMockRouter(mockCtrl)

		if tt.errTTL {
			// since TTL is checked after we've determined the transport type, if we have an
			// error with TTL it will be discovered after we read from the router
			spec := transport.NewUnaryHandlerSpec(panickedHandler{})
			reg.EXPECT().Choose(gomock.Any(), routertest.NewMatcher().
				WithService(service).
				WithProcedure(procedure),
			).Return(spec, nil)
		}

		h := handler{router: reg, tracer: &opentracing.NoopTracer{}, bothResponseError: true}

		rw := httptest.NewRecorder()
		h.ServeHTTP(rw, tt.req)

		httpStatusCode := rw.Code
		assert.True(t, httpStatusCode >= 400 && httpStatusCode < 500, "expected 400 level code")
		code := statusCodeToBestCode(httpStatusCode)
		assert.Equal(t, tt.wantCode, code)
		assert.Equal(t, "text/plain; charset=utf8", rw.Header().Get("Content-Type"))
	}
}

func TestHandlerInternalFailure(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	headers := make(http.Header)
	headers.Set(CallerHeader, "somecaller")
	headers.Set(EncodingHeader, "raw")
	headers.Set(TTLMSHeader, "1000")
	headers.Set(ProcedureHeader, "hello")
	headers.Set(ServiceHeader, "fake")

	request := http.Request{
		Method: "POST",
		Header: headers,
		Body:   io.NopCloser(bytes.NewReader(nil)),
	}

	rpcHandler := transporttest.NewMockUnaryHandler(mockCtrl)
	rpcHandler.EXPECT().Handle(
		transporttest.NewContextMatcher(t, transporttest.ContextTTL(time.Second)),
		transporttest.NewRequestMatcher(
			t, &transport.Request{
				Caller:    "somecaller",
				Service:   "fake",
				Transport: "http",
				Encoding:  raw.Encoding,
				Procedure: "hello",
				Body:      bytes.NewReader(nil),
			},
		),
		gomock.Any(),
	).Return(fmt.Errorf("great sadness"))

	router := transporttest.NewMockRouter(mockCtrl)
	spec := transport.NewUnaryHandlerSpec(rpcHandler)

	router.EXPECT().Choose(gomock.Any(), routertest.NewMatcher().
		WithService("fake").
		WithProcedure("hello"),
	).Return(spec, nil)

	httpHandler := handler{
		router:            router,
		tracer:            &opentracing.NoopTracer{},
		bothResponseError: true,
		transport:         NewTransport(),
	}
	httpResponse := httptest.NewRecorder()
	httpHandler.ServeHTTP(httpResponse, &request)

	code := httpResponse.Code
	assert.True(t, code >= 500 && code < 600, "expected 500 level response")
	assert.Equal(t,
		`error for service "fake" and procedure "hello": great sadness`+"\n",
		httpResponse.Body.String())
}

type panickedHandler struct{}

func (th panickedHandler) Handle(context.Context, *transport.Request, transport.ResponseWriter) error {
	panic("oops I panicked!")
}

func TestHandlerPanic(t *testing.T) {
	httpTransport := NewTransport()
	inbound := httpTransport.NewInbound("localhost:0")
	serverDispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "yarpc-test",
		Inbounds: yarpc.Inbounds{inbound},
	})
	serverDispatcher.Register([]transport.Procedure{
		{
			Name:        "panic",
			HandlerSpec: transport.NewUnaryHandlerSpec(panickedHandler{}),
		},
	})

	require.NoError(t, serverDispatcher.Start())
	defer serverDispatcher.Stop()

	clientDispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-test-client",
		Outbounds: yarpc.Outbounds{
			"yarpc-test": {
				Unary: httpTransport.NewSingleOutbound(fmt.Sprintf("http://%s", inbound.Addr().String())),
			},
		},
	})
	require.NoError(t, clientDispatcher.Start())
	defer clientDispatcher.Stop()

	client := raw.New(clientDispatcher.ClientConfig("yarpc-test"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := client.Call(ctx, "panic", []byte{})

	assert.Equal(t, yarpcerrors.CodeUnknown, yarpcerrors.FromError(err).Code())
}

func headerCopyWithout(headers http.Header, names ...string) http.Header {
	newHeaders := make(http.Header)
	for k, vs := range headers {
		for _, v := range vs {
			newHeaders.Add(k, v)
		}
	}

	for _, k := range names {
		newHeaders.Del(k)
	}

	return newHeaders
}

func TestResponseWriter(t *testing.T) {
	const (
		appErrDetails = "thrift ex message"
		appErrName    = "thrift ex name"
	)
	appErrCode := yarpcerrors.CodeAborted

	recorder := httptest.NewRecorder()
	writer := newResponseWriter(recorder)

	headers := transport.HeadersFromMap(map[string]string{
		"foo":       "bar",
		"shard-key": "123",
	})
	writer.AddHeaders(headers)

	writer.SetApplicationErrorMeta(&transport.ApplicationErrorMeta{
		Details: appErrDetails,
		Name:    appErrName,
		Code:    &appErrCode,
	})

	_, err := writer.Write([]byte("hello"))
	require.NoError(t, err)
	writer.Close(http.StatusOK)

	assert.Equal(t, "bar", recorder.Header().Get("rpc-header-foo"))
	assert.Equal(t, "123", recorder.Header().Get("rpc-header-shard-key"))
	assert.Equal(t, "hello", recorder.Body.String())

	assert.Equal(t, appErrDetails, recorder.Header().Get(_applicationErrorDetailsHeader))
	assert.Equal(t, appErrName, recorder.Header().Get(_applicationErrorNameHeader))
	assert.Equal(t, strconv.Itoa(int(appErrCode)), recorder.Header().Get(_applicationErrorCodeHeader))
}

func TestTruncatedHeader(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		wantTruncate bool
	}{
		{
			name:  "no-op",
			value: "foo bar",
		},
		{
			name:  "max",
			value: strings.Repeat("a", _maxAppErrDetailsHeaderLen),
		},
		{
			name:         "truncate",
			value:        strings.Repeat("b", _maxAppErrDetailsHeaderLen*2),
			wantTruncate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateAppErrDetails(tt.value)

			if !tt.wantTruncate {
				assert.Equal(t, tt.value, got, "expected no-op")
				return
			}

			assert.True(t, strings.HasSuffix(got, _truncatedHeaderMessage), "unexpected truncate suffix")
			assert.Len(t, got, _maxAppErrDetailsHeaderLen, "did not truncate")
		})
	}
}

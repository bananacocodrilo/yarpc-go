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
	"io"
	"sync"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	// Encoding is the wire encoding name. We use "proto" to be wire-compatible
	// with existing protobuf clients and servers.
	Encoding transport.Encoding = "proto"

	// JSONEncoding is the JSON wire encoding name.
	JSONEncoding transport.Encoding = "json"
)

var _bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 0, 1024)
		return &buf
	},
}

func getBuffer() *[]byte {
	buf := _bufferPool.Get().(*[]byte)
	*buf = (*buf)[:0]
	return buf
}

func putBuffer(buf *[]byte) {
	_bufferPool.Put(buf)
}

// unmarshal reads the full body from reader and unmarshals into message,
// dispatching on the encoding.
func unmarshal(encoding transport.Encoding, reader io.Reader, message proto.Message) error {
	buf := getBuffer()
	defer putBuffer(buf)

	// Read full body into buffer.
	var err error
	*buf, err = io.ReadAll(reader)
	if err != nil {
		return err
	}
	body := *buf
	if len(body) == 0 {
		return nil
	}
	return unmarshalBytes(encoding, body, message)
}

func unmarshalBytes(encoding transport.Encoding, body []byte, message proto.Message) error {
	switch encoding {
	case Encoding:
		return proto.Unmarshal(body, message)
	case JSONEncoding:
		return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(body, message)
	default:
		return yarpcerrors.Newf(yarpcerrors.CodeInternal,
			"hyperpb: unexpected encoding %q", encoding)
	}
}

// marshal serializes a proto.Message, dispatching on the encoding.
func marshal(encoding transport.Encoding, message proto.Message) ([]byte, func(), error) {
	switch encoding {
	case Encoding:
		return marshalProto(message)
	case JSONEncoding:
		return marshalJSON(message)
	default:
		return nil, nil, yarpcerrors.Newf(yarpcerrors.CodeInternal,
			"hyperpb: unexpected encoding %q", encoding)
	}
}

func marshalProto(message proto.Message) ([]byte, func(), error) {
	buf := getBuffer()
	cleanup := func() { putBuffer(buf) }

	data, err := proto.MarshalOptions{}.MarshalAppend(*buf, message)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	*buf = data
	return data, cleanup, nil
}

func marshalJSON(message proto.Message) ([]byte, func(), error) {
	data, err := protojson.MarshalOptions{}.Marshal(message)
	if err != nil {
		return nil, nil, err
	}
	return data, func() {}, nil
}

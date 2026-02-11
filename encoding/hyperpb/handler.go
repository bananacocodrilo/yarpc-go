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
	"context"

	apiencoding "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/errors"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// UnaryHandler is the handler function signature for hyperpb unary methods.
//
// The request message is a *hyperpb.Message (read-only after unmarshal).
// The response must be any proto.Message implementation; use dynamicpb or
// a concrete generated type to build responses.
type UnaryHandler func(context.Context, proto.Message) (proto.Message, error)

// unaryHandlerWrapper adapts a UnaryHandler + TypeRegistry into a
// transport.UnaryHandler.
type unaryHandlerWrapper struct {
	handler        UnaryHandler
	requestMsgName protoreflect.FullName
	registry       *TypeRegistry
}

var _ transport.UnaryHandler = (*unaryHandlerWrapper)(nil)

func (h *unaryHandlerWrapper) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	// Validate encoding.
	if err := errors.ExpectEncodings(req, Encoding, JSONEncoding); err != nil {
		return err
	}

	// Set up inbound call metadata.
	ctx, call := apiencoding.NewInboundCall(ctx)
	if err := call.ReadFromRequest(req); err != nil {
		return err
	}

	// Allocate a hyperpb message for the request type.
	request, err := h.registry.NewMessage(h.requestMsgName)
	if err != nil {
		return yarpcerrors.Newf(yarpcerrors.CodeInternal,
			"hyperpb: failed to create request message: %v", err)
	}

	// Unmarshal the request body.
	if err := unmarshal(req.Encoding, req.Body, request); err != nil {
		return errors.RequestBodyDecodeError(req, err)
	}

	// Call the user's handler.
	response, appErr := h.handler(ctx, request)

	// Write response headers.
	if err := call.WriteToResponse(resw); err != nil {
		return err
	}

	// Marshal the response.
	if response != nil {
		responseData, cleanup, err := marshal(req.Encoding, response)
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			return errors.ResponseBodyEncodeError(req, err)
		}
		if _, err := resw.Write(responseData); err != nil {
			return err
		}
	}

	if appErr != nil {
		resw.SetApplicationError()
		return appErr
	}
	return nil
}

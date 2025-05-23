// @generated Code generated by thrift-gen. Do not modify.

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

// Package echo is generated code used to make or handle TChannel calls using Thrift.
package echo

import (
	"fmt"

	athrift "github.com/uber/tchannel-go/thirdparty/github.com/apache/thrift/lib/go/thrift"
	"github.com/uber/tchannel-go/thrift"
)

// Interfaces for the service and client for the services defined in the IDL.

// TChanEcho is the interface that defines the server handler and client interface.
type TChanEcho interface {
	Echo(ctx thrift.Context, ping *Ping) (*Pong, error)
}

// Implementation of a client and service handler.

type tchanEchoClient struct {
	thriftService string
	client        thrift.TChanClient
}

func NewTChanEchoInheritedClient(thriftService string, client thrift.TChanClient) *tchanEchoClient {
	return &tchanEchoClient{
		thriftService,
		client,
	}
}

// NewTChanEchoClient creates a client that can be used to make remote calls.
func NewTChanEchoClient(client thrift.TChanClient) TChanEcho {
	return NewTChanEchoInheritedClient("Echo", client)
}

func (c *tchanEchoClient) Echo(ctx thrift.Context, ping *Ping) (*Pong, error) {
	var resp EchoEchoResult
	args := EchoEchoArgs{
		Ping: ping,
	}
	success, err := c.client.Call(ctx, c.thriftService, "echo", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for echo")
		}
	}

	return resp.GetSuccess(), err
}

type tchanEchoServer struct {
	handler TChanEcho
}

// NewTChanEchoServer wraps a handler for TChanEcho so it can be
// registered with a thrift.Server.
func NewTChanEchoServer(handler TChanEcho) thrift.TChanServer {
	return &tchanEchoServer{
		handler,
	}
}

func (s *tchanEchoServer) Service() string {
	return "Echo"
}

func (s *tchanEchoServer) Methods() []string {
	return []string{
		"echo",
	}
}

func (s *tchanEchoServer) Handle(ctx thrift.Context, methodName string, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	switch methodName {
	case "echo":
		return s.handleEcho(ctx, protocol)

	default:
		return false, nil, fmt.Errorf("method %v not found in service %v", methodName, s.Service())
	}
}

func (s *tchanEchoServer) handleEcho(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req EchoEchoArgs
	var res EchoEchoResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.Echo(ctx, req.Ping)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = r
	}

	return err == nil, &res, nil
}

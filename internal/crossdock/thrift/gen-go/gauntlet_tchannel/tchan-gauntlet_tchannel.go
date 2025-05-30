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

// Package gauntlet_tchannel is generated code used to make or handle TChannel calls using Thrift.
package gauntlet_tchannel

import (
	"fmt"

	athrift "github.com/uber/tchannel-go/thirdparty/github.com/apache/thrift/lib/go/thrift"
	"github.com/uber/tchannel-go/thrift"
)

// Interfaces for the service and client for the services defined in the IDL.

// TChanSecondService is the interface that defines the server handler and client interface.
type TChanSecondService interface {
	BlahBlah(ctx thrift.Context) error
	SecondtestString(ctx thrift.Context, thing string) (string, error)
}

// TChanThriftTest is the interface that defines the server handler and client interface.
type TChanThriftTest interface {
	TestBinary(ctx thrift.Context, thing []byte) ([]byte, error)
	TestByte(ctx thrift.Context, thing int8) (int8, error)
	TestDouble(ctx thrift.Context, thing float64) (float64, error)
	TestEnum(ctx thrift.Context, thing Numberz) (Numberz, error)
	TestException(ctx thrift.Context, arg string) error
	TestI32(ctx thrift.Context, thing int32) (int32, error)
	TestI64(ctx thrift.Context, thing int64) (int64, error)
	TestInsanity(ctx thrift.Context, argument *Insanity) (map[UserId]map[Numberz]*Insanity, error)
	TestList(ctx thrift.Context, thing []int32) ([]int32, error)
	TestMap(ctx thrift.Context, thing map[int32]int32) (map[int32]int32, error)
	TestMapMap(ctx thrift.Context, hello int32) (map[int32]map[int32]int32, error)
	TestMulti(ctx thrift.Context, arg0 int8, arg1 int32, arg2 int64, arg3 map[int16]string, arg4 Numberz, arg5 UserId) (*Xtruct, error)
	TestMultiException(ctx thrift.Context, arg0 string, arg1 string) (*Xtruct, error)
	TestNest(ctx thrift.Context, thing *Xtruct2) (*Xtruct2, error)
	TestSet(ctx thrift.Context, thing map[int32]bool) (map[int32]bool, error)
	TestString(ctx thrift.Context, thing string) (string, error)
	TestStringMap(ctx thrift.Context, thing map[string]string) (map[string]string, error)
	TestStruct(ctx thrift.Context, thing *Xtruct) (*Xtruct, error)
	TestTypedef(ctx thrift.Context, thing UserId) (UserId, error)
	TestVoid(ctx thrift.Context) error
}

// Implementation of a client and service handler.

type tchanSecondServiceClient struct {
	thriftService string
	client        thrift.TChanClient
}

func NewTChanSecondServiceInheritedClient(thriftService string, client thrift.TChanClient) *tchanSecondServiceClient {
	return &tchanSecondServiceClient{
		thriftService,
		client,
	}
}

// NewTChanSecondServiceClient creates a client that can be used to make remote calls.
func NewTChanSecondServiceClient(client thrift.TChanClient) TChanSecondService {
	return NewTChanSecondServiceInheritedClient("SecondService", client)
}

func (c *tchanSecondServiceClient) BlahBlah(ctx thrift.Context) error {
	var resp SecondServiceBlahBlahResult
	args := SecondServiceBlahBlahArgs{}
	success, err := c.client.Call(ctx, c.thriftService, "blahBlah", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for blahBlah")
		}
	}

	return err
}

func (c *tchanSecondServiceClient) SecondtestString(ctx thrift.Context, thing string) (string, error) {
	var resp SecondServiceSecondtestStringResult
	args := SecondServiceSecondtestStringArgs{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "secondtestString", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for secondtestString")
		}
	}

	return resp.GetSuccess(), err
}

type tchanSecondServiceServer struct {
	handler TChanSecondService
}

// NewTChanSecondServiceServer wraps a handler for TChanSecondService so it can be
// registered with a thrift.Server.
func NewTChanSecondServiceServer(handler TChanSecondService) thrift.TChanServer {
	return &tchanSecondServiceServer{
		handler,
	}
}

func (s *tchanSecondServiceServer) Service() string {
	return "SecondService"
}

func (s *tchanSecondServiceServer) Methods() []string {
	return []string{
		"blahBlah",
		"secondtestString",
	}
}

func (s *tchanSecondServiceServer) Handle(ctx thrift.Context, methodName string, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	switch methodName {
	case "blahBlah":
		return s.handleBlahBlah(ctx, protocol)
	case "secondtestString":
		return s.handleSecondtestString(ctx, protocol)

	default:
		return false, nil, fmt.Errorf("method %v not found in service %v", methodName, s.Service())
	}
}

func (s *tchanSecondServiceServer) handleBlahBlah(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req SecondServiceBlahBlahArgs
	var res SecondServiceBlahBlahResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	err :=
		s.handler.BlahBlah(ctx)

	if err != nil {
		return false, nil, err
	} else {
	}

	return err == nil, &res, nil
}

func (s *tchanSecondServiceServer) handleSecondtestString(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req SecondServiceSecondtestStringArgs
	var res SecondServiceSecondtestStringResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.SecondtestString(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = &r
	}

	return err == nil, &res, nil
}

type tchanThriftTestClient struct {
	thriftService string
	client        thrift.TChanClient
}

func NewTChanThriftTestInheritedClient(thriftService string, client thrift.TChanClient) *tchanThriftTestClient {
	return &tchanThriftTestClient{
		thriftService,
		client,
	}
}

// NewTChanThriftTestClient creates a client that can be used to make remote calls.
func NewTChanThriftTestClient(client thrift.TChanClient) TChanThriftTest {
	return NewTChanThriftTestInheritedClient("ThriftTest", client)
}

func (c *tchanThriftTestClient) TestBinary(ctx thrift.Context, thing []byte) ([]byte, error) {
	var resp ThriftTestTestBinaryResult
	args := ThriftTestTestBinaryArgs{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testBinary", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testBinary")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestByte(ctx thrift.Context, thing int8) (int8, error) {
	var resp ThriftTestTestByteResult
	args := ThriftTestTestByteArgs{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testByte", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testByte")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestDouble(ctx thrift.Context, thing float64) (float64, error) {
	var resp ThriftTestTestDoubleResult
	args := ThriftTestTestDoubleArgs{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testDouble", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testDouble")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestEnum(ctx thrift.Context, thing Numberz) (Numberz, error) {
	var resp ThriftTestTestEnumResult
	args := ThriftTestTestEnumArgs{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testEnum", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testEnum")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestException(ctx thrift.Context, arg string) error {
	var resp ThriftTestTestExceptionResult
	args := ThriftTestTestExceptionArgs{
		Arg: arg,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testException", &args, &resp)
	if err == nil && !success {
		switch {
		case resp.Err1 != nil:
			err = resp.Err1
		default:
			err = fmt.Errorf("received no result or unknown exception for testException")
		}
	}

	return err
}

func (c *tchanThriftTestClient) TestI32(ctx thrift.Context, thing int32) (int32, error) {
	var resp ThriftTestTestI32Result
	args := ThriftTestTestI32Args{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testI32", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testI32")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestI64(ctx thrift.Context, thing int64) (int64, error) {
	var resp ThriftTestTestI64Result
	args := ThriftTestTestI64Args{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testI64", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testI64")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestInsanity(ctx thrift.Context, argument *Insanity) (map[UserId]map[Numberz]*Insanity, error) {
	var resp ThriftTestTestInsanityResult
	args := ThriftTestTestInsanityArgs{
		Argument: argument,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testInsanity", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testInsanity")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestList(ctx thrift.Context, thing []int32) ([]int32, error) {
	var resp ThriftTestTestListResult
	args := ThriftTestTestListArgs{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testList", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testList")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestMap(ctx thrift.Context, thing map[int32]int32) (map[int32]int32, error) {
	var resp ThriftTestTestMapResult
	args := ThriftTestTestMapArgs{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testMap", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testMap")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestMapMap(ctx thrift.Context, hello int32) (map[int32]map[int32]int32, error) {
	var resp ThriftTestTestMapMapResult
	args := ThriftTestTestMapMapArgs{
		Hello: hello,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testMapMap", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testMapMap")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestMulti(ctx thrift.Context, arg0 int8, arg1 int32, arg2 int64, arg3 map[int16]string, arg4 Numberz, arg5 UserId) (*Xtruct, error) {
	var resp ThriftTestTestMultiResult
	args := ThriftTestTestMultiArgs{
		Arg0: arg0,
		Arg1: arg1,
		Arg2: arg2,
		Arg3: arg3,
		Arg4: arg4,
		Arg5: arg5,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testMulti", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testMulti")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestMultiException(ctx thrift.Context, arg0 string, arg1 string) (*Xtruct, error) {
	var resp ThriftTestTestMultiExceptionResult
	args := ThriftTestTestMultiExceptionArgs{
		Arg0: arg0,
		Arg1: arg1,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testMultiException", &args, &resp)
	if err == nil && !success {
		switch {
		case resp.Err1 != nil:
			err = resp.Err1
		case resp.Err2 != nil:
			err = resp.Err2
		default:
			err = fmt.Errorf("received no result or unknown exception for testMultiException")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestNest(ctx thrift.Context, thing *Xtruct2) (*Xtruct2, error) {
	var resp ThriftTestTestNestResult
	args := ThriftTestTestNestArgs{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testNest", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testNest")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestSet(ctx thrift.Context, thing map[int32]bool) (map[int32]bool, error) {
	var resp ThriftTestTestSetResult
	args := ThriftTestTestSetArgs{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testSet", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testSet")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestString(ctx thrift.Context, thing string) (string, error) {
	var resp ThriftTestTestStringResult
	args := ThriftTestTestStringArgs{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testString", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testString")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestStringMap(ctx thrift.Context, thing map[string]string) (map[string]string, error) {
	var resp ThriftTestTestStringMapResult
	args := ThriftTestTestStringMapArgs{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testStringMap", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testStringMap")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestStruct(ctx thrift.Context, thing *Xtruct) (*Xtruct, error) {
	var resp ThriftTestTestStructResult
	args := ThriftTestTestStructArgs{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testStruct", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testStruct")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestTypedef(ctx thrift.Context, thing UserId) (UserId, error) {
	var resp ThriftTestTestTypedefResult
	args := ThriftTestTestTypedefArgs{
		Thing: thing,
	}
	success, err := c.client.Call(ctx, c.thriftService, "testTypedef", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testTypedef")
		}
	}

	return resp.GetSuccess(), err
}

func (c *tchanThriftTestClient) TestVoid(ctx thrift.Context) error {
	var resp ThriftTestTestVoidResult
	args := ThriftTestTestVoidArgs{}
	success, err := c.client.Call(ctx, c.thriftService, "testVoid", &args, &resp)
	if err == nil && !success {
		switch {
		default:
			err = fmt.Errorf("received no result or unknown exception for testVoid")
		}
	}

	return err
}

type tchanThriftTestServer struct {
	handler TChanThriftTest
}

// NewTChanThriftTestServer wraps a handler for TChanThriftTest so it can be
// registered with a thrift.Server.
func NewTChanThriftTestServer(handler TChanThriftTest) thrift.TChanServer {
	return &tchanThriftTestServer{
		handler,
	}
}

func (s *tchanThriftTestServer) Service() string {
	return "ThriftTest"
}

func (s *tchanThriftTestServer) Methods() []string {
	return []string{
		"testBinary",
		"testByte",
		"testDouble",
		"testEnum",
		"testException",
		"testI32",
		"testI64",
		"testInsanity",
		"testList",
		"testMap",
		"testMapMap",
		"testMulti",
		"testMultiException",
		"testNest",
		"testSet",
		"testString",
		"testStringMap",
		"testStruct",
		"testTypedef",
		"testVoid",
	}
}

func (s *tchanThriftTestServer) Handle(ctx thrift.Context, methodName string, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	switch methodName {
	case "testBinary":
		return s.handleTestBinary(ctx, protocol)
	case "testByte":
		return s.handleTestByte(ctx, protocol)
	case "testDouble":
		return s.handleTestDouble(ctx, protocol)
	case "testEnum":
		return s.handleTestEnum(ctx, protocol)
	case "testException":
		return s.handleTestException(ctx, protocol)
	case "testI32":
		return s.handleTestI32(ctx, protocol)
	case "testI64":
		return s.handleTestI64(ctx, protocol)
	case "testInsanity":
		return s.handleTestInsanity(ctx, protocol)
	case "testList":
		return s.handleTestList(ctx, protocol)
	case "testMap":
		return s.handleTestMap(ctx, protocol)
	case "testMapMap":
		return s.handleTestMapMap(ctx, protocol)
	case "testMulti":
		return s.handleTestMulti(ctx, protocol)
	case "testMultiException":
		return s.handleTestMultiException(ctx, protocol)
	case "testNest":
		return s.handleTestNest(ctx, protocol)
	case "testSet":
		return s.handleTestSet(ctx, protocol)
	case "testString":
		return s.handleTestString(ctx, protocol)
	case "testStringMap":
		return s.handleTestStringMap(ctx, protocol)
	case "testStruct":
		return s.handleTestStruct(ctx, protocol)
	case "testTypedef":
		return s.handleTestTypedef(ctx, protocol)
	case "testVoid":
		return s.handleTestVoid(ctx, protocol)

	default:
		return false, nil, fmt.Errorf("method %v not found in service %v", methodName, s.Service())
	}
}

func (s *tchanThriftTestServer) handleTestBinary(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestBinaryArgs
	var res ThriftTestTestBinaryResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestBinary(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestByte(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestByteArgs
	var res ThriftTestTestByteResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestByte(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = &r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestDouble(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestDoubleArgs
	var res ThriftTestTestDoubleResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestDouble(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = &r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestEnum(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestEnumArgs
	var res ThriftTestTestEnumResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestEnum(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = &r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestException(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestExceptionArgs
	var res ThriftTestTestExceptionResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	err :=
		s.handler.TestException(ctx, req.Arg)

	if err != nil {
		switch v := err.(type) {
		case *Xception:
			if v == nil {
				return false, nil, fmt.Errorf("Handler for err1 returned non-nil error type *Xception but nil value")
			}
			res.Err1 = v
		default:
			return false, nil, err
		}
	} else {
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestI32(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestI32Args
	var res ThriftTestTestI32Result

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestI32(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = &r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestI64(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestI64Args
	var res ThriftTestTestI64Result

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestI64(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = &r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestInsanity(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestInsanityArgs
	var res ThriftTestTestInsanityResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestInsanity(ctx, req.Argument)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestList(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestListArgs
	var res ThriftTestTestListResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestList(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestMap(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestMapArgs
	var res ThriftTestTestMapResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestMap(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestMapMap(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestMapMapArgs
	var res ThriftTestTestMapMapResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestMapMap(ctx, req.Hello)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestMulti(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestMultiArgs
	var res ThriftTestTestMultiResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestMulti(ctx, req.Arg0, req.Arg1, req.Arg2, req.Arg3, req.Arg4, req.Arg5)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestMultiException(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestMultiExceptionArgs
	var res ThriftTestTestMultiExceptionResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestMultiException(ctx, req.Arg0, req.Arg1)

	if err != nil {
		switch v := err.(type) {
		case *Xception:
			if v == nil {
				return false, nil, fmt.Errorf("Handler for err1 returned non-nil error type *Xception but nil value")
			}
			res.Err1 = v
		case *Xception2:
			if v == nil {
				return false, nil, fmt.Errorf("Handler for err2 returned non-nil error type *Xception2 but nil value")
			}
			res.Err2 = v
		default:
			return false, nil, err
		}
	} else {
		res.Success = r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestNest(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestNestArgs
	var res ThriftTestTestNestResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestNest(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestSet(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestSetArgs
	var res ThriftTestTestSetResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestSet(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestString(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestStringArgs
	var res ThriftTestTestStringResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestString(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = &r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestStringMap(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestStringMapArgs
	var res ThriftTestTestStringMapResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestStringMap(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestStruct(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestStructArgs
	var res ThriftTestTestStructResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestStruct(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestTypedef(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestTypedefArgs
	var res ThriftTestTestTypedefResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	r, err :=
		s.handler.TestTypedef(ctx, req.Thing)

	if err != nil {
		return false, nil, err
	} else {
		res.Success = &r
	}

	return err == nil, &res, nil
}

func (s *tchanThriftTestServer) handleTestVoid(ctx thrift.Context, protocol athrift.TProtocol) (bool, athrift.TStruct, error) {
	var req ThriftTestTestVoidArgs
	var res ThriftTestTestVoidResult

	if err := req.Read(protocol); err != nil {
		return false, nil, err
	}

	err :=
		s.handler.TestVoid(ctx)

	if err != nil {
		return false, nil, err
	} else {
	}

	return err == nil, &res, nil
}

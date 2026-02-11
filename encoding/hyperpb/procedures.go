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
	"fmt"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/procedure"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// BuildProceduresParams contains the parameters for BuildProcedures.
type BuildProceduresParams struct {
	// ServiceName is the fully-qualified protobuf service name,
	// e.g. "my.package.v1.MyService".
	ServiceName string

	// Registry is the TypeRegistry used to allocate request messages.
	// All request message types for the service must be registered before
	// calling BuildProcedures.
	Registry *TypeRegistry

	// UnaryHandlers maps method names to their handler functions.
	// The keys are the simple method name, e.g. "Echo", not the
	// fully-qualified name.
	UnaryHandlers map[string]UnaryHandler
}

// BuildProcedures builds transport.Procedures from a service descriptor and
// a map of handler functions. Each method gets two procedure registrations:
// one for "proto" encoding and one for "json" encoding.
//
// Procedure names follow YARPC convention: "ServiceName::MethodName".
func BuildProcedures(params BuildProceduresParams) []transport.Procedure {
	procedures := make([]transport.Procedure, 0, len(params.UnaryHandlers)*2)

	for methodName, handler := range params.UnaryHandlers {
		procName := procedure.ToName(params.ServiceName, methodName)

		wrapper := &unaryHandlerWrapper{
			handler:        handler,
			requestMsgName: protoreflect.FullName(fmt.Sprintf("%s.%sRequest", params.ServiceName, methodName)),
			registry:       params.Registry,
		}

		procedures = append(procedures,
			transport.Procedure{
				Name:        procName,
				HandlerSpec: transport.NewUnaryHandlerSpec(wrapper),
				Encoding:    Encoding,
			},
			transport.Procedure{
				Name:        procName,
				HandlerSpec: transport.NewUnaryHandlerSpec(wrapper),
				Encoding:    JSONEncoding,
			},
		)
	}

	return procedures
}

// BuildProceduresFromDescriptor builds transport.Procedures using a protobuf
// ServiceDescriptor to resolve request message types. This is the preferred
// way to register procedures when you have the service descriptor available.
//
// The handlers map keys must match the method names in the service descriptor.
func BuildProceduresFromDescriptor(
	sd protoreflect.ServiceDescriptor,
	registry *TypeRegistry,
	handlers map[string]UnaryHandler,
) ([]transport.Procedure, error) {
	serviceName := string(sd.FullName())
	methods := sd.Methods()

	procedures := make([]transport.Procedure, 0, len(handlers)*2)

	for methodName, handler := range handlers {
		// Look up the method in the service descriptor.
		md := methods.ByName(protoreflect.Name(methodName))
		if md == nil {
			return nil, fmt.Errorf("hyperpb: method %q not found in service %q", methodName, serviceName)
		}

		procName := procedure.ToName(serviceName, methodName)
		inputName := md.Input().FullName()

		wrapper := &unaryHandlerWrapper{
			handler:        handler,
			requestMsgName: inputName,
			registry:       registry,
		}

		procedures = append(procedures,
			transport.Procedure{
				Name:        procName,
				HandlerSpec: transport.NewUnaryHandlerSpec(wrapper),
				Encoding:    Encoding,
			},
			transport.Procedure{
				Name:        procName,
				HandlerSpec: transport.NewUnaryHandlerSpec(wrapper),
				Encoding:    JSONEncoding,
			},
		)
	}

	return procedures, nil
}

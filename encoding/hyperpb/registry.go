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
	"sync"

	lib "buf.build/go/hyperpb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TypeRegistry compiles and caches hyperpb MessageTypes from protobuf
// descriptors. It is safe for concurrent use.
//
// Use RegisterMessageDescriptor to pre-compile types, then NewMessage to
// allocate messages for parsing.
type TypeRegistry struct {
	mu    sync.RWMutex
	types map[protoreflect.FullName]*lib.MessageType
}

// NewTypeRegistry creates an empty TypeRegistry.
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types: make(map[protoreflect.FullName]*lib.MessageType),
	}
}

// RegisterMessageDescriptor compiles a message descriptor and caches the
// resulting MessageType. If the descriptor has already been registered, this
// is a no-op.
//
// This calls hyperpb.CompileMessageDescriptor, which is an expensive
// optimizing compiler step. It should be done at init or startup time.
func (r *TypeRegistry) RegisterMessageDescriptor(md protoreflect.MessageDescriptor, opts ...lib.CompileOption) {
	name := md.FullName()

	r.mu.RLock()
	_, ok := r.types[name]
	r.mu.RUnlock()
	if ok {
		return
	}

	compiled := lib.CompileMessageDescriptor(md, opts...)

	r.mu.Lock()
	defer r.mu.Unlock()
	// Double-check after acquiring write lock.
	if _, ok := r.types[name]; !ok {
		r.types[name] = compiled
	}
}

// RegisterServiceDescriptor registers all input and output message types for
// every method in the given service descriptor.
func (r *TypeRegistry) RegisterServiceDescriptor(sd protoreflect.ServiceDescriptor, opts ...lib.CompileOption) {
	methods := sd.Methods()
	for i := 0; i < methods.Len(); i++ {
		m := methods.Get(i)
		r.RegisterMessageDescriptor(m.Input(), opts...)
		r.RegisterMessageDescriptor(m.Output(), opts...)
	}
}

// NewMessage allocates a new hyperpb.Message for the given fully-qualified
// message name. The returned message implements proto.Message and can be
// used with proto.Unmarshal.
//
// Returns an error if the type has not been registered.
func (r *TypeRegistry) NewMessage(name protoreflect.FullName) (proto.Message, error) {
	r.mu.RLock()
	mt, ok := r.types[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("hyperpb: message type %q not registered", name)
	}
	return lib.NewMessage(mt), nil
}

// MessageType returns the compiled MessageType for the given name, or nil
// if not registered.
func (r *TypeRegistry) MessageType(name protoreflect.FullName) *lib.MessageType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.types[name]
}

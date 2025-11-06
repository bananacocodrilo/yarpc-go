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

package protobuf

import (
	"sync"

	"github.com/gogo/protobuf/jsonpb"
	"go.uber.org/yarpc/api/transport"
	"google.golang.org/grpc/mem"
)

// Codec defines the interface that custom codecs must implement.
// This matches YARPC's transport/grpc customCodec interface.
type Codec interface {
	// Marshal encodes a message to mem.BufferSlice
	Marshal(v any) (mem.BufferSlice, error)
	// Unmarshal decodes mem.BufferSlice to a message
	Unmarshal(data mem.BufferSlice, v any) error
	// Name returns the codec identifier
	Name() string
}

// Global codec registry for encoding-based codec overrides
var (
	codecRegistryMutex sync.RWMutex
	codecRegistry      = make(map[string]Codec)
)

// RegisterCodec registers a codec for gRPC transport
func RegisterCodecForEncoding(codec Codec) {
	codecRegistryMutex.Lock()
	defer codecRegistryMutex.Unlock()
	codecRegistry[codec.Name()] = codec
}

// CreateCustomCodec creates a custom codec with specific marshal/unmarshal behavior.
// This is a helper function to create codecs that can be registered.
func CreateCustomCodec(anyResolver jsonpb.AnyResolver) *codec {
	return newCodec(anyResolver)
}

// getCodecForEncoding returns the codec for an encoding, or nil if none registered.
func getCodecForEncoding(encoding transport.Encoding) Codec {
	codecRegistryMutex.RLock()
	defer codecRegistryMutex.RUnlock()

	if codec, exists := codecRegistry[string(encoding)]; exists {
		return codec
	}
	return nil
}

// GetCodecForEncoding is the public version of getCodecForEncoding for testing/examples
func GetCodecForEncoding(encoding transport.Encoding) Codec {
	return getCodecForEncoding(encoding)
}

// GetCodecNames returns the names of all registered codecs
func GetCodecNames() []transport.Encoding {
	codecRegistryMutex.RLock()
	defer codecRegistryMutex.RUnlock()
	names := make([]transport.Encoding, 2+len(codecRegistry))
	names[0] = Encoding
	names[1] = JSONEncoding
	for encoding := range codecRegistry {
		names = append(names, transport.Encoding(encoding))
	}
	return names
}

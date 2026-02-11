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

// Package hyperpb provides a YARPC encoding backed by buf.build/go/hyperpb,
// a high-performance dynamic Protobuf parser.
//
// This package is a standalone encoding that does NOT share handlers with
// encoding/protobuf (gogo) or encoding/protobuf/v2. It registers procedures
// with the same "proto" wire encoding, so it is wire-compatible with existing
// generated clients.
//
// hyperpb is best suited for:
//   - Gateway/proxy services that forward messages without deep inspection.
//   - Services with many IDL files where binary size is a concern.
//   - High-throughput pipelines where marshal/unmarshal is a bottleneck.
//
// Trade-offs:
//   - Handlers receive dynamic messages accessed via protobuf reflection,
//     not typed structs.
//   - Parsed hyperpb messages are read-only; responses must be constructed
//     using another proto.Message implementation (e.g., dynamicpb).
package hyperpb

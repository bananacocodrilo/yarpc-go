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

package yarpcerrors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFaultTypeFromErrorForClientErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "invalid argument",
			err:  InvalidArgumentErrorf("test"),
		},
		{
			name: "cancelled",
			err:  CancelledErrorf("test"),
		},
		{
			name: "not found",
			err:  NotFoundErrorf("test"),
		},
		{
			name: "already exists",
			err:  AlreadyExistsErrorf("test"),
		},
		{
			name: "permission denied",
			err:  PermissionDeniedErrorf("test"),
		},
		{
			name: "resource exhausted",
			err:  ResourceExhaustedErrorf("test"),
		},
		{
			name: "failed precondition",
			err:  FailedPreconditionErrorf("test"),
		},
		{
			name: "aborted",
			err:  AbortedErrorf("test"),
		},
		{
			name: "out of range",
			err:  OutOfRangeErrorf("test"),
		},
		{
			name: "unimplemented",
			err:  UnimplementedErrorf("test"),
		},
		{
			name: "unauthenticated",
			err:  UnauthenticatedErrorf("test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, ClientFault, GetFaultTypeFromError(tt.err))
		})
	}
}

func TestGetFaultTypeFromErrorForServerErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "unknown",
			err:  UnknownErrorf("test"),
		},
		{
			name: "deadline exceeded",
			err:  DeadlineExceededErrorf("test"),
		},
		{
			name: "internal",
			err:  InternalErrorf("test"),
		},
		{
			name: "unavailable",
			err:  UnavailableErrorf("test"),
		},
		{
			name: "data loss",
			err:  DataLossErrorf("test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, ServerFault, GetFaultTypeFromError(tt.err))
		})
	}
}

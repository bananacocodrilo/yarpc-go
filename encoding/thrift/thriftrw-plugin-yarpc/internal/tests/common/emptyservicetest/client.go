// Code generated by thriftrw-plugin-yarpc
// @generated

package emptyservicetest

import (
	"github.com/golang/mock/gomock"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/emptyserviceclient"
)

// MockClient implements a gomock-compatible mock client for service
// EmptyService.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *_MockClientRecorder
}

var _ emptyserviceclient.Interface = (*MockClient)(nil)

type _MockClientRecorder struct {
	mock *MockClient
}

// Build a new mock client for service EmptyService.
//
// 	mockCtrl := gomock.NewController(t)
// 	client := emptyservicetest.NewMockClient(mockCtrl)
//
// Use EXPECT() to set expectations on the mock.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &_MockClientRecorder{mock}
	return mock
}

// EXPECT returns an object that allows you to define an expectation on the
// EmptyService mock client.
func (m *MockClient) EXPECT() *_MockClientRecorder {
	return m.recorder
}

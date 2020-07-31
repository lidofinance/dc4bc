// Code generated by MockGen. DO NOT EDIT.
// Source: ./../client/state.go

// Package clientMocks is a generated GoMock package.
package clientMocks

import (
	client "github.com/depools/dc4bc/client"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockState is a mock of State interface
type MockState struct {
	ctrl     *gomock.Controller
	recorder *MockStateMockRecorder
}

// MockStateMockRecorder is the mock recorder for MockState
type MockStateMockRecorder struct {
	mock *MockState
}

// NewMockState creates a new mock instance
func NewMockState(ctrl *gomock.Controller) *MockState {
	mock := &MockState{ctrl: ctrl}
	mock.recorder = &MockStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockState) EXPECT() *MockStateMockRecorder {
	return m.recorder
}

// SaveOffset mocks base method
func (m *MockState) SaveOffset(arg0 uint64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveOffset", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveOffset indicates an expected call of SaveOffset
func (mr *MockStateMockRecorder) SaveOffset(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveOffset", reflect.TypeOf((*MockState)(nil).SaveOffset), arg0)
}

// LoadOffset mocks base method
func (m *MockState) LoadOffset() (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LoadOffset")
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LoadOffset indicates an expected call of LoadOffset
func (mr *MockStateMockRecorder) LoadOffset() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoadOffset", reflect.TypeOf((*MockState)(nil).LoadOffset))
}

// SaveFSM mocks base method
func (m *MockState) SaveFSM(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveFSM", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveFSM indicates an expected call of SaveFSM
func (mr *MockStateMockRecorder) SaveFSM(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveFSM", reflect.TypeOf((*MockState)(nil).SaveFSM), arg0)
}

// LoadFSM mocks base method
func (m *MockState) LoadFSM() (interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LoadFSM")
	ret0, _ := ret[0].(interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LoadFSM indicates an expected call of LoadFSM
func (mr *MockStateMockRecorder) LoadFSM() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoadFSM", reflect.TypeOf((*MockState)(nil).LoadFSM))
}

// PutOperation mocks base method
func (m *MockState) PutOperation(operation *client.Operation) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PutOperation", operation)
	ret0, _ := ret[0].(error)
	return ret0
}

// PutOperation indicates an expected call of PutOperation
func (mr *MockStateMockRecorder) PutOperation(operation interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PutOperation", reflect.TypeOf((*MockState)(nil).PutOperation), operation)
}

// DeleteOperation mocks base method
func (m *MockState) DeleteOperation(operationID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteOperation", operationID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteOperation indicates an expected call of DeleteOperation
func (mr *MockStateMockRecorder) DeleteOperation(operationID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteOperation", reflect.TypeOf((*MockState)(nil).DeleteOperation), operationID)
}

// GetOperations mocks base method
func (m *MockState) GetOperations() (map[string]*client.Operation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOperations")
	ret0, _ := ret[0].(map[string]*client.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOperations indicates an expected call of GetOperations
func (mr *MockStateMockRecorder) GetOperations() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOperations", reflect.TypeOf((*MockState)(nil).GetOperations))
}

// GetOperationByID mocks base method
func (m *MockState) GetOperationByID(operationID string) (*client.Operation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOperationByID", operationID)
	ret0, _ := ret[0].(*client.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOperationByID indicates an expected call of GetOperationByID
func (mr *MockStateMockRecorder) GetOperationByID(operationID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOperationByID", reflect.TypeOf((*MockState)(nil).GetOperationByID), operationID)
}

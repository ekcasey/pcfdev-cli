// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/pivotal-cf/pcfdev-cli/debug (interfaces: FS)

package mocks

import (
	gomock "github.com/golang/mock/gomock"
	io "io"
)

// Mock of FS interface
type MockFS struct {
	ctrl     *gomock.Controller
	recorder *_MockFSRecorder
}

// Recorder for MockFS (not exported)
type _MockFSRecorder struct {
	mock *MockFS
}

func NewMockFS(ctrl *gomock.Controller) *MockFS {
	mock := &MockFS{ctrl: ctrl}
	mock.recorder = &_MockFSRecorder{mock}
	return mock
}

func (_m *MockFS) EXPECT() *_MockFSRecorder {
	return _m.recorder
}

func (_m *MockFS) Compress(_param0 string, _param1 string, _param2 []string) error {
	ret := _m.ctrl.Call(_m, "Compress", _param0, _param1, _param2)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockFSRecorder) Compress(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Compress", arg0, arg1, arg2)
}

func (_m *MockFS) Read(_param0 string) ([]byte, error) {
	ret := _m.ctrl.Call(_m, "Read", _param0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockFSRecorder) Read(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Read", arg0)
}

func (_m *MockFS) TempDir() (string, error) {
	ret := _m.ctrl.Call(_m, "TempDir")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockFSRecorder) TempDir() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "TempDir")
}

func (_m *MockFS) Write(_param0 string, _param1 io.Reader, _param2 bool) error {
	ret := _m.ctrl.Call(_m, "Write", _param0, _param1, _param2)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockFSRecorder) Write(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Write", arg0, arg1, arg2)
}

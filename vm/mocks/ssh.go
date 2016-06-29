// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/pivotal-cf/pcfdev-cli/vm (interfaces: SSH)

package mocks

import (
	gomock "github.com/golang/mock/gomock"
	io "io"
	time "time"
)

// Mock of SSH interface
type MockSSH struct {
	ctrl     *gomock.Controller
	recorder *_MockSSHRecorder
}

// Recorder for MockSSH (not exported)
type _MockSSHRecorder struct {
	mock *MockSSH
}

func NewMockSSH(ctrl *gomock.Controller) *MockSSH {
	mock := &MockSSH{ctrl: ctrl}
	mock.recorder = &_MockSSHRecorder{mock}
	return mock
}

func (_m *MockSSH) EXPECT() *_MockSSHRecorder {
	return _m.recorder
}

func (_m *MockSSH) GenerateAddress() (string, string, error) {
	ret := _m.ctrl.Call(_m, "GenerateAddress")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (_mr *_MockSSHRecorder) GenerateAddress() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GenerateAddress")
}

func (_m *MockSSH) GetSSHOutput(_param0 string, _param1 string, _param2 string, _param3 time.Duration) (string, error) {
	ret := _m.ctrl.Call(_m, "GetSSHOutput", _param0, _param1, _param2, _param3)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockSSHRecorder) GetSSHOutput(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetSSHOutput", arg0, arg1, arg2, arg3)
}

func (_m *MockSSH) RunSSHCommand(_param0 string, _param1 string, _param2 time.Duration, _param3 io.Writer, _param4 io.Writer) error {
	ret := _m.ctrl.Call(_m, "RunSSHCommand", _param0, _param1, _param2, _param3, _param4)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockSSHRecorder) RunSSHCommand(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "RunSSHCommand", arg0, arg1, arg2, arg3, arg4)
}

func (_m *MockSSH) WaitForSSH(_param0 string, _param1 string, _param2 time.Duration) error {
	ret := _m.ctrl.Call(_m, "WaitForSSH", _param0, _param1, _param2)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockSSHRecorder) WaitForSSH(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "WaitForSSH", arg0, arg1, arg2)
}

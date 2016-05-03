// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/pivotal-cf/pcfdev-cli/vbox (interfaces: SSH)

package mocks

import (
	gomock "github.com/golang/mock/gomock"
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

func (_m *MockSSH) RunSSHCommand(_param0 string, _param1 string, _param2 time.Duration) ([]byte, error) {
	ret := _m.ctrl.Call(_m, "RunSSHCommand", _param0, _param1, _param2)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockSSHRecorder) RunSSHCommand(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "RunSSHCommand", arg0, arg1, arg2)
}

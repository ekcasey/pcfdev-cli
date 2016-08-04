// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/pivotal-cf/pcfdev-cli/vm (interfaces: VBox)

package mocks

import (
	gomock "github.com/golang/mock/gomock"
	config "github.com/pivotal-cf/pcfdev-cli/config"
)

// Mock of VBox interface
type MockVBox struct {
	ctrl     *gomock.Controller
	recorder *_MockVBoxRecorder
}

// Recorder for MockVBox (not exported)
type _MockVBoxRecorder struct {
	mock *MockVBox
}

func NewMockVBox(ctrl *gomock.Controller) *MockVBox {
	mock := &MockVBox{ctrl: ctrl}
	mock.recorder = &_MockVBoxRecorder{mock}
	return mock
}

func (_m *MockVBox) EXPECT() *_MockVBoxRecorder {
	return _m.recorder
}

func (_m *MockVBox) ImportVM(_param0 *config.VMConfig) error {
	ret := _m.ctrl.Call(_m, "ImportVM", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockVBoxRecorder) ImportVM(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "ImportVM", arg0)
}

func (_m *MockVBox) PowerOffVM(_param0 *config.VMConfig) error {
	ret := _m.ctrl.Call(_m, "PowerOffVM", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockVBoxRecorder) PowerOffVM(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "PowerOffVM", arg0)
}

func (_m *MockVBox) ResumeVM(_param0 *config.VMConfig) error {
	ret := _m.ctrl.Call(_m, "ResumeVM", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockVBoxRecorder) ResumeVM(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "ResumeVM", arg0)
}

func (_m *MockVBox) StartVM(_param0 *config.VMConfig) error {
	ret := _m.ctrl.Call(_m, "StartVM", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockVBoxRecorder) StartVM(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "StartVM", arg0)
}

func (_m *MockVBox) StopVM(_param0 *config.VMConfig) error {
	ret := _m.ctrl.Call(_m, "StopVM", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockVBoxRecorder) StopVM(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "StopVM", arg0)
}

func (_m *MockVBox) SuspendVM(_param0 *config.VMConfig) error {
	ret := _m.ctrl.Call(_m, "SuspendVM", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockVBoxRecorder) SuspendVM(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "SuspendVM", arg0)
}

func (_m *MockVBox) VMConfig(_param0 string) (*config.VMConfig, error) {
	ret := _m.ctrl.Call(_m, "VMConfig", _param0)
	ret0, _ := ret[0].(*config.VMConfig)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockVBoxRecorder) VMConfig(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "VMConfig", arg0)
}

func (_m *MockVBox) VMExists(_param0 string) (bool, error) {
	ret := _m.ctrl.Call(_m, "VMExists", _param0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockVBoxRecorder) VMExists(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "VMExists", arg0)
}

func (_m *MockVBox) VMState(_param0 string) (string, error) {
	ret := _m.ctrl.Call(_m, "VMState", _param0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockVBoxRecorder) VMState(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "VMState", arg0)
}

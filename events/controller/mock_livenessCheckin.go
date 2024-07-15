// Code generated by mockery v2.42.1. DO NOT EDIT.

package controller

import (
	context "context"

	registry "github.com/metal-toolbox/rivets/events/registry"
	mock "github.com/stretchr/testify/mock"
)

// MockLivenessCheckin is an autogenerated mock type for the LivenessCheckin type
type MockLivenessCheckin struct {
	mock.Mock
}

type MockLivenessCheckin_Expecter struct {
	mock *mock.Mock
}

func (_m *MockLivenessCheckin) EXPECT() *MockLivenessCheckin_Expecter {
	return &MockLivenessCheckin_Expecter{mock: &_m.Mock}
}

// ControllerID provides a mock function with given fields:
func (_m *MockLivenessCheckin) ControllerID() registry.ControllerID {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ControllerID")
	}

	var r0 registry.ControllerID
	if rf, ok := ret.Get(0).(func() registry.ControllerID); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(registry.ControllerID)
		}
	}

	return r0
}

// MockLivenessCheckin_ControllerID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ControllerID'
type MockLivenessCheckin_ControllerID_Call struct {
	*mock.Call
}

// ControllerID is a helper method to define mock.On call
func (_e *MockLivenessCheckin_Expecter) ControllerID() *MockLivenessCheckin_ControllerID_Call {
	return &MockLivenessCheckin_ControllerID_Call{Call: _e.mock.On("ControllerID")}
}

func (_c *MockLivenessCheckin_ControllerID_Call) Run(run func()) *MockLivenessCheckin_ControllerID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockLivenessCheckin_ControllerID_Call) Return(_a0 registry.ControllerID) *MockLivenessCheckin_ControllerID_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockLivenessCheckin_ControllerID_Call) RunAndReturn(run func() registry.ControllerID) *MockLivenessCheckin_ControllerID_Call {
	_c.Call.Return(run)
	return _c
}

// StartLivenessCheckin provides a mock function with given fields: ctx
func (_m *MockLivenessCheckin) StartLivenessCheckin(ctx context.Context) {
	_m.Called(ctx)
}

// MockLivenessCheckin_StartLivenessCheckin_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StartLivenessCheckin'
type MockLivenessCheckin_StartLivenessCheckin_Call struct {
	*mock.Call
}

// StartLivenessCheckin is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockLivenessCheckin_Expecter) StartLivenessCheckin(ctx interface{}) *MockLivenessCheckin_StartLivenessCheckin_Call {
	return &MockLivenessCheckin_StartLivenessCheckin_Call{Call: _e.mock.On("StartLivenessCheckin", ctx)}
}

func (_c *MockLivenessCheckin_StartLivenessCheckin_Call) Run(run func(ctx context.Context)) *MockLivenessCheckin_StartLivenessCheckin_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockLivenessCheckin_StartLivenessCheckin_Call) Return() *MockLivenessCheckin_StartLivenessCheckin_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockLivenessCheckin_StartLivenessCheckin_Call) RunAndReturn(run func(context.Context)) *MockLivenessCheckin_StartLivenessCheckin_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockLivenessCheckin creates a new instance of MockLivenessCheckin. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockLivenessCheckin(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockLivenessCheckin {
	mock := &MockLivenessCheckin{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
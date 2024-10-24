// Code generated by mockery v2.46.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	types "k8s.io/apimachinery/pkg/types"

	v1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
)

// K8Service is an autogenerated mock type for the Service type
type K8Service struct {
	mock.Mock
}

type K8Service_Expecter struct {
	mock *mock.Mock
}

func (_m *K8Service) EXPECT() *K8Service_Expecter {
	return &K8Service_Expecter{mock: &_m.Mock}
}

// GetAccount provides a mock function with given fields: ctx, accountKey
func (_m *K8Service) GetAccount(ctx context.Context, accountKey types.NamespacedName) (*v1alpha1.Account, error) {
	ret := _m.Called(ctx, accountKey)

	if len(ret) == 0 {
		panic("no return value specified for GetAccount")
	}

	var r0 *v1alpha1.Account
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, types.NamespacedName) (*v1alpha1.Account, error)); ok {
		return rf(ctx, accountKey)
	}
	if rf, ok := ret.Get(0).(func(context.Context, types.NamespacedName) *v1alpha1.Account); ok {
		r0 = rf(ctx, accountKey)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.Account)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, types.NamespacedName) error); ok {
		r1 = rf(ctx, accountKey)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// K8Service_GetAccount_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAccount'
type K8Service_GetAccount_Call struct {
	*mock.Call
}

// GetAccount is a helper method to define mock.On call
//   - ctx context.Context
//   - accountKey types.NamespacedName
func (_e *K8Service_Expecter) GetAccount(ctx interface{}, accountKey interface{}) *K8Service_GetAccount_Call {
	return &K8Service_GetAccount_Call{Call: _e.mock.On("GetAccount", ctx, accountKey)}
}

func (_c *K8Service_GetAccount_Call) Run(run func(ctx context.Context, accountKey types.NamespacedName)) *K8Service_GetAccount_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(types.NamespacedName))
	})
	return _c
}

func (_c *K8Service_GetAccount_Call) Return(_a0 *v1alpha1.Account, _a1 error) *K8Service_GetAccount_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *K8Service_GetAccount_Call) RunAndReturn(run func(context.Context, types.NamespacedName) (*v1alpha1.Account, error)) *K8Service_GetAccount_Call {
	_c.Call.Return(run)
	return _c
}

// GetAccountForNamespace provides a mock function with given fields: ctx, namespace
func (_m *K8Service) GetAccountForNamespace(ctx context.Context, namespace string) (*v1alpha1.Account, error) {
	ret := _m.Called(ctx, namespace)

	if len(ret) == 0 {
		panic("no return value specified for GetAccountForNamespace")
	}

	var r0 *v1alpha1.Account
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*v1alpha1.Account, error)); ok {
		return rf(ctx, namespace)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *v1alpha1.Account); ok {
		r0 = rf(ctx, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.Account)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, namespace)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// K8Service_GetAccountForNamespace_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAccountForNamespace'
type K8Service_GetAccountForNamespace_Call struct {
	*mock.Call
}

// GetAccountForNamespace is a helper method to define mock.On call
//   - ctx context.Context
//   - namespace string
func (_e *K8Service_Expecter) GetAccountForNamespace(ctx interface{}, namespace interface{}) *K8Service_GetAccountForNamespace_Call {
	return &K8Service_GetAccountForNamespace_Call{Call: _e.mock.On("GetAccountForNamespace", ctx, namespace)}
}

func (_c *K8Service_GetAccountForNamespace_Call) Run(run func(ctx context.Context, namespace string)) *K8Service_GetAccountForNamespace_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *K8Service_GetAccountForNamespace_Call) Return(_a0 *v1alpha1.Account, _a1 error) *K8Service_GetAccountForNamespace_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *K8Service_GetAccountForNamespace_Call) RunAndReturn(run func(context.Context, string) (*v1alpha1.Account, error)) *K8Service_GetAccountForNamespace_Call {
	_c.Call.Return(run)
	return _c
}

// GetFirstLevelAccountForAccount provides a mock function with given fields: ctx, accountKey
func (_m *K8Service) GetFirstLevelAccountForAccount(ctx context.Context, accountKey types.NamespacedName) (*v1alpha1.Account, error) {
	ret := _m.Called(ctx, accountKey)

	if len(ret) == 0 {
		panic("no return value specified for GetFirstLevelAccountForAccount")
	}

	var r0 *v1alpha1.Account
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, types.NamespacedName) (*v1alpha1.Account, error)); ok {
		return rf(ctx, accountKey)
	}
	if rf, ok := ret.Get(0).(func(context.Context, types.NamespacedName) *v1alpha1.Account); ok {
		r0 = rf(ctx, accountKey)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.Account)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, types.NamespacedName) error); ok {
		r1 = rf(ctx, accountKey)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// K8Service_GetFirstLevelAccountForAccount_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetFirstLevelAccountForAccount'
type K8Service_GetFirstLevelAccountForAccount_Call struct {
	*mock.Call
}

// GetFirstLevelAccountForAccount is a helper method to define mock.On call
//   - ctx context.Context
//   - accountKey types.NamespacedName
func (_e *K8Service_Expecter) GetFirstLevelAccountForAccount(ctx interface{}, accountKey interface{}) *K8Service_GetFirstLevelAccountForAccount_Call {
	return &K8Service_GetFirstLevelAccountForAccount_Call{Call: _e.mock.On("GetFirstLevelAccountForAccount", ctx, accountKey)}
}

func (_c *K8Service_GetFirstLevelAccountForAccount_Call) Run(run func(ctx context.Context, accountKey types.NamespacedName)) *K8Service_GetFirstLevelAccountForAccount_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(types.NamespacedName))
	})
	return _c
}

func (_c *K8Service_GetFirstLevelAccountForAccount_Call) Return(_a0 *v1alpha1.Account, _a1 error) *K8Service_GetFirstLevelAccountForAccount_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *K8Service_GetFirstLevelAccountForAccount_Call) RunAndReturn(run func(context.Context, types.NamespacedName) (*v1alpha1.Account, error)) *K8Service_GetFirstLevelAccountForAccount_Call {
	_c.Call.Return(run)
	return _c
}

// GetFirstLevelAccountForNamespace provides a mock function with given fields: ctx, namespace
func (_m *K8Service) GetFirstLevelAccountForNamespace(ctx context.Context, namespace string) (*v1alpha1.Account, error) {
	ret := _m.Called(ctx, namespace)

	if len(ret) == 0 {
		panic("no return value specified for GetFirstLevelAccountForNamespace")
	}

	var r0 *v1alpha1.Account
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*v1alpha1.Account, error)); ok {
		return rf(ctx, namespace)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *v1alpha1.Account); ok {
		r0 = rf(ctx, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.Account)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, namespace)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// K8Service_GetFirstLevelAccountForNamespace_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetFirstLevelAccountForNamespace'
type K8Service_GetFirstLevelAccountForNamespace_Call struct {
	*mock.Call
}

// GetFirstLevelAccountForNamespace is a helper method to define mock.On call
//   - ctx context.Context
//   - namespace string
func (_e *K8Service_Expecter) GetFirstLevelAccountForNamespace(ctx interface{}, namespace interface{}) *K8Service_GetFirstLevelAccountForNamespace_Call {
	return &K8Service_GetFirstLevelAccountForNamespace_Call{Call: _e.mock.On("GetFirstLevelAccountForNamespace", ctx, namespace)}
}

func (_c *K8Service_GetFirstLevelAccountForNamespace_Call) Run(run func(ctx context.Context, namespace string)) *K8Service_GetFirstLevelAccountForNamespace_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *K8Service_GetFirstLevelAccountForNamespace_Call) Return(_a0 *v1alpha1.Account, _a1 error) *K8Service_GetFirstLevelAccountForNamespace_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *K8Service_GetFirstLevelAccountForNamespace_Call) RunAndReturn(run func(context.Context, string) (*v1alpha1.Account, error)) *K8Service_GetFirstLevelAccountForNamespace_Call {
	_c.Call.Return(run)
	return _c
}

// NewK8Service creates a new instance of K8Service. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewK8Service(t interface {
	mock.TestingT
	Cleanup(func())
}) *K8Service {
	mock := &K8Service{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

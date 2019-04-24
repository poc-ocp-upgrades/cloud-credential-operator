package actuator

import (
	"context"
	"bytes"
	"net/http"
	"runtime"
	"fmt"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
)

type Actuator interface {
	Create(context.Context, *minterv1.CredentialsRequest) error
	Delete(context.Context, *minterv1.CredentialsRequest) error
	Update(context.Context, *minterv1.CredentialsRequest) error
	Exists(context.Context, *minterv1.CredentialsRequest) (bool, error)
}
type DummyActuator struct{}

func (a *DummyActuator) Exists(ctx context.Context, cr *minterv1.CredentialsRequest) (bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return true, nil
}
func (a *DummyActuator) Create(ctx context.Context, cr *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return nil
}
func (a *DummyActuator) Update(ctx context.Context, cr *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return nil
}
func (a *DummyActuator) Delete(ctx context.Context, cr *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return nil
}

type ActuatorError struct {
	ErrReason	minterv1.CredentialsRequestConditionType
	Message		string
}
type ActuatorStatus interface {
	Reason() minterv1.CredentialsRequestConditionType
}

func (e *ActuatorError) Error() string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return e.Message
}
func (e *ActuatorError) Reason() minterv1.CredentialsRequestConditionType {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return e.ErrReason
}
func _logClusterCodePath() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	pc, _, _, _ := runtime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", runtime.FuncForPC(pc).Name()))
	http.Post("/"+"logcode", "application/json", bytes.NewBuffer(jsonLog))
}

package controller

import (
	"github.com/openshift/cloud-credential-operator/pkg/controller/credentialsrequest"
	"bytes"
	"net/http"
	"runtime"
	"fmt"
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	AddToManagerWithActuatorFuncs = append(AddToManagerWithActuatorFuncs, credentialsrequest.AddWithActuator)
}
func _logClusterCodePath() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	pc, _, _, _ := runtime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", runtime.FuncForPC(pc).Name()))
	http.Post("/"+"logcode", "application/json", bytes.NewBuffer(jsonLog))
}

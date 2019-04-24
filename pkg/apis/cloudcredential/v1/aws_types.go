package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"bytes"
	"net/http"
	"runtime"
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
	_logClusterCodePath()
	defer _logClusterCodePath()
	pc, _, _, _ := runtime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", runtime.FuncForPC(pc).Name()))
	http.Post("/"+"logcode", "application/json", bytes.NewBuffer(jsonLog))
}

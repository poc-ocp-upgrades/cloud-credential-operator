package apis

import (
	"k8s.io/apimachinery/pkg/runtime"
)

var AddToSchemes runtime.SchemeBuilder

func AddToScheme(s *runtime.Scheme) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return AddToSchemes.AddToScheme(s)
}

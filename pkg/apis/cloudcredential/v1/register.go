package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

var (
	SchemeGroupVersion	= schema.GroupVersion{Group: "cloudcredential.openshift.io", Version: "v1"}
	SchemeBuilder		= &scheme.Builder{GroupVersion: SchemeGroupVersion}
	AddToScheme		= SchemeBuilder.AddToScheme
)

func Resource(resource string) schema.GroupResource {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

package azure

import (
	"github.com/openshift/cloud-credential-operator/pkg/controller/secretannotator"
	corev1 "k8s.io/api/core/v1"
)

type secret struct{ corev1.Secret }

func (s *secret) HasAnnotation() bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if s.ObjectMeta.Annotations == nil {
		return false
	}
	if _, ok := s.ObjectMeta.Annotations[secretannotator.AnnotationKey]; !ok {
		return false
	}
	return true
}
func (s *secret) Clone() *secret {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &secret{*s.Secret.DeepCopy()}
}

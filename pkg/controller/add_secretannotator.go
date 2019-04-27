package controller

import (
	"github.com/openshift/cloud-credential-operator/pkg/controller/secretannotator"
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	AddToManagerFuncs = append(AddToManagerFuncs, secretannotator.Add)
}

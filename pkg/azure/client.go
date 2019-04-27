package azure

import (
	"context"
	"fmt"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	"github.com/openshift/cloud-credential-operator/pkg/controller/credentialsrequest/actuator"
	"github.com/openshift/cloud-credential-operator/pkg/controller/secretannotator"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RootSecretNamespace	= "openshift-config"
	RootSecretName		= "azure-creds"
)

var RootSecretKey = client.ObjectKey{Name: RootSecretName, Namespace: RootSecretNamespace}

type clientWrapper struct{ client.Client }

func newClientWrapper(c client.Client) *clientWrapper {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &clientWrapper{Client: c}
}
func (cw *clientWrapper) RootSecret(ctx context.Context) (*secret, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	secret, err := cw.Secret(ctx, RootSecretKey)
	if err != nil {
		return nil, err
	}
	if !secret.HasAnnotation() {
		return nil, &actuator.ActuatorError{ErrReason: minterv1.CredentialsProvisionFailure, Message: fmt.Sprintf("cannot proceed without cloud cred secret annotation %+v", secret)}
	}
	return secret, nil
}
func (cw *clientWrapper) Secret(ctx context.Context, key client.ObjectKey) (*secret, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	s := secret{}
	if err := cw.Get(ctx, key, &s.Secret); err != nil {
		return nil, err
	}
	return &s, nil
}
func (cw *clientWrapper) Mode(ctx context.Context) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	rs, err := cw.RootSecret(ctx)
	if err != nil {
		return "", err
	}
	return rs.Annotations[secretannotator.AnnotationKey], nil
}

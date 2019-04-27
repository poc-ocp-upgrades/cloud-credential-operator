package azure

import (
	"context"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

type base struct{ client *clientWrapper }

func (a *base) Delete(context.Context, *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return nil
}
func (a *base) Exists(ctx context.Context, cr *minterv1.CredentialsRequest) (bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	req, err := newRequest(cr)
	if err != nil {
		return false, err
	}
	if req.AzureStatus.ServicePrincipalName == "" || req.AzureStatus.AppID == "" {
		return false, nil
	}
	existingSecret := &corev1.Secret{}
	err = a.client.Get(ctx, types.NamespacedName{Namespace: cr.Spec.SecretRef.Namespace, Name: cr.Spec.SecretRef.Name}, existingSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

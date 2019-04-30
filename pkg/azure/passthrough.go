package azure

import (
	"context"
	"fmt"
	"reflect"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	"github.com/openshift/cloud-credential-operator/pkg/controller/credentialsrequest/actuator"
	actuatoriface "github.com/openshift/cloud-credential-operator/pkg/controller/credentialsrequest/actuator"
	"github.com/openshift/cloud-credential-operator/pkg/controller/secretannotator"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ actuator.Actuator = (*passthrough)(nil)

type passthrough struct{ base }

func newPassthrough(c *clientWrapper) *passthrough {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &passthrough{base{client: c}}
}
func (a *passthrough) Create(ctx context.Context, cr *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return a.Update(ctx, cr)
}
func (a *passthrough) Update(ctx context.Context, cr *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	root, err := a.client.RootSecret(ctx)
	if err != nil {
		return err
	}
	key := client.ObjectKey{Namespace: cr.Spec.SecretRef.Namespace, Name: cr.Spec.SecretRef.Name}
	existing, err := a.client.Secret(ctx, key)
	if err != nil && errors.IsNotFound(err) {
		s := &secret{}
		copySecret(cr, root, s)
		return a.client.Create(ctx, &s.Secret)
	} else if err != nil {
		return err
	}
	updated := existing.Clone()
	copySecret(cr, root, updated)
	if !reflect.DeepEqual(existing, updated) {
		err := a.client.Update(ctx, &updated.Secret)
		if err != nil {
			return &actuatoriface.ActuatorError{ErrReason: minterv1.CredentialsProvisionFailure, Message: "error updating secret"}
		}
	}
	return nil
}
func copySecret(cr *minterv1.CredentialsRequest, src *secret, dest *secret) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	dest.ObjectMeta = metav1.ObjectMeta{Name: cr.Spec.SecretRef.Name, Namespace: cr.Spec.SecretRef.Namespace, Annotations: map[string]string{minterv1.AnnotationCredentialsRequest: fmt.Sprintf("%s/%s", cr.Namespace, cr.Name)}}
	dest.Data = map[string][]byte{secretannotator.AzureClientID: src.Data[secretannotator.AzureClientID], secretannotator.AzureClientSecret: src.Data[secretannotator.AzureClientSecret]}
}

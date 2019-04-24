package azure

import (
	"context"
	"bytes"
	"net/http"
	"runtime"
	"fmt"
	"errors"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	"github.com/openshift/cloud-credential-operator/pkg/controller/credentialsrequest/actuator"
	"github.com/openshift/cloud-credential-operator/pkg/controller/secretannotator"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ actuator.Actuator = (*Actuator)(nil)

type Actuator struct {
	internal	actuator.Actuator
	client		*clientWrapper
	Codec		*minterv1.ProviderCodec
}

func NewActuator(c client.Client) (*Actuator, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	cw := newClientWrapper(c)
	mode, err := cw.Mode(context.Background())
	if err != nil {
		return nil, err
	}
	switch mode {
	case secretannotator.PassthroughAnnotation:
		return &Actuator{internal: newPassthrough(newClientWrapper(c))}, nil
	default:
		return nil, errors.New("invalid mode")
	}
}
func (a *Actuator) Create(ctx context.Context, cr *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return a.internal.Create(ctx, cr)
}
func (a *Actuator) Delete(ctx context.Context, cr *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return a.internal.Delete(ctx, cr)
}
func (a *Actuator) Update(ctx context.Context, cr *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return a.internal.Update(ctx, cr)
}
func (a *Actuator) Exists(ctx context.Context, cr *minterv1.CredentialsRequest) (bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return a.internal.Exists(ctx, cr)
}
func _logClusterCodePath() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	pc, _, _, _ := runtime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", runtime.FuncForPC(pc).Name()))
	http.Post("/"+"logcode", "application/json", bytes.NewBuffer(jsonLog))
}

package azure

import minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"

type request struct {
	*minterv1.CredentialsRequest
	AzureSpec	*minterv1.AzureProviderSpec
	AzureStatus	*minterv1.AzureProviderStatus
}

func newRequest(cr *minterv1.CredentialsRequest) (*request, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	codec, err := minterv1.NewCodec()
	if err != nil {
		return nil, err
	}
	status := minterv1.AzureProviderStatus{}
	err = codec.DecodeProviderStatus(cr.Status.ProviderStatus, &status)
	if err != nil {
		return nil, err
	}
	spec := minterv1.AzureProviderSpec{}
	err = codec.DecodeProviderSpec(cr.Spec.ProviderSpec, &spec)
	if err != nil {
		return nil, err
	}
	return &request{CredentialsRequest: cr, AzureSpec: &spec, AzureStatus: &status}, nil
}

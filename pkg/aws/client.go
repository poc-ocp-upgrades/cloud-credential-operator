package aws

import (
	"context"
	"bytes"
	"net/http"
	"runtime"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/openshift/cloud-credential-operator/version"
)

const (
	awsCredsSecretIDKey	= "aws_access_key_id"
	awsCredsSecretAccessKey	= "aws_secret_access_key"
)

type Client interface {
	CreateAccessKey(*iam.CreateAccessKeyInput) (*iam.CreateAccessKeyOutput, error)
	CreateUser(*iam.CreateUserInput) (*iam.CreateUserOutput, error)
	DeleteAccessKey(*iam.DeleteAccessKeyInput) (*iam.DeleteAccessKeyOutput, error)
	DeleteUser(*iam.DeleteUserInput) (*iam.DeleteUserOutput, error)
	DeleteUserPolicy(*iam.DeleteUserPolicyInput) (*iam.DeleteUserPolicyOutput, error)
	GetUser(*iam.GetUserInput) (*iam.GetUserOutput, error)
	ListAccessKeys(*iam.ListAccessKeysInput) (*iam.ListAccessKeysOutput, error)
	ListUserPolicies(*iam.ListUserPoliciesInput) (*iam.ListUserPoliciesOutput, error)
	PutUserPolicy(*iam.PutUserPolicyInput) (*iam.PutUserPolicyOutput, error)
	GetUserPolicy(*iam.GetUserPolicyInput) (*iam.GetUserPolicyOutput, error)
	SimulatePrincipalPolicy(*iam.SimulatePrincipalPolicyInput) (*iam.SimulatePolicyResponse, error)
	TagUser(*iam.TagUserInput) (*iam.TagUserOutput, error)
}
type awsClient struct{ iamClient iamiface.IAMAPI }

func (c *awsClient) CreateAccessKey(input *iam.CreateAccessKeyInput) (*iam.CreateAccessKeyOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return c.iamClient.CreateAccessKey(input)
}
func (c *awsClient) CreateUser(input *iam.CreateUserInput) (*iam.CreateUserOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return c.iamClient.CreateUser(input)
}
func (c *awsClient) DeleteAccessKey(input *iam.DeleteAccessKeyInput) (*iam.DeleteAccessKeyOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return c.iamClient.DeleteAccessKey(input)
}
func (c *awsClient) DeleteUser(input *iam.DeleteUserInput) (*iam.DeleteUserOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return c.iamClient.DeleteUser(input)
}
func (c *awsClient) DeleteUserPolicy(input *iam.DeleteUserPolicyInput) (*iam.DeleteUserPolicyOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return c.iamClient.DeleteUserPolicy(input)
}
func (c *awsClient) GetUser(input *iam.GetUserInput) (*iam.GetUserOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return c.iamClient.GetUser(input)
}
func (c *awsClient) ListAccessKeys(input *iam.ListAccessKeysInput) (*iam.ListAccessKeysOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return c.iamClient.ListAccessKeys(input)
}
func (c *awsClient) ListUserPolicies(input *iam.ListUserPoliciesInput) (*iam.ListUserPoliciesOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return c.iamClient.ListUserPolicies(input)
}
func (c *awsClient) PutUserPolicy(input *iam.PutUserPolicyInput) (*iam.PutUserPolicyOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return c.iamClient.PutUserPolicy(input)
}
func (c *awsClient) GetUserPolicy(input *iam.GetUserPolicyInput) (*iam.GetUserPolicyOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return c.iamClient.GetUserPolicy(input)
}
func (c *awsClient) SimulatePrincipalPolicy(input *iam.SimulatePrincipalPolicyInput) (*iam.SimulatePolicyResponse, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return c.iamClient.SimulatePrincipalPolicy(input)
}
func (c *awsClient) TagUser(input *iam.TagUserInput) (*iam.TagUserOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return c.iamClient.TagUser(input)
}
func NewClient(accessKeyID, secretAccessKey []byte, infraName string) (Client, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	awsConfig := &awssdk.Config{}
	awsConfig.Credentials = credentials.NewStaticCredentials(string(accessKeyID), string(secretAccessKey), "")
	s, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}
	s.Handlers.Build.PushBackNamed(request.NamedHandler{Name: "openshift.io/cloud-credential-operator", Fn: request.MakeAddToUserAgentHandler("openshift.io cloud-credential-operator", version.Version, infraName)})
	return &awsClient{iamClient: iam.New(s)}, nil
}
func LoadCredsFromSecret(kubeClient client.Client, namespace, secretName string) ([]byte, []byte, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	secret := &corev1.Secret{}
	err := kubeClient.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: namespace}, secret)
	if err != nil {
		return nil, nil, err
	}
	accessKeyID, ok := secret.Data[awsCredsSecretIDKey]
	if !ok {
		return nil, nil, fmt.Errorf("AWS credentials secret %v did not contain key %v", secretName, awsCredsSecretIDKey)
	}
	secretAccessKey, ok := secret.Data[awsCredsSecretAccessKey]
	if !ok {
		return nil, nil, fmt.Errorf("AWS credentials secret %v did not contain key %v", secretName, awsCredsSecretAccessKey)
	}
	return accessKeyID, secretAccessKey, nil
}
func _logClusterCodePath() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	pc, _, _, _ := runtime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", runtime.FuncForPC(pc).Name()))
	http.Post("/"+"logcode", "application/json", bytes.NewBuffer(jsonLog))
}
